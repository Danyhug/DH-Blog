// 截断落在标点上时，省略号会紧挨着它（，… / 。…）。标点不比省略号多传递信息，直接隐去。
// CSS line-clamp 画不出「省略号前一个字」，所以在同一元素上按行数测量，再写回已经截好的文本。
import type { Directive } from 'vue'

const ELLIPSIS = '\u2026'

// 只去掉会和省略号打架的句读（，。！？、；：以及省略号、破折号本身）。
// 括号、引号和 % 留着，吃掉会改意思。
const TRAILING_PUNCTUATION = new Set(Array.from('。！？，、；：．｡,.!?;:…⋯‥—–-~～·・'))

interface ClampBinding {
  text?: string
  lines: number
}

interface ClampState {
  text: string
  lines: number
  lastWidth: number
  frameId: number
  resizeObserver: ResizeObserver
}

const clampState = new WeakMap<HTMLElement, ClampState>()

function omitTrailingPunctuation(text: string): string {
  const characters = Array.from(text)
  while (characters.length > 0 && isTrailingPunctuation(characters[characters.length - 1])) {
    characters.pop()
  }
  return characters.join('')
}

function isTrailingPunctuation(character: string): boolean {
  return TRAILING_PUNCTUATION.has(character) || /\s/u.test(character)
}

function normalizeWhitespace(text: string): string {
  return text.replace(/[ \t\r\n\u3000]+/g, ' ').trim()
}

function readLineHeight(element: HTMLElement): number {
  const parsedLineHeight = parseFloat(getComputedStyle(element).lineHeight)
  if (Number.isFinite(parsedLineHeight) && parsedLineHeight > 0) return parsedLineHeight
  const fontSize = parseFloat(getComputedStyle(element).fontSize)
  return Number.isFinite(fontSize) && fontSize > 0 ? fontSize * 1.2 : 16
}

function exceedsLineLimit(element: HTMLElement, lineHeight: number, lines: number): boolean {
  // 新的一行大约再增高一个 lineHeight；1px 的取整误差不能当成溢出。
  return element.scrollHeight > lineHeight * lines + lineHeight / 2
}

function textFittingLines(characters: string[], length: number): string {
  const stripped = omitTrailingPunctuation(characters.slice(0, length).join(''))
  if (!stripped) return ELLIPSIS
  return `${stripped}${ELLIPSIS}`
}

// fits 收到的是即将展示的字符串（超长时已去掉末尾标点并带上省略号）。放得下的原文原样返回，句号不会被吃掉。
function clampToFit(text: string, fits: (candidate: string) => boolean): string {
  const fullText = normalizeWhitespace(text)
  if (!fullText || fits(fullText)) return fullText

  const characters = Array.from(fullText)
  let lowerBound = 0
  let upperBound = characters.length
  let longestFittingLength = 0

  while (lowerBound <= upperBound) {
    const middle = Math.floor((lowerBound + upperBound) / 2)
    if (fits(textFittingLines(characters, middle))) {
      longestFittingLength = middle
      lowerBound = middle + 1
    } else {
      upperBound = middle - 1
    }
  }

  return textFittingLines(characters, longestFittingLength)
}

function applyLineClamp(element: HTMLElement, text: string, lines: number) {
  const fullText = normalizeWhitespace(text)
  const lineCount = lines > 0 ? lines : 1
  if (!fullText || element.clientWidth === 0) {
    element.textContent = fullText
    return
  }

  const lineHeight = readLineHeight(element)
  element.textContent = clampToFit(fullText, candidate => {
    element.textContent = candidate
    return !exceedsLineLimit(element, lineHeight, lineCount)
  })
}

function readClampBinding(value: ClampBinding | null | undefined): { text: string, lines: number } {
  return {
    text: value?.text ?? '',
    lines: value?.lines && value.lines > 0 ? value.lines : 1,
  }
}

function renderClampedText(element: HTMLElement) {
  const state = clampState.get(element)
  if (!state) return
  applyLineClamp(element, state.text, state.lines)
}

export const vClamp: Directive<HTMLElement, ClampBinding> = {
  mounted(element, binding) {
    const resizeObserver = new ResizeObserver(entries => {
      const state = clampState.get(element)
      if (!state) return
      const width = entries[0]?.contentRect.width ?? 0
      if (width === state.lastWidth) return
      state.lastWidth = width
      renderClampedText(element)
    })
    clampState.set(element, {
      ...readClampBinding(binding.value),
      lastWidth: -1,
      frameId: 0,
      resizeObserver,
    })
    resizeObserver.observe(element)
    renderClampedText(element)
    // 挂载当帧宽度偶发为 0（过渡、父级尚未完成布局），下一帧再量一次。
    const state = clampState.get(element)
    if (!state) return
    state.frameId = requestAnimationFrame(() => renderClampedText(element))
  },
  updated(element, binding) {
    const state = clampState.get(element)
    if (!state) return
    const next = readClampBinding(binding.value)
    if (next.text === state.text && next.lines === state.lines) return
    state.text = next.text
    state.lines = next.lines
    renderClampedText(element)
  },
  unmounted(element) {
    const state = clampState.get(element)
    if (!state) return
    cancelAnimationFrame(state.frameId)
    state.resizeObserver.disconnect()
    clampState.delete(element)
  },
}
