import { getServerUrl } from '@/api/axios'
import type { BackgroundEvent, ServerLogLine } from '@/api/eventlog'

/** 连接状态，用于页面上的状态指示 */
export type SocketState = 'connecting' | 'open' | 'closed'

/** 连接建立时服务端下发的游标，用来判断断线期间是否漏了事件 */
export interface HelloPayload {
  cursor: number
  logCursor: number
  clients: number
  serverTime: string
}

interface SocketHandlers {
  onEvent?: (event: BackgroundEvent) => void
  onLog?: (line: ServerLogLine) => void
  onHello?: (hello: HelloPayload) => void
  onState?: (state: SocketState) => void
}

/** 帧类型，与后端 eventlog/hub.go 的常量一一对应 */
const FRAME_HELLO = 'hello'
const FRAME_EVENT = 'event'
const FRAME_LOG = 'log'
const FRAME_PING = 'ping'

const MIN_RETRY_DELAY = 1000
const MAX_RETRY_DELAY = 30000
/** 应用层心跳间隔，比服务端 50s 的协议 ping 更密，用来更早发现半开连接 */
const HEARTBEAT_INTERVAL = 25000
/** 超过这个时间没收到任何帧就主动重连 */
const SILENCE_TIMEOUT = 60000

/**
 * 后台事件的 WebSocket 客户端。
 *
 * 负责重连与心跳，不负责去重和补齐 —— 那两件事需要页面自己的游标，
 * 由调用方在 onHello 里决定要不要走 /admin/events/since。
 */
export class EventSocket {
  private socket: WebSocket | null = null
  private handlers: SocketHandlers
  private retryDelay = MIN_RETRY_DELAY
  private retryTimer: ReturnType<typeof setTimeout> | null = null
  private heartbeatTimer: ReturnType<typeof setInterval> | null = null
  private lastFrameAt = 0
  /** 主动关闭时置位，避免 onclose 又把自己拉起来 */
  private disposed = false

  constructor(handlers: SocketHandlers) {
    this.handlers = handlers
  }

  open() {
    this.disposed = false
    this.connect()
  }

  close() {
    this.disposed = true
    this.clearTimers()
    // 先摘掉回调再关，否则 onclose 还会触发一次状态变更
    if (this.socket) {
      this.socket.onclose = null
      this.socket.onerror = null
      this.socket.onmessage = null
      this.socket.onopen = null
      this.socket.close()
      this.socket = null
    }
    this.handlers.onState?.('closed')
  }

  private buildUrl(): string {
    // 浏览器的 WebSocket 不能自定义请求头，token 只能走 query，
    // 后端 middleware/jwt.go 的 extractToken 支持这种取法
    const token = localStorage.getItem('token') || ''
    const url = new URL(`${getServerUrl()}/admin/events/ws`, window.location.href)
    url.protocol = url.protocol === 'https:' ? 'wss:' : 'ws:'
    url.searchParams.set('token', token)
    return url.toString()
  }

  private connect() {
    if (this.disposed) return
    this.handlers.onState?.('connecting')

    let socket: WebSocket
    try {
      socket = new WebSocket(this.buildUrl())
    } catch {
      this.scheduleRetry()
      return
    }
    this.socket = socket

    socket.onopen = () => {
      this.retryDelay = MIN_RETRY_DELAY
      this.lastFrameAt = Date.now()
      this.startHeartbeat()
      this.handlers.onState?.('open')
    }

    socket.onmessage = (raw) => {
      this.lastFrameAt = Date.now()
      let frame: { type?: string; payload?: unknown }
      try {
        frame = JSON.parse(raw.data)
      } catch {
        return
      }
      if (frame.type === FRAME_EVENT) {
        this.handlers.onEvent?.(frame.payload as BackgroundEvent)
      } else if (frame.type === FRAME_LOG) {
        this.handlers.onLog?.(frame.payload as ServerLogLine)
      } else if (frame.type === FRAME_HELLO) {
        this.handlers.onHello?.(frame.payload as HelloPayload)
      }
      // pong / error 帧只用来证明连接还活着，上面已经刷新过时间戳
    }

    socket.onerror = () => {
      // 真正的收尾统一放在 onclose，浏览器一定会跟着触发
    }

    socket.onclose = () => {
      this.clearTimers()
      this.socket = null
      if (this.disposed) return
      this.handlers.onState?.('connecting')
      this.scheduleRetry()
    }
  }

  private startHeartbeat() {
    this.stopHeartbeat()
    this.heartbeatTimer = setInterval(() => {
      if (this.socket?.readyState !== WebSocket.OPEN) return
      // 静默太久说明连接已经半开，主动断开让重连逻辑接手
      if (Date.now() - this.lastFrameAt > SILENCE_TIMEOUT) {
        this.socket.close()
        return
      }
      this.socket.send(JSON.stringify({ type: FRAME_PING }))
    }, HEARTBEAT_INTERVAL)
  }

  private stopHeartbeat() {
    if (this.heartbeatTimer !== null) {
      clearInterval(this.heartbeatTimer)
      this.heartbeatTimer = null
    }
  }

  private clearTimers() {
    this.stopHeartbeat()
    if (this.retryTimer !== null) {
      clearTimeout(this.retryTimer)
      this.retryTimer = null
    }
  }

  private scheduleRetry() {
    if (this.disposed || this.retryTimer !== null) return
    const delay = this.retryDelay
    this.retryTimer = setTimeout(() => {
      this.retryTimer = null
      this.connect()
    }, delay)
    // 指数退避，封顶 30s：token 失效时握手会一直失败，不能一直重试到把服务器打爆
    this.retryDelay = Math.min(this.retryDelay * 2, MAX_RETRY_DELAY)
  }
}
