<template>
  <!-- 词云：字号随文章数开方增长，按阿基米德螺线由中心向外摆放，互不重叠。
       词是真实的 <button>（可聚焦、可读屏）而不是画在 canvas 上 -->
  <div ref="boxRef" class="relative size-full overflow-hidden [perspective:1000px]"
    @pointermove="onTilt" @pointerleave="resetTilt">
    <!-- 整片词云随指针轻微倾斜，像在转动一块星盘 -->
    <!-- transform 由脚本直接写（见 onTilt），不经响应式：否则每次指针移动都会重渲染、比对全部词 -->
    <div ref="tiltRef" class="group/cloud absolute inset-0 transition-transform duration-500 ease-out">
      <button v-for="(word, index) in placed" :key="word.key" type="button"
        class="group/word absolute -translate-x-1/2 -translate-y-1/2 cursor-pointer whitespace-nowrap rounded-md px-1 py-0.5 font-semibold leading-none text-(--tag) dark:text-[color-mix(in_oklab,var(--tag)_62%,white)] transition-[left,top,opacity,scale,text-shadow] duration-300 ease-out animate-word-pop group-has-[button:hover]/cloud:opacity-35 hover:z-10 hover:scale-115 hover:opacity-100! hover:[text-shadow:0_0_18px_color-mix(in_oklab,var(--tag)_45%,transparent)] active:scale-95 focus-visible:z-10 focus-visible:opacity-100! focus-visible:outline-2 focus-visible:outline-(--tag)"
        :style="{ left: `${word.x}px`, top: `${word.y}px`, fontSize: `${word.size}px`, '--tag': word.color, animationDelay: `${Math.min(index, 40) * 30}ms` }"
        :aria-label="`标签 ${word.name}，${word.count} 篇文章`"
        @click="onSelect(word, $event)"
        @mouseenter="emit('hover', word.key)" @mouseleave="emit('hover', '')"
        @focus="emit('hover', word.key)" @blur="emit('hover', '')">
        <!-- 浮动放在内层：按钮自身的 transform 已被弹出动画占用，两个动画写在同一元素上会互相覆盖 -->
        <span class="relative inline-block animate-float group-hover/word:[animation-play-state:paused]"
          :style="{ animationDuration: `${word.floatDuration}s`, animationDelay: `-${word.floatDelay}s` }">
          {{ word.name }}
          <!-- 绝对定位：不参与宽度，布局时量出的尺寸就是按钮的真实尺寸 -->
          <sup class="pointer-events-none absolute left-full top-0 ml-0.5 text-[max(0.4em,10px)] font-medium opacity-0 transition-opacity duration-200 group-hover/word:opacity-80 group-focus-visible/word:opacity-80">{{ word.count }}</sup>
        </span>
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref, watch } from 'vue';
import { tagBaseColor, tagNameHash } from '@/utils/tagColor';
import { debounce } from '@/utils/tool';

interface CloudWord {
  key: string;
  name: string;
  type: string;
  count: number;
}

interface PlacedWord extends CloudWord {
  x: number;
  y: number;
  size: number;
  color: string;
  floatDuration: number;
  floatDelay: number;
}

interface Rect {
  left: number;
  top: number;
  right: number;
  bottom: number;
}

const props = defineProps<{ words: CloudWord[] }>();
const emit = defineEmits<{
  // origin: viewport centre of the clicked word, so the article dialog can fly out of it.
  select: [word: CloudWord, origin: { x: number; y: number }];
  hover: [key: string];
}>();

const boxRef = ref<HTMLElement | null>(null);
const placed = ref<PlacedWord[]>([]);
const tiltRef = ref<HTMLElement | null>(null);
const reducedMotion = matchMedia('(prefers-reduced-motion: reduce)').matches;
const MAX_TILT = 5;
let tiltFrame = 0;
let tiltTarget = '';

// Pointer events can fire several times per frame; coalesce into one style write per frame.
const applyTilt = (transform: string) => {
  tiltTarget = transform;
  if (tiltFrame) return;
  tiltFrame = requestAnimationFrame(() => {
    tiltFrame = 0;
    if (tiltRef.value) tiltRef.value.style.transform = tiltTarget;
  });
};
const onTilt = (event: PointerEvent) => {
  if (reducedMotion || event.pointerType !== 'mouse' || !boxRef.value) return;
  const rect = boxRef.value.getBoundingClientRect();
  const nx = (event.clientX - rect.left) / rect.width - 0.5;
  const ny = (event.clientY - rect.top) / rect.height - 0.5;
  applyTilt(`rotateX(${(-ny * MAX_TILT * 2).toFixed(2)}deg) rotateY(${(nx * MAX_TILT * 2).toFixed(2)}deg)`);
};
const resetTilt = () => applyTilt('');

const onSelect = (word: CloudWord, event: MouseEvent) => {
  const rect = (event.currentTarget as HTMLElement).getBoundingClientRect();
  emit('select', word, { x: rect.left + rect.width / 2, y: rect.top + rect.height / 2 });
};

const measureContext = document.createElement('canvas').getContext('2d');

const GAP = 5;
const MIN_SIZE = 13;
const MAX_SPIRAL_STEPS = 5000;

const overlaps = (a: Rect, b: Rect) =>
  a.left < b.right + GAP && a.right + GAP > b.left && a.top < b.bottom + GAP && a.bottom + GAP > b.top;

// Walk an Archimedean spiral (stretched to the box's aspect ratio) until the box fits.
const findSpot = (width: number, height: number, boxWidth: number, boxHeight: number, startAngle: number, taken: Rect[]) => {
  const aspect = Math.max(1, boxWidth / boxHeight);
  for (let step = 0; step < MAX_SPIRAL_STEPS; step++) {
    const theta = step * 0.1;
    const radius = theta * 1.1;
    const cx = boxWidth / 2 + Math.cos(theta + startAngle) * radius * aspect;
    const cy = boxHeight / 2 + Math.sin(theta + startAngle) * radius;
    const rect = { left: cx - width / 2, top: cy - height / 2, right: cx + width / 2, bottom: cy + height / 2 };
    if (rect.left < GAP || rect.top < GAP || rect.right > boxWidth - GAP || rect.bottom > boxHeight - GAP) {
      // The spiral has grown past the box on both axes: nothing further out can fit.
      if (radius * aspect > boxWidth && radius > boxHeight) return null;
      continue;
    }
    if (!taken.some(other => overlaps(rect, other))) return { cx, cy, rect };
  }
  return null;
};

const layout = () => {
  const box = boxRef.value;
  if (!box || !measureContext) return;
  const boxWidth = box.clientWidth, boxHeight = box.clientHeight;
  if (!boxWidth || !boxHeight) return;

  const family = getComputedStyle(box).fontFamily;
  // Biggest first: they claim the centre, small words fill the gaps around them.
  const sorted = [...props.words].sort((a, b) => b.count - a.count || a.name.localeCompare(b.name));
  const maxCount = sorted[0]?.count ?? 1;
  const minCount = sorted[sorted.length - 1]?.count ?? 1;
  const maxSize = Math.min(60, Math.max(26, Math.min(boxWidth, boxHeight) / 6));

  const taken: Rect[] = [];
  const result: PlacedWord[] = [];
  for (const word of sorted) {
    // sqrt keeps one very popular tag from dwarfing everything else.
    const ratio = maxCount === minCount ? 0.5 : Math.sqrt((word.count - minCount) / (maxCount - minCount));
    let size = MIN_SIZE + (maxSize - MIN_SIZE) * ratio;
    const startAngle = (tagNameHash(word.name) % 360) * (Math.PI / 180);
    // Shrink and retry before giving up, so a crowded cloud still shows every tag where possible.
    for (let attempt = 0; attempt < 4; attempt++, size = Math.max(MIN_SIZE * 0.85, size * 0.85)) {
      measureContext.font = `600 ${size}px ${family}`;
      // + px-1 / py-0.5 padding of the button.
      const width = measureContext.measureText(word.name).width + 8;
      const height = size + 4;
      const spot = findSpot(width, height, boxWidth, boxHeight, startAngle, taken);
      if (!spot) continue;
      taken.push(spot.rect);
      const seed = tagNameHash(word.name);
      result.push({
        ...word, x: spot.cx, y: spot.cy, size, color: tagBaseColor(word.name),
        // Hash-derived so each word keeps its own rhythm across relayouts.
        floatDuration: 5 + (seed % 30) / 10,
        floatDelay: (seed % 70) / 10,
      });
      break;
    }
  }
  placed.value = result;
};

const relayout = debounce(layout, 150);
let resizeObserver: ResizeObserver | null = null;

watch(() => props.words, layout);

onMounted(() => {
  layout();
  // Re-measure once web fonts settle; a fallback font measures differently.
  document.fonts?.ready.then(layout);
  resizeObserver = new ResizeObserver(relayout);
  if (boxRef.value) resizeObserver.observe(boxRef.value);
});

onBeforeUnmount(() => {
  resizeObserver?.disconnect();
  cancelAnimationFrame(tiltFrame);
});
</script>
