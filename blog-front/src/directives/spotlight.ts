import type { Directive } from 'vue'

// Tracks the pointer inside an element as --mx / --my on its `[data-spotlight]` child, so that
// overlay's radial-gradient can follow the cursor.
// The variables go on the overlay, not the host: custom properties inherit, so writing them on the
// host would restyle every descendant (all the cloud's words) on each pointer move.
const frames = new WeakMap<HTMLElement, number>()

const onMove = (event: PointerEvent) => {
  const el = event.currentTarget as HTMLElement
  const { clientX, clientY } = event
  // Several pointer events can land in one frame; write once per frame.
  if (frames.get(el)) return
  frames.set(el, requestAnimationFrame(() => {
    frames.set(el, 0)
    const overlay = el.querySelector<HTMLElement>(':scope > [data-spotlight]')
    if (!overlay) return
    const rect = el.getBoundingClientRect()
    overlay.style.setProperty('--mx', `${clientX - rect.left}px`)
    overlay.style.setProperty('--my', `${clientY - rect.top}px`)
  }))
}

export const vSpotlight: Directive<HTMLElement> = {
  mounted(el) {
    el.addEventListener('pointermove', onMove, { passive: true })
  },
  unmounted(el) {
    cancelAnimationFrame(frames.get(el) ?? 0)
    el.removeEventListener('pointermove', onMove)
  },
}
