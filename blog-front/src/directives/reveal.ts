import type { Directive } from 'vue'

// Full class literals so Tailwind's source scan generates them.
const HIDDEN = ['opacity-0', 'translate-y-6']
const MOTION = ['transition-[opacity,translate]', 'duration-700', 'ease-out']
// Stagger only within one observer batch: cards entering together cascade,
// while a card reached later by scrolling starts at once instead of waiting on its list index.
const STAGGER_MS = 80

const observer = new IntersectionObserver((entries, self) => {
  // Order by position, not observe() order: Vue's keyed diff mounts a replaced list back to front.
  entries.filter(entry => entry.isIntersecting)
    .sort((a, b) => a.boundingClientRect.top - b.boundingClientRect.top)
    .forEach((entry, index) => {
      const el = entry.target as HTMLElement
      self.unobserve(el)
      el.style.transitionDelay = `${index * STAGGER_MS}ms`
      // Transition classes go on only now: CSS takes transition-* from the after-change style,
      // so adding them together with removing HIDDEN still animates.
      el.classList.add(...MOTION)
      el.addEventListener('transitionend', function cleanup(event) {
        // Descendant transitions (e.g. the cover image hover zoom) bubble up here too.
        if (event.target !== el) return
        el.removeEventListener('transitionend', cleanup)
        el.style.transitionDelay = ''
        el.classList.remove(...MOTION)
      })
      el.classList.remove(...HIDDEN)
    })
}, { rootMargin: '0px 0px -10% 0px' })

// Fades an element up the first time it scrolls into view, then leaves it untouched.
export const vReveal: Directive<HTMLElement> = {
  mounted(el) {
    // HIDDEN only, no transition yet: if something forced a style recalc between insertion and this hook
    // (v-loading reads getComputedStyle on mount), adding a transition here would animate 1 → 0 instead.
    el.classList.add(...HIDDEN)
    observer.observe(el)
  },
  unmounted(el) {
    observer.unobserve(el)
  },
}
