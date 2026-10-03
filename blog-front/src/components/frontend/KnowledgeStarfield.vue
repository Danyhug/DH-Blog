<template>
  <!-- 知识星图背景：底色渐变 + 画布。每颗亮星对应一个分类/标签，大小随文章数变化，近邻连成星座。
       pointer-events-none 不挡卡片点击，鼠标位置改从 window 读 -->
  <div aria-hidden="true"
    class="pointer-events-none fixed inset-0 z-0 bg-[radial-gradient(ellipse_at_15%_0%,#e3eafb_0%,#f4f7fc_50%,#eef1f8_100%)] dark:bg-[radial-gradient(ellipse_at_15%_0%,#1b2244_0%,#0d1022_50%,#07090f_100%)]">
    <canvas ref="canvasRef" class="block size-full"></canvas>
  </div>
</template>

<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref, watch } from 'vue';
import { tagBaseColor } from '@/utils/tagColor';

interface StarItem {
  name: string;
  type: string;
  count: number;
}

const props = defineProps<{
  items: StarItem[];
  // `${type}:${name}` of the card under the pointer; its star lights up.
  activeKey?: string;
  // Freeze the sky, e.g. while a backdrop-blurred dialog covers it: blurring a canvas that
  // changes every frame forces the browser to redo the blur every frame.
  paused?: boolean;
}>();

interface Node {
  key: string;
  name: string;
  isCategory: boolean;
  // Tag stars wear the tag's own colour (same as its word in the cloud); dark gets a lighter mix.
  rgb: { light: string; dark: string };
  // Anchor in 0..1 page coordinates; the star drifts around it.
  ax: number;
  ay: number;
  radius: number;
  phase: number;
  bornAt: number;
  x: number;
  y: number;
}

interface Dust {
  x: number;
  y: number;
  size: number;
  alpha: number;
  speed: number;
  phase: number;
}

interface Meteor {
  x: number;
  y: number;
  vx: number;
  vy: number;
  bornAt: number;
}

// RGB triplets; light mode reads as ink on a printed chart, dark as a night sky.
const PALETTES = {
  light: { key: 'light', dust: '59,74,120', grid: '70,90,150', link: '90,105,160', category: '0,150,160', label: '45,55,72', gridAlpha: 0.09, glow: 0.14 },
  dark: { key: 'dark', dust: '225,232,255', grid: '150,170,255', link: '150,170,255', category: '80,230,215', label: '220,228,255', gridAlpha: 0.07, glow: 0.3 },
} as const;

// '#rrggbb' → 'r,g,b', optionally mixed towards white (matches the chips' dark-mode color-mix).
const toRgb = (hex: string, white = 0) => [1, 3, 5]
  .map(i => Math.round(parseInt(hex.slice(i, i + 2), 16) * (1 - white) + 255 * white))
  .join(',');

type Palette = (typeof PALETTES)[keyof typeof PALETTES];
const starRgb = (node: Node, palette: Palette) => (node.isCategory ? palette.category : node.rgb[palette.key]);

const LINK_NEIGHBOURS = 2;
const ACTIVE_NEIGHBOURS = 6;
const LINK_DISTANCE = 190;
const POINTER_DISTANCE = 150;
const DRIFT = 10;
const FADE_IN_MS = 900;
// Parallax shift (px) at the screen edge per layer: nearer layers move more, which reads as depth.
const PARALLAX = { grid: 8, dust: 14, stars: 26 };
const RIPPLE_LIFE_MS = 1500;
const RIPPLE_SPEED = 0.55;
const RIPPLE_BAND = 36;
// A full-screen canvas at 2x is 4x the pixels of 1x; 1.5 keeps the dots crisp at ~56% of that fill cost.
const MAX_DPR = 1.5;
// Glow sprite: the halo spans 5x the core radius, as the old per-frame radial gradient did.
const SPRITE_SIZE = 64;
const SPRITE_CORE = SPRITE_SIZE / 10;

const canvasRef = ref<HTMLCanvasElement | null>(null);
const reducedMotion = window.matchMedia('(prefers-reduced-motion: reduce)').matches;

let ctx: CanvasRenderingContext2D | null = null;
let width = 0;
let height = 0;
let dpr = 1;
let frame = 0;
let nodes: Node[] = [];
// Constellation edges as index pairs. Drift is ±10px and parallax shifts every star equally,
// so nearest neighbours never change between frames: compute them once per layout, not per frame.
let links: [number, number][] = [];
let activeLinks: number[] = [];
let dust: Dust[] = [];
let meteor: Meteor | null = null;
let nextMeteorAt = 0;
let pointer: { x: number; y: number } | null = null;
// Eased towards the pointer each frame so the sky glides instead of snapping.
const parallax = { x: 0, y: 0 };
let ripples: { x: number; y: number; bornAt: number }[] = [];
let themeObserver: MutationObserver | null = null;
// Static dashed rings, rendered once per size/theme; dashed arcs are the priciest strokes to redo per frame.
let ringLayer: HTMLCanvasElement | null = null;
let ringLayerTheme = '';
const RING_MARGIN = PARALLAX.grid * 2;
// One pre-rendered glow per colour+theme, stamped with drawImage instead of building gradients per star per frame.
const sprites = new Map<string, HTMLCanvasElement>();

// Stable per-name randomness: a tag keeps its spot in the sky across visits.
const hash = (text: string, seed = 0) => {
  let h = 2166136261 ^ seed;
  for (const char of text) h = Math.imul(h ^ char.codePointAt(0)!, 16777619);
  return (h >>> 0) / 4294967296;
};

const sprite = (rgb: string, glow: number) => {
  const key = `${rgb}|${glow}`;
  let canvas = sprites.get(key);
  if (canvas) return canvas;
  canvas = document.createElement('canvas');
  canvas.width = canvas.height = SPRITE_SIZE;
  const c = canvas.getContext('2d')!;
  const mid = SPRITE_SIZE / 2;
  const halo = c.createRadialGradient(mid, mid, 0, mid, mid, mid);
  halo.addColorStop(0, `rgba(${rgb},${glow})`);
  halo.addColorStop(1, `rgba(${rgb},0)`);
  c.fillStyle = halo;
  c.fillRect(0, 0, SPRITE_SIZE, SPRITE_SIZE);
  c.fillStyle = `rgb(${rgb})`;
  c.beginPath();
  c.arc(mid, mid, SPRITE_CORE, 0, Math.PI * 2);
  c.fill();
  sprites.set(key, canvas);
  return canvas;
};

// Nearest neighbours of `index` within `range`, closest first, measured on anchors.
const nearest = (index: number, count: number, range: number) => {
  const node = nodes[index];
  const found: { j: number; dist: number }[] = [];
  for (let j = 0; j < nodes.length; j++) {
    if (j === index) continue;
    const dist = Math.hypot((nodes[j].ax - node.ax) * width, (nodes[j].ay - node.ay) * height);
    if (dist < range) found.push({ j, dist });
  }
  return found.sort((a, b) => a.dist - b.dist).slice(0, count).map(({ j }) => j);
};

const buildLinks = () => {
  const seen = new Set<string>();
  links = [];
  nodes.forEach((_, i) => {
    for (const j of nearest(i, LINK_NEIGHBOURS, LINK_DISTANCE)) {
      const id = i < j ? `${i}-${j}` : `${j}-${i}`;
      if (seen.has(id)) continue;
      seen.add(id);
      links.push([i, j]);
    }
  });
  buildActiveLinks();
};

const buildActiveLinks = () => {
  const index = nodes.findIndex(node => node.key === props.activeKey);
  activeLinks = index < 0 ? [] : nearest(index, ACTIVE_NEIGHBOURS, LINK_DISTANCE * 1.6);
};

const buildNodes = (items: StarItem[]) => {
  const now = performance.now();
  const previous = new Map(nodes.map(node => [node.key, node]));
  const next = items.map((item, index) => {
    const key = `${item.type}:${item.name}`;
    const isCategory = item.type === 'category';
    const weight = Math.log2((item.count || 0) + 1);
    return {
      key,
      name: item.name,
      isCategory,
      rgb: { light: toRgb(tagBaseColor(item.name)), dark: toRgb(tagBaseColor(item.name), 0.4) },
      ax: 0.04 + hash(key, 1) * 0.92,
      ay: 0.06 + hash(key, 2) * 0.88,
      radius: isCategory ? 2.4 + weight * 1.1 : 1.3 + weight * 0.75,
      phase: hash(key, 3) * Math.PI * 2,
      // Stagger the first appearance so the constellation "lights up" instead of popping in.
      bornAt: previous.get(key)?.bornAt ?? now + index * 35,
      x: 0,
      y: 0,
    };
  });
  // Push anchors apart a little; pure hashing clumps stars and leaves big empty patches.
  const aspect = width / Math.max(height, 1) || 1;
  for (let round = 0; round < 40; round++) {
    for (let i = 0; i < next.length; i++) {
      for (let j = i + 1; j < next.length; j++) {
        const a = next[i], b = next[j];
        const dx = (a.ax - b.ax) * aspect, dy = a.ay - b.ay;
        const dist = Math.hypot(dx, dy) || 0.001;
        const min = 0.09;
        if (dist >= min) continue;
        const push = (min - dist) / 2 / dist;
        a.ax += (dx * push) / aspect; a.ay += dy * push;
        b.ax -= (dx * push) / aspect; b.ay -= dy * push;
      }
    }
    for (const node of next) {
      node.ax = Math.min(0.97, Math.max(0.03, node.ax));
      node.ay = Math.min(0.96, Math.max(0.04, node.ay));
    }
  }
  nodes = next;
  buildLinks();
};

const buildDust = () => {
  const count = Math.round(Math.min(220, (width * height) / 9000));
  dust = Array.from({ length: count }, () => ({
    x: Math.random() * width,
    y: Math.random() * height,
    size: 0.3 + Math.random() * 0.9,
    alpha: 0.25 + Math.random() * 0.55,
    speed: 0.6 + Math.random() * 1.6,
    phase: Math.random() * Math.PI * 2,
  }));
};

const gridCentre = () => ({ cx: width * 0.82, cy: height * 0.18 });

const buildRingLayer = (palette: Palette) => {
  const layer = ringLayer ?? document.createElement('canvas');
  layer.width = (width + RING_MARGIN * 2) * dpr;
  layer.height = (height + RING_MARGIN * 2) * dpr;
  const c = layer.getContext('2d')!;
  c.setTransform(dpr, 0, 0, dpr, RING_MARGIN * dpr, RING_MARGIN * dpr);
  const { cx, cy } = gridCentre();
  const maxR = Math.hypot(Math.max(cx, width - cx), Math.max(cy, height - cy)) + RING_MARGIN;
  c.strokeStyle = `rgba(${palette.grid},${palette.gridAlpha})`;
  c.lineWidth = 1;
  c.setLineDash([2, 6]);
  for (let r = 110; r < maxR; r += 110) {
    c.beginPath();
    c.arc(cx, cy, r, 0, Math.PI * 2);
    c.stroke();
  }
  ringLayer = layer;
  ringLayerTheme = palette.key;
};

const resize = () => {
  const canvas = canvasRef.value;
  if (!canvas) return;
  dpr = Math.min(window.devicePixelRatio || 1, MAX_DPR);
  width = window.innerWidth;
  height = window.innerHeight;
  canvas.width = width * dpr;
  canvas.height = height * dpr;
  ctx = canvas.getContext('2d');
  ctx?.setTransform(dpr, 0, 0, dpr, 0, 0);
  ringLayerTheme = '';
  buildDust();
  buildNodes(props.items);
  drawStill();
};

// A planisphere-style grid (dashed rings + hour lines) is what makes it read as a star *chart*.
const drawGrid = (c: CanvasRenderingContext2D, t: number, palette: Palette) => {
  if (ringLayerTheme !== palette.key) buildRingLayer(palette);
  const ox = -parallax.x * PARALLAX.grid, oy = -parallax.y * PARALLAX.grid;
  c.drawImage(ringLayer!, ox - RING_MARGIN, oy - RING_MARGIN, width + RING_MARGIN * 2, height + RING_MARGIN * 2);

  const { cx: baseX, cy: baseY } = gridCentre();
  const cx = baseX + ox, cy = baseY + oy;
  const maxR = Math.hypot(Math.max(cx, width - cx), Math.max(cy, height - cy));
  const rotation = reducedMotion ? 0 : t * 0.000012;
  c.strokeStyle = `rgba(${palette.grid},${palette.gridAlpha})`;
  c.lineWidth = 1;
  // All twelve hour lines in one path: one stroke call instead of twelve.
  c.beginPath();
  for (let i = 0; i < 12; i++) {
    const angle = rotation + (i * Math.PI) / 6;
    c.moveTo(cx + Math.cos(angle) * 40, cy + Math.sin(angle) * 40);
    c.lineTo(cx + Math.cos(angle) * maxR, cy + Math.sin(angle) * maxR);
  }
  c.stroke();
  // Tilted ecliptic-like ellipse crossing the page.
  c.strokeStyle = `rgba(${palette.grid},${palette.gridAlpha * 1.3})`;
  c.beginPath();
  c.ellipse(width * 0.5 + ox, height * 0.55 + oy, width * 0.62, height * 0.22, -0.18 + rotation * 0.5, 0, Math.PI * 2);
  c.stroke();
};

const drawMeteor = (c: CanvasRenderingContext2D, t: number, palette: Palette) => {
  if (!meteor && t >= nextMeteorAt) {
    const angle = Math.PI * (0.15 + Math.random() * 0.15);
    const speed = 0.9 + Math.random() * 0.5;
    meteor = { x: Math.random() * width * 0.8, y: Math.random() * height * 0.35, vx: Math.cos(angle) * speed, vy: Math.sin(angle) * speed, bornAt: t };
  }
  if (!meteor) return;
  const age = t - meteor.bornAt;
  const life = 1100;
  if (age > life) {
    meteor = null;
    nextMeteorAt = t + 5000 + Math.random() * 7000;
    return;
  }
  const headX = meteor.x + meteor.vx * age, headY = meteor.y + meteor.vy * age;
  const tail = 120;
  const tailX = headX - meteor.vx * tail, tailY = headY - meteor.vy * tail;
  const fade = Math.sin((age / life) * Math.PI);
  const gradient = c.createLinearGradient(headX, headY, tailX, tailY);
  gradient.addColorStop(0, `rgba(${palette.dust},${0.8 * fade})`);
  gradient.addColorStop(1, `rgba(${palette.dust},0)`);
  c.strokeStyle = gradient;
  c.lineWidth = 1.4;
  c.beginPath();
  c.moveTo(headX, headY);
  c.lineTo(tailX, tailY);
  c.stroke();
};

const drawStar = (c: CanvasRenderingContext2D, x: number, y: number, radius: number, image: HTMLCanvasElement, alpha: number) => {
  const size = radius * (SPRITE_SIZE / SPRITE_CORE);
  c.globalAlpha = alpha;
  c.drawImage(image, x - size / 2, y - size / 2, size, size);
};

// 0..1 brightness boost for a point the expanding click ripple is currently passing over.
const rippleBoost = (x: number, y: number, t: number) => {
  let boost = 0;
  for (const ripple of ripples) {
    const age = t - ripple.bornAt;
    const band = Math.abs(Math.hypot(x - ripple.x, y - ripple.y) - age * RIPPLE_SPEED);
    if (band < RIPPLE_BAND) boost = Math.max(boost, (1 - band / RIPPLE_BAND) * (1 - age / RIPPLE_LIFE_MS));
  }
  return boost;
};

const drawRipples = (c: CanvasRenderingContext2D, t: number, palette: Palette) => {
  ripples = ripples.filter(ripple => t - ripple.bornAt < RIPPLE_LIFE_MS);
  c.strokeStyle = `rgb(${palette.link})`;
  c.lineWidth = 1.2;
  for (const ripple of ripples) {
    const age = t - ripple.bornAt;
    const fade = 1 - age / RIPPLE_LIFE_MS;
    // Two rings, the inner one trailing, so it reads as a wave rather than a single circle.
    for (const [lag, alpha] of [[0, 0.45], [160, 0.25]]) {
      const radius = (age - lag) * RIPPLE_SPEED;
      if (radius <= 0) continue;
      c.globalAlpha = alpha * fade;
      c.beginPath();
      c.arc(ripple.x, ripple.y, radius, 0, Math.PI * 2);
      c.stroke();
    }
  }
  c.globalAlpha = 1;
};

const draw = (t: number) => {
  const c = ctx;
  if (!c) return;
  const palette = document.documentElement.classList.contains('dark') ? PALETTES.dark : PALETTES.light;
  c.clearRect(0, 0, width, height);

  if (!reducedMotion) {
    const targetX = pointer ? pointer.x / width - 0.5 : 0;
    const targetY = pointer ? pointer.y / height - 0.5 : 0;
    parallax.x += (targetX * 2 - parallax.x) * 0.04;
    parallax.y += (targetY * 2 - parallax.y) * 0.04;
  }

  drawGrid(c, t, palette);

  // Dust: one fill colour, per-star globalAlpha, and squares instead of arcs —
  // at under 2px they look the same and skip path building entirely.
  c.fillStyle = `rgb(${palette.dust})`;
  const hasRipples = ripples.length > 0;
  for (const star of dust) {
    const twinkle = reducedMotion ? 0.8 : 0.55 + 0.45 * Math.sin(t * 0.001 * star.speed + star.phase);
    // Bigger dust sits "closer", so it shifts further with the pointer.
    const x = star.x - parallax.x * PARALLAX.dust * star.size;
    const y = star.y - parallax.y * PARALLAX.dust * star.size;
    const boost = hasRipples ? rippleBoost(x, y, t) : 0;
    const size = star.size * 2 * (1 + boost);
    c.globalAlpha = Math.min(1, star.alpha * twinkle + boost);
    c.fillRect(x - size / 2, y - size / 2, size, size);
  }
  c.globalAlpha = 1;

  drawRipples(c, t, palette);
  if (!reducedMotion) drawMeteor(c, t, palette);

  const drift = reducedMotion ? 0 : DRIFT;
  for (const node of nodes) {
    node.x = node.ax * width + Math.sin(t * 0.00023 + node.phase) * drift - parallax.x * PARALLAX.stars;
    node.y = node.ay * height + Math.cos(t * 0.00019 + node.phase * 1.3) * drift - parallax.y * PARALLAX.stars;
  }
  const appear = (node: Node) => (reducedMotion ? 1 : Math.min(1, Math.max(0, (t - node.bornAt) / FADE_IN_MS)));
  const active = nodes.find(node => node.key === props.activeKey);

  // Constellation lines.
  c.lineWidth = 0.8;
  c.strokeStyle = `rgb(${palette.link})`;
  for (const [i, j] of links) {
    const a = nodes[i], b = nodes[j];
    if (a === active || b === active) continue;
    const dist = Math.hypot(a.x - b.x, a.y - b.y);
    c.globalAlpha = Math.max(0, 0.22 * (1 - dist / (LINK_DISTANCE * 1.6))) * Math.min(appear(a), appear(b));
    c.beginPath();
    c.moveTo(a.x, a.y);
    c.lineTo(b.x, b.y);
    c.stroke();
  }
  if (active) {
    c.strokeStyle = `rgb(${starRgb(active, palette)})`;
    for (const j of activeLinks) {
      const other = nodes[j];
      const dist = Math.hypot(active.x - other.x, active.y - other.y);
      c.globalAlpha = Math.max(0, 0.55 * (1 - dist / (LINK_DISTANCE * 1.6))) * appear(other);
      c.beginPath();
      c.moveTo(active.x, active.y);
      c.lineTo(other.x, other.y);
      c.stroke();
    }
  }

  if (pointer) {
    c.strokeStyle = `rgb(${palette.link})`;
    for (const node of nodes) {
      const dist = Math.hypot(node.x - pointer.x, node.y - pointer.y);
      if (dist > POINTER_DISTANCE) continue;
      c.globalAlpha = 0.35 * (1 - dist / POINTER_DISTANCE) * appear(node);
      c.beginPath();
      c.moveTo(pointer.x, pointer.y);
      c.lineTo(node.x, node.y);
      c.stroke();
    }
  }

  for (const node of nodes) {
    const shown = appear(node);
    if (!shown) continue;
    const twinkle = reducedMotion ? 1 : 0.8 + 0.2 * Math.sin(t * 0.0015 + node.phase);
    const isActive = node === active;
    const boost = hasRipples ? rippleBoost(node.x, node.y, t) : 0;
    const scale = (isActive ? 1.6 : 1) + boost * 0.8;
    const image = sprite(starRgb(node, palette), palette.glow);
    const alpha = Math.min(1, 0.75 * twinkle + (isActive ? 0.3 : 0) + boost) * shown;
    drawStar(c, node.x, node.y, node.radius * scale, image, alpha);
    // A second stamp doubles the halo for the highlighted star.
    if (isActive || boost > 0.3) drawStar(c, node.x, node.y, node.radius * scale, image, alpha * (isActive ? 1 : boost));
  }
  c.globalAlpha = 1;

  if (active) {
    const rgb = starRgb(active, palette);
    const pulse = reducedMotion ? 0.5 : (t % 1600) / 1600;
    c.strokeStyle = `rgba(${rgb},${0.6 * (1 - pulse)})`;
    c.lineWidth = 1.2;
    c.beginPath();
    c.arc(active.x, active.y, active.radius * 1.6 + 4 + pulse * 18, 0, Math.PI * 2);
    c.stroke();
    c.font = '12px -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif';
    c.fillStyle = `rgba(${palette.label},0.85)`;
    c.textAlign = active.x > width - 120 ? 'right' : 'left';
    c.fillText(active.name, active.x + (c.textAlign === 'right' ? -12 : 12), active.y - 10);
  }
};

const loop = (t: number) => {
  draw(t);
  frame = requestAnimationFrame(loop);
};
const start = () => {
  if (reducedMotion || frame || props.paused) return;
  frame = requestAnimationFrame(loop);
};
const stop = () => {
  cancelAnimationFrame(frame);
  frame = 0;
};
// Without a running loop (reduced motion, or paused) changes still need one repaint.
const drawStill = () => {
  if (!frame) draw(performance.now());
};

const onPointerMove = (event: PointerEvent) => {
  pointer = { x: event.clientX, y: event.clientY };
};
const onPointerLeave = () => {
  pointer = null;
};
// A click anywhere sends a wave across the sky; stars light up as it passes them.
const onPointerDown = (event: PointerEvent) => {
  ripples = [...ripples.slice(-3), { x: event.clientX, y: event.clientY, bornAt: performance.now() }];
};

watch(() => props.items, items => {
  buildNodes(items);
  drawStill();
});
watch(() => props.activeKey, () => {
  buildActiveLinks();
  drawStill();
});
watch(() => props.paused, paused => (paused ? stop() : start()));

onMounted(() => {
  resize();
  nextMeteorAt = performance.now() + 2500;
  window.addEventListener('resize', resize);
  window.addEventListener('pointermove', onPointerMove, { passive: true });
  document.documentElement.addEventListener('pointerleave', onPointerLeave);
  // No running loop to pick up a theme switch, so redraw when <html class="dark"> flips.
  themeObserver = new MutationObserver(drawStill);
  themeObserver.observe(document.documentElement, { attributes: true, attributeFilter: ['class'] });
  if (!reducedMotion) window.addEventListener('pointerdown', onPointerDown, { passive: true });
  start();
});

onBeforeUnmount(() => {
  stop();
  themeObserver?.disconnect();
  window.removeEventListener('resize', resize);
  window.removeEventListener('pointermove', onPointerMove);
  window.removeEventListener('pointerdown', onPointerDown);
  document.documentElement.removeEventListener('pointerleave', onPointerLeave);
});
</script>
