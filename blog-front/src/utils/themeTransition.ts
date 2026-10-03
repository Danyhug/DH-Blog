import { nextTick } from 'vue';

// Reveal the new theme as a circle growing from `origin` (styles in tailwind.css).
// Shared by the article toolbar and the knowledge page so both toggles feel the same.
export const revealThemeChange = (origin: HTMLElement, flip: () => void) => {
  // element.animate() ignores the global reduced-motion CSS, so honour the preference here.
  if (typeof document.startViewTransition !== 'function' || matchMedia('(prefers-reduced-motion: reduce)').matches) {
    flip();
    return;
  }
  // Use the element's centre rather than the pointer: keyboard activation reports 0,0.
  const rect = origin.getBoundingClientRect();
  const x = rect.left + rect.width / 2;
  const y = rect.top + rect.height / 2;
  const radius = Math.hypot(Math.max(x, innerWidth - x), Math.max(y, innerHeight - y));
  const transition = document.startViewTransition(async () => {
    flip();
    await nextTick();
  });
  transition.ready.then(() => {
    document.documentElement.animate(
      { clipPath: [`circle(0px at ${x}px ${y}px)`, `circle(${radius}px at ${x}px ${y}px)`] },
      { duration: 450, easing: 'ease-in-out', pseudoElement: '::view-transition-new(root)' },
    );
  }).catch(() => {
    // Transition was skipped (e.g. tab hidden); the theme has still switched.
  });
};
