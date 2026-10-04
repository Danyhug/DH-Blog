import type { Directive } from 'vue'

// Full class literals so Tailwind's source scan generates them.
// While waiting the image has nothing to paint, so its own background is the placeholder;
// text-transparent hides the alt text some browsers draw in the meantime.
const PENDING = ['animate-pulse', 'bg-black/5', 'dark:bg-white/10', 'text-transparent']
// Markdown images have no size until they load. Reserve a box so the reader sees where a picture goes
// and the text below jumps less. A vw cap, not a percentage: the theme wraps images in an inline-flex
// <figure>, which sizes itself from its content, so a % width would resolve to nothing.
const CONTENT_BOX = ['w-[min(36rem,80vw)]', 'aspect-video', 'rounded-lg']
const APPEAR = 'animate-image-in'

const cleanups = new WeakMap<HTMLImageElement, () => void>()
const trackedSrc = new WeakMap<HTMLImageElement, string>()
const observers = new WeakMap<HTMLElement, MutationObserver>()

function track(img: HTMLImageElement, box: string[]) {
  cleanups.get(img)?.()
  trackedSrc.set(img, img.src)
  // Cached images paint at once and broken ones are already settled: only fade those that wait on the network.
  // A lazy image not yet near the viewport reports complete = false too, so it keeps the placeholder until it loads.
  if (img.complete) return

  const pending = [...PENDING, ...box]
  img.classList.add(...pending)
  const settle = (event: Event) => {
    cleanup()
    if (event.type !== 'load') return
    img.classList.add(APPEAR)
    img.addEventListener('animationend', function done(e) {
      if (e.target !== img) return
      img.removeEventListener('animationend', done)
      img.classList.remove(APPEAR)
    })
  }
  const cleanup = () => {
    img.removeEventListener('load', settle)
    img.removeEventListener('error', settle)
    img.classList.remove(...pending)
    cleanups.delete(img)
  }
  img.addEventListener('load', settle)
  img.addEventListener('error', settle)
  cleanups.set(img, cleanup)
}

// Markdown output usually has no size; an <img> that states one keeps it.
const contentBox = (img: HTMLImageElement) =>
  img.hasAttribute('width') || img.hasAttribute('height') ? [] : CONTENT_BOX

function trackTree(node: Node) {
  if (node instanceof HTMLImageElement) track(node, contentBox(node))
  else if (node instanceof Element) node.querySelectorAll('img').forEach(img => track(img, contentBox(img)))
}

/*
  Network images show a pulsing placeholder while loading, then fade in.
  - On an <img>: tracks that image; the binding value adds placeholder sizing classes (literal strings in the template).
  - On any other element: tracks every <img> inside it, including ones inserted later (rendered Markdown).
*/
export const vImgFade: Directive<HTMLElement, string | undefined> = {
  mounted(el, { value }) {
    if (el instanceof HTMLImageElement) {
      track(el, value ? value.split(' ') : [])
      return
    }
    trackTree(el)
    const observer = new MutationObserver(records => records.forEach(record => record.addedNodes.forEach(trackTree)))
    observer.observe(el, { childList: true, subtree: true })
    observers.set(el, observer)
  },
  updated(el, { value }) {
    // A bound :src that changes (e.g. the sidebar cover between articles) starts a new load.
    if (el instanceof HTMLImageElement && trackedSrc.get(el) !== el.src) track(el, value ? value.split(' ') : [])
  },
  unmounted(el) {
    if (el instanceof HTMLImageElement) cleanups.get(el)?.()
    observers.get(el)?.disconnect()
    observers.delete(el)
  },
}
