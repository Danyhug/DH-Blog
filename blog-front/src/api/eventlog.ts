import request from './axios'

/** 事件状态，与后端 eventlog/model.go 的常量一一对应 */
export type EventStatus = 'queued' | 'running' | 'success' | 'retrying' | 'failed'

/** 事件来源，与后端 eventlog/model.go 的常量一一对应 */
export type EventSource = 'task' | 'article' | 'webdav' | 'gateway'

/** 一条后台事件 */
export interface BackgroundEvent {
  id: number
  createdAt: string
  source: string
  /** 任务标识，如 AI_Gen_Tags / disk_sync / usage_sync */
  kind: string
  status: EventStatus
  /** 关联的业务记录（目前是文章 ID），0 表示无 */
  targetId: number
  title: string
  detail: string
  /** 第几次重试，0 表示首次执行 */
  attempt: number
}

export interface EventQuery {
  page?: number
  pageSize?: number
  source?: string
  kind?: string
  status?: string
}

/** 分页拉取历史事件（首屏与翻页用） */
export const getBackgroundEvents = (
  params: EventQuery
): Promise<{ total: number; list: BackgroundEvent[] }> => {
  return request.get('/admin/events', { params })
}

/** 断线重连后补齐漏掉的事件，truncated 为真说明缺口比一次能取的还大 */
export const replayBackgroundEvents = (
  sinceId: number
): Promise<{ events: BackgroundEvent[]; cursor: number; truncated: boolean }> => {
  return request.get('/admin/events/since', { params: { sinceId } })
}

/** 服务器输出的一行，对应后端 eventlog/logstream.go 的 LogLine */
export interface ServerLogLine {
  /** 单调递增序号，用来和 WS 推来的行去重 */
  seq: number
  time: string
  level: string
  message: string
  fields?: Record<string, string>
}

/** 拉取控制台回滚内容（首屏用），返回值按时间正序 */
export const getServerLogs = (
  limit = 200
): Promise<{ lines: ServerLogLine[]; cursor: number }> => {
  return request.get('/admin/events/logs', { params: { limit } })
}
