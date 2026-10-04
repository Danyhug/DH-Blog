// Delete effect for drive cards: the card is wiped away diagonally (bottom-right → top-left) while its
// icon and text break into soft particles that a wind blows off to the top-left like smoke.
// The particle model follows the "wind smoke" demo the effect was designed from: weak gravity, a dominant
// wind, light damping, a sideways sine wobble for S-shaped drift, and particles that swell and fade.

const WIPE_MS = 520
const FRAME_MS = 1000 / 60
const MAX_PARTICLES = 900
const ALPHA_THRESHOLD = 40

const GRAVITY = 0.012
const WIND_X = -0.085
const WIND_Y = -0.045
const DAMPING = 0.982
const BURST_SPEED = [0.5, 1.8]
const WIND_SPEED = [1.6, 4.0]
const JITTER = 1.0
const TURBULENCE = 0.03
const WOBBLE_AMP = 0.07
const WOBBLE_FREQ = [0.05, 0.12]
const SIZE = [2.5, 5.0]
const LIFE = [100, 170]
const GROW = 1.6
const STRETCH_MAX = 1.6
const SPIN = 0.3

// Wind blows along the wipe: towards the top-left. PERP is the wobble axis.
const WIND_NX = -Math.SQRT1_2
const WIND_NY = -Math.SQRT1_2
const WIND_ANGLE = Math.atan2(WIND_NY, WIND_NX)
const PERP_X = -WIND_NY
const PERP_Y = WIND_NX

interface Particle {
  x: number
  y: number
  vx: number
  vy: number
  size: number
  life: number
  maxLife: number
  sprite: HTMLCanvasElement
  rot: number
  age: number
  freq: number
  phase: number
  amp: number
  /** Frames to wait until the wipe line reaches this particle's spot */
  delay: number
}

const between = ([min, max]: number[]) => min + Math.random() * (max - min)

// ---- Soft glow sprites ----
// Each particle is a small radial-gradient puff instead of a hard square: that is what makes it read as smoke.
// Colors are quantised so a card only needs a handful of sprites.
const SPRITE_SIZE = 48
const sprites = new Map<number, HTMLCanvasElement>()

function sprite(r: number, g: number, b: number) {
  const q = (v: number) => Math.min(255, Math.round(v / 16) * 16)
  const [qr, qg, qb] = [q(r), q(g), q(b)]
  const key = (qr << 16) | (qg << 8) | qb
  let cached = sprites.get(key)
  if (cached) return cached

  cached = document.createElement('canvas')
  cached.width = cached.height = SPRITE_SIZE
  const ctx = cached.getContext('2d')!
  const half = SPRITE_SIZE / 2
  const gradient = ctx.createRadialGradient(half, half, 0, half, half, half)
  for (const [stop, alpha] of [[0, 1], [0.14, 0.92], [0.34, 0.42], [0.6, 0.14], [0.82, 0.035], [1, 0]]) {
    gradient.addColorStop(stop, `rgba(${qr},${qg},${qb},${alpha})`)
  }
  ctx.fillStyle = gradient
  ctx.fillRect(0, 0, SPRITE_SIZE, SPRITE_SIZE)
  sprites.set(key, cached)
  return cached
}

// ---- Shared overlay canvas ----
// One fixed full-viewport canvas for every running effect; it only exists while particles are alive.
let canvas: HTMLCanvasElement | null = null
let ctx: CanvasRenderingContext2D | null = null
let particles: Particle[] = []
let frame = 0
let lastTime = 0

function ensureCanvas() {
  if (!canvas) {
    canvas = document.createElement('canvas')
    canvas.setAttribute('aria-hidden', 'true')
    Object.assign(canvas.style, { position: 'fixed', inset: '0', width: '100vw', height: '100vh', pointerEvents: 'none', zIndex: '2000' })
    document.body.appendChild(canvas)
    ctx = canvas.getContext('2d')
  }
  const dpr = Math.min(window.devicePixelRatio || 1, 2)
  const width = Math.round(innerWidth * dpr)
  const height = Math.round(innerHeight * dpr)
  if (canvas.width !== width || canvas.height !== height) {
    canvas.width = width
    canvas.height = height
  }
  ctx!.setTransform(dpr, 0, 0, dpr, 0, 0)
}

function tick(now: number) {
  const c = ctx!
  let dt = (now - lastTime) / FRAME_MS
  lastTime = now
  if (!Number.isFinite(dt) || dt <= 0) dt = 1
  if (dt > 3) dt = 3

  c.clearRect(0, 0, innerWidth, innerHeight)
  const damp = Math.pow(DAMPING, dt)
  for (let i = particles.length - 1; i >= 0; i--) {
    const p = particles[i]
    if (p.delay > 0) {
      p.delay -= dt
      continue
    }
    p.age += dt
    p.vx += WIND_X * dt
    p.vy += (WIND_Y + GRAVITY) * dt
    const wobble = Math.sin(p.age * p.freq + p.phase) * p.amp
    p.vx += (PERP_X * wobble + (Math.random() - 0.5) * TURBULENCE) * dt
    p.vy += (PERP_Y * wobble + (Math.random() - 0.5) * TURBULENCE) * dt
    p.vx *= damp
    p.vy *= damp
    p.x += p.vx * dt
    p.y += p.vy * dt
    p.life -= dt

    if (p.life <= 0 || p.x < -100 || p.y < -100 || p.x > innerWidth + 100 || p.y > innerHeight + 100) {
      particles[i] = particles[particles.length - 1]
      particles.pop()
      continue
    }

    // Swells and fades as it drifts, stretched a little along its motion
    const t = p.life / p.maxLife
    const size = p.size * (1 + GROW * (1 - t))
    const speed = Math.hypot(p.vx, p.vy)
    const stretch = 1 + Math.min(speed * 0.14, STRETCH_MAX - 1)
    c.globalAlpha = Math.min(1, Math.pow(t, 0.7))
    c.save()
    c.translate(p.x, p.y)
    c.rotate(Math.atan2(p.vy, p.vx) + p.rot)
    if (stretch > 1.03) c.scale(stretch, 1)
    c.drawImage(p.sprite, -size / 2, -size / 2, size, size)
    c.restore()
  }
  c.globalAlpha = 1

  if (particles.length) {
    frame = requestAnimationFrame(tick)
  } else {
    frame = 0
    canvas?.remove()
    canvas = ctx = null
  }
}

function start() {
  if (frame) return
  lastTime = performance.now()
  frame = requestAnimationFrame(tick)
}

// ---- Snapshot of the card's content ----
// Only the icon and the text become particles: the card background is white (or nearly black) on a page of
// the same color, so its particles would be invisible anyway and would crowd out the ones that show.

async function drawSvg(target: CanvasRenderingContext2D, svg: SVGSVGElement, origin: DOMRect) {
  const rect = svg.getBoundingClientRect()
  if (!rect.width || !rect.height) return
  const clone = svg.cloneNode(true) as SVGSVGElement
  // Outside the page currentColor has nothing to inherit from; pin it to the color the icon is drawn in
  clone.setAttribute('xmlns', 'http://www.w3.org/2000/svg')
  clone.setAttribute('width', String(rect.width))
  clone.setAttribute('height', String(rect.height))
  clone.setAttribute('color', getComputedStyle(svg).color)
  clone.removeAttribute('class')
  const image = new Image()
  image.src = `data:image/svg+xml;charset=utf-8,${encodeURIComponent(new XMLSerializer().serializeToString(clone))}`
  try {
    await image.decode()
  } catch {
    return
  }
  target.drawImage(image, rect.left - origin.left, rect.top - origin.top, rect.width, rect.height)
}

function drawText(target: CanvasRenderingContext2D, root: HTMLElement, origin: DOMRect) {
  const walker = document.createTreeWalker(root, NodeFilter.SHOW_TEXT)
  const range = document.createRange()
  for (let node = walker.nextNode(); node; node = walker.nextNode()) {
    const text = node.textContent?.trim()
    const parent = node.parentElement
    if (!text || !parent) continue
    range.selectNodeContents(node)
    const rect = range.getBoundingClientRect()
    if (!rect.width) continue
    const style = getComputedStyle(parent)
    const clip = parent.getBoundingClientRect()
    target.save()
    // Truncated names overflow their box; clip to what is actually visible
    target.beginPath()
    target.rect(clip.left - origin.left, clip.top - origin.top, clip.width, clip.height)
    target.clip()
    target.font = `${style.fontWeight} ${style.fontSize} ${style.fontFamily}`
    target.fillStyle = style.color
    target.textBaseline = 'middle'
    target.fillText(text, rect.left - origin.left, rect.top - origin.top + rect.height / 2)
    target.restore()
  }
}

async function spawn(el: HTMLElement, rect: DOMRect) {
  const scale = 2
  const snapshot = document.createElement('canvas')
  snapshot.width = Math.ceil(rect.width * scale)
  snapshot.height = Math.ceil(rect.height * scale)
  const target = snapshot.getContext('2d', { willReadFrequently: true })!
  target.scale(scale, scale)
  await Promise.all([...el.querySelectorAll('svg')].map(svg => drawSvg(target, svg, rect)))
  drawText(target, el, rect)

  const { width, height } = snapshot
  const data = target.getImageData(0, 0, width, height).data
  let opaque = 0
  for (let i = 3; i < data.length; i += 16) if (data[i] >= ALPHA_THRESHOLD) opaque++
  // Sample on a grid coarse enough to stay under the cap (opaque counted on every 4th pixel above)
  const step = Math.max(3 * scale, Math.ceil(Math.sqrt((opaque * 4) / MAX_PARTICLES)))
  const waveFrames = WIPE_MS / FRAME_MS
  const cx = rect.left + rect.width / 2
  const cy = rect.top + rect.height / 2

  for (let y = 0; y < height; y += step) {
    for (let x = 0; x < width; x += step) {
      const i = (y * width + x) * 4
      if (data[i + 3] < ALPHA_THRESHOLD) continue
      const px = rect.left + (x + step / 2) / scale
      const py = rect.top + (y + step / 2) / scale
      // A weak radial burst keeps the shape recognisable for a moment; the wind does the rest
      const radialAngle = Math.atan2(py - cy, px - cx) + (Math.random() - 0.5) * JITTER
      const radialSpeed = between(BURST_SPEED)
      const windAngle = WIND_ANGLE + (Math.random() - 0.5) * JITTER * 0.7
      const windSpeed = between(WIND_SPEED)
      const maxLife = between(LIFE)
      // In sync with the wipe: the bottom-right corner peels off first, the top-left last
      const progress = ((x + step / 2) / width + (y + step / 2) / height) / 2
      particles.push({
        x: px,
        y: py,
        vx: Math.cos(radialAngle) * radialSpeed + Math.cos(windAngle) * windSpeed,
        vy: Math.sin(radialAngle) * radialSpeed + Math.sin(windAngle) * windSpeed,
        size: between(SIZE),
        life: maxLife,
        maxLife,
        sprite: sprite(data[i], data[i + 1], data[i + 2]),
        rot: (Math.random() - 0.5) * SPIN,
        age: 0,
        freq: between(WOBBLE_FREQ),
        phase: Math.random() * Math.PI * 2,
        amp: WOBBLE_AMP * (0.5 + Math.random()),
        delay: Math.max(0, (1 - progress) * waveFrames + (Math.random() - 0.5) * 1.5)
      })
    }
  }
  ensureCanvas()
  start()
}

function wipe(el: HTMLElement) {
  return new Promise<void>(resolve => {
    const begin = performance.now()
    const step = (now: number) => {
      const t = Math.min(1, (now - begin) / WIPE_MS)
      const edge = (1 - t) * 100
      // A few percent of feather so the edge looks like it crumbles rather than being cut
      el.style.maskImage = `linear-gradient(135deg, #000 ${edge - 4}%, transparent ${edge}%)`
      if (t < 1) requestAnimationFrame(step)
      else resolve()
    }
    requestAnimationFrame(step)
  })
}

/** Plays the delete effect on an element; resolves once it is fully wiped (particles keep drifting on their own). */
export async function dissolve(el: HTMLElement) {
  el.style.pointerEvents = 'none'
  if (matchMedia('(prefers-reduced-motion: reduce)').matches) {
    await el.animate([{ opacity: 1 }, { opacity: 0 }], { duration: 200, fill: 'forwards' }).finished
    return
  }
  const rect = el.getBoundingClientRect()
  // Snapshot first, while the content is still fully visible; then wipe and blow it away together
  await spawn(el, rect).catch(() => undefined)
  await wipe(el)
  el.style.visibility = 'hidden'
}

/**
 * Remaining items glide into the gap instead of jumping (FLIP).
 * Call before the list changes, then call the returned function once the DOM has updated.
 */
export function captureLayout(items: HTMLElement[]) {
  const before = new Map(items.map(item => [item, item.getBoundingClientRect()]))
  return () => {
    for (const [item, rect] of before) {
      if (!item.isConnected) continue
      const now = item.getBoundingClientRect()
      const dx = rect.left - now.left
      const dy = rect.top - now.top
      if (!dx && !dy) continue
      item.animate(
        [{ transform: `translate(${dx}px, ${dy}px)` }, { transform: 'none' }],
        { duration: 420, easing: 'cubic-bezier(0.34, 1.56, 0.64, 1)' }
      )
    }
  }
}
