import { ref, watch, onMounted, onUnmounted, type Ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'

// ---- TV 模式 ----
// 影院主要在 Android TV 上用：遥控器只有方向键、确认、返回，GPU 也远比电脑弱。
// TV 模式做两件事：关掉最费 GPU 的背景模糊与漂移动画（见 tailwind.css 的 tv 变体），
// 以及让播放器的方向键优先用于在控件间移动焦点。方向键导航本身不论是否 TV 模式都开着，
// 否则识别失败的电视连切换 TV 模式的按钮都按不到。

const STORAGE_KEY = 'dh-blog:theater-tv'
// Android TV / Google TV / Fire TV（AFT 开头的机型）/ 各家智能电视系统的 UA 特征
const TV_UA = /Android TV|GoogleTV|Google TV|\bAFT[A-Z]|BRAVIA|SmartTV|SMART-TV|Tizen|Web0S|webOS|HbbTV|\bTV\b|MiTV|MIBOX|AndroidTV/i

function readOverride(): boolean | null {
  // ?tv=1 / ?tv=0 显式开关并记住，便于在识别不出来的电视浏览器上手动打开
  const match = /[?&]tv=([01])\b/.exec(location.hash) ?? /[?&]tv=([01])\b/.exec(location.search)
  try {
    if (match) localStorage.setItem(STORAGE_KEY, match[1])
    const saved = localStorage.getItem(STORAGE_KEY)
    return saved === null ? null : saved === '1'
  } catch {
    return match ? match[1] === '1' : null
  }
}

export const tvMode = ref(readOverride() ?? TV_UA.test(navigator.userAgent))

export function setTvMode(value: boolean) {
  tvMode.value = value
  try {
    localStorage.setItem(STORAGE_KEY, value ? '1' : '0')
  } catch {
    // 记不住只影响下次打开
  }
}

// ---- 方向键空间导航 ----
// 遥控器的方向键在网页里就是 ArrowUp/Down/Left/Right。浏览器默认只会滚动页面，
// 这里按几何位置把焦点移到该方向上最近的可聚焦元素，确认键（Enter）交给元素自己的 click。

// tabindex="-1" 的元素（卡片里只给鼠标用的小按钮）不参与导航
const FOCUSABLE = ':is(button:not([disabled]), a[href], input:not([disabled]), select:not([disabled]), textarea:not([disabled]), [tabindex]):not([tabindex="-1"])'
type Direction = 'up' | 'down' | 'left' | 'right'
const DIRECTIONS: Record<string, Direction> = { ArrowUp: 'up', ArrowDown: 'down', ArrowLeft: 'left', ArrowRight: 'right' }

function isVisible(el: Element) {
  const rect = el.getBoundingClientRect()
  if (!rect.width || !rect.height) return false
  // 悬停才显示的按钮（opacity-0）遥控器看不见，不能让焦点停在上面；
  // 获得焦点时会自己显示出来的按钮标 data-nav-reveal，照常参与导航
  const check = (el as Element & { checkVisibility?: (options: object) => boolean }).checkVisibility
  return check ? check.call(el, { opacityProperty: !el.hasAttribute('data-nav-reveal'), visibilityProperty: true }) : true
}

/** 当前最上层的导航范围：打开的弹层（详情、播放器、正在播放）挂 data-nav-scope，焦点只在最上层里移动 */
function activeScope(): Element {
  const dialog = document.activeElement?.closest('.el-dialog, .el-overlay-dialog')
  if (dialog) return dialog
  const scopes = [...document.querySelectorAll('[data-nav-scope]')].filter(isVisible)
  return scopes[scopes.length - 1] ?? document.body
}

function candidates(scope: Element) {
  return [...scope.querySelectorAll<HTMLElement>(FOCUSABLE)].filter(el => !el.closest('[inert]') && isVisible(el))
}

function center(rect: DOMRect) {
  return { x: rect.left + rect.width / 2, y: rect.top + rect.height / 2 }
}

interface Candidate {
  el: HTMLElement
  /** 沿移动方向到对方近边的距离 */
  major: number
  /** 沿移动方向到对方远边的距离 */
  majorFar: number
  /** 两个中心在垂直方向上的偏差 */
  minor: number
  /** 是否落在当前元素沿移动方向延伸出的那条带子里 */
  inBeam: boolean
}

/**
 * 方向上最近的元素，规则照搬 Android 的 FocusFinder，电视用户对这套手感最熟：
 * 带子里的候选优先，除非带子外的候选沿移动方向明显更近；其余按 13×主轴² + 副轴² 比较。
 * 左右移动只在同一水平带里找：右边没有东西时停在原地，而不是斜着跳到下面几行去。
 */
function nearest(from: DOMRect, list: HTMLElement[], direction: Direction) {
  const origin = center(from)
  const horizontal = direction === 'left' || direction === 'right'
  const found: Candidate[] = []
  for (const el of list) {
    const rect = el.getBoundingClientRect()
    const point = center(rect)
    let major: number
    let majorFar: number
    switch (direction) {
      case 'right':
        if (point.x <= origin.x) continue
        major = rect.left - from.right
        majorFar = rect.right - from.right
        break
      case 'left':
        if (point.x >= origin.x) continue
        major = from.left - rect.right
        majorFar = from.left - rect.left
        break
      case 'down':
        if (point.y <= origin.y) continue
        major = rect.top - from.bottom
        majorFar = rect.bottom - from.bottom
        break
      case 'up':
        if (point.y >= origin.y) continue
        major = from.top - rect.bottom
        majorFar = from.top - rect.top
        break
    }
    const inBeam = horizontal
      ? rect.bottom > from.top && rect.top < from.bottom
      : rect.right > from.left && rect.left < from.right
    if (horizontal && !inBeam) continue
    found.push({
      el,
      major: Math.max(0, major),
      majorFar: Math.max(0, majorFar),
      minor: horizontal ? Math.abs(point.y - origin.y) : Math.abs(point.x - origin.x),
      inBeam
    })
  }
  const weight = (c: Candidate) => 13 * c.major * c.major + c.minor * c.minor
  const better = (a: Candidate, b: Candidate) => {
    if (a.inBeam !== b.inBeam) {
      const [inside, outside] = a.inBeam ? [a, b] : [b, a]
      // 带子里的候选只要不比带子外的候选整个都远，就算它赢
      if (inside.major < outside.majorFar) return a === inside
    }
    return weight(a) < weight(b)
  }
  return found.reduce<Candidate | null>((best, c) => (!best || better(c, best) ? c : best), null)?.el ?? null
}

export function focusElement(el: HTMLElement, direction?: Direction) {
  el.focus({ preventScroll: true })
  // 横向移动只需让行内滚到可见；纵向移动把新的一行滚到屏幕中间，避免被顶部悬浮导航挡住
  el.scrollIntoView({
    behavior: 'smooth',
    block: direction === 'left' || direction === 'right' ? 'nearest' : 'center',
    inline: 'nearest'
  })
}

/** 焦点还不在当前范围里时（刚打开页面或弹层），落到 data-nav-autofocus 或屏幕上第一个可聚焦元素 */
function focusFirst(scope: Element = activeScope()) {
  const preferred = scope.querySelector<HTMLElement>('[data-nav-autofocus]')
  const target = preferred && isVisible(preferred) ? preferred : candidates(scope).find(el => {
    const rect = el.getBoundingClientRect()
    return rect.bottom > 0 && rect.top < innerHeight
  })
  if (target) focusElement(target)
  return !!target
}

function onKeydown(event: KeyboardEvent) {
  const direction = DIRECTIONS[event.key]
  if (!direction || event.defaultPrevented || event.altKey || event.ctrlKey || event.metaKey || event.shiftKey) return
  const active = document.activeElement as HTMLElement | null
  if (active) {
    // 输入框里左右键用来移动光标；上下键才离开输入框
    if (active.matches('input, textarea, [contenteditable="true"]') && (direction === 'left' || direction === 'right')) return
    // 组件自己处理方向键的地方不抢：滑块用左右调值（上下照常离开），下拉列表四个方向都归它
    if (active.matches('[role="slider"]') && (direction === 'left' || direction === 'right')) return
    if (active.closest('[role="listbox"], .el-select-dropdown, .el-dropdown-menu')) return
  }

  const scope = activeScope()
  event.preventDefault()
  if (!active || active === document.body || !scope.contains(active)) {
    focusFirst(scope)
    return
  }
  const target = nearest(active.getBoundingClientRect(), candidates(scope).filter(el => el !== active), direction)
  if (target) focusElement(target, direction)
}

/** 在影院外壳上启用一次即可，整个影院（含弹层）共用 */
export function useSpatialNavigation() {
  onMounted(() => window.addEventListener('keydown', onKeydown))
  onUnmounted(() => window.removeEventListener('keydown', onKeydown))
}

// ---- 弹层与浏览器历史 ----
// 遥控器的返回键就是浏览器后退。详情、播放器这些弹层原本只是组件状态，按返回会直接退出影院；
// 打开时往历史里压一条（只在 query 上加个标记），返回键就先关掉最上层的弹层。

export function useHistoryLayer(key: string, isOpen: Ref<boolean>, close: () => void) {
  const route = useRoute()
  const router = useRouter()
  let pushed = false

  watch(isOpen, open => {
    const present = route.query[key] !== undefined
    if (open && !present) {
      pushed = true
      router.push({ query: { ...route.query, [key]: '1' } })
    } else if (!open && present) {
      // 界面上点了关闭：自己压的那条就退回去，深链接带进来的就直接去掉标记
      if (pushed) router.back()
      else router.replace({ query: { ...route.query, [key]: undefined } })
      pushed = false
    }
  })

  // 标记从地址里消失（按了返回键）而弹层还开着：关掉它
  watch(() => route.query[key], value => {
    if (value === undefined && isOpen.value) {
      pushed = false
      close()
    }
  })

  // 刷新页面后弹层状态已经没了，残留的标记一并清掉
  onMounted(() => {
    if (route.query[key] !== undefined && !isOpen.value) router.replace({ query: { ...route.query, [key]: undefined } })
  })
}

/** 弹层关闭后把焦点还给打开它的那张卡片，遥控器接着从原来的位置走，而不是跳回页面顶部 */
export function useFocusRestore() {
  let previous: HTMLElement | null = null
  onMounted(() => {
    previous = document.activeElement instanceof HTMLElement && document.activeElement !== document.body ? document.activeElement : null
  })
  onUnmounted(() => {
    if (previous?.isConnected) previous.focus({ preventScroll: true })
  })
}
