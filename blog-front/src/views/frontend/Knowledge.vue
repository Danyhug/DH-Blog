<template>
  <div class="relative min-h-screen text-[#2d3748] dark:text-[#dbe2ff] [font-family:-apple-system,BlinkMacSystemFont,'Segoe_UI',Roboto,'Helvetica_Neue',Arial,sans-serif]">
    <KnowledgeStarfield :items="allData" :active-key="activeKey" :paused="modal.visible" />

    <div class="relative z-[1] mx-auto flex min-h-screen max-w-[1280px] flex-col px-4 pb-10 sm:px-8">
      <!-- 顶栏 -->
      <header class="flex items-center justify-between py-5">
        <router-link to="/"
          class="group inline-flex items-center gap-1.5 rounded-full px-3 py-1.5 text-[14px] text-[#4a5578] no-underline transition-colors hover:bg-white/60 dark:text-[#aab4d8] dark:hover:bg-white/10">
          <ArrowLeft class="size-4 transition-transform duration-200 group-hover:-translate-x-0.5" aria-hidden="true" />
          内容中心
        </router-link>
        <button type="button" :aria-label="isDark ? '切换到日间模式' : '切换到夜间模式'" :aria-pressed="isDark"
          class="grid size-9 cursor-pointer place-items-center overflow-hidden rounded-full text-[#4a5578] transition-[background-color,scale] duration-300 hover:scale-110 hover:bg-white/60 active:scale-90 dark:text-[#aab4d8] dark:hover:bg-white/10 focus-visible:outline-2 focus-visible:outline-[#4f6bed]"
          @click="toggleTheme">
          <!-- 日月图标旋转交替 -->
          <Transition mode="out-in"
            enter-active-class="transition-[opacity,rotate,scale] duration-300 ease-out" enter-from-class="opacity-0 -rotate-90 scale-50"
            leave-active-class="transition-[opacity,rotate,scale] duration-150 ease-in" leave-to-class="opacity-0 rotate-90 scale-50">
            <Sunny v-if="isDark" key="sun" class="size-5" />
            <Moon v-else key="moon" class="size-5" />
          </Transition>
        </button>
      </header>

      <!-- 标题与统计 -->
      <section class="pt-6 pb-10 text-center animate-fade-in-up">
        <p class="m-0 mb-3 text-[13px] tracking-[0.4em] text-[#6b7bb0] animate-tracking-in dark:text-[#8f9bd0]">KNOWLEDGE&nbsp;CONSTELLATION</p>
        <!-- 渐变色带两倍宽并来回平移，标题颜色缓慢流动 -->
        <h1 class="m-0 bg-linear-to-r from-[#0d9488] via-[#4f46e5] to-[#c026d3] bg-size-[200%_auto] animate-gradient-pan bg-clip-text text-[clamp(2.2rem,6vw,3.6rem)] font-bold! tracking-[0.08em] text-transparent dark:from-[#5eead4] dark:via-[#a5b4fc] dark:to-[#f0abfc]">知识星图</h1>
        <p class="m-0 mt-3 text-[15px] text-[#5a6690] dark:text-[#9aa5cf]">每一颗星都是一个主题，越亮的星，写得越多</p>
        <div class="mt-7 flex flex-wrap justify-center gap-3">
          <div v-for="(stat, index) in stats" :key="stat.label" v-spotlight :class="glassClass"
            class="group/panel relative min-w-[120px] overflow-hidden rounded-2xl px-5 py-3 transition-[translate,box-shadow] duration-300 hover:-translate-y-1 hover:shadow-[0_14px_36px_rgba(79,70,229,0.16)] animate-fade-in-up"
            :style="{ animationDelay: `${200 + index * 120}ms` }">
            <div aria-hidden="true" data-spotlight class="pointer-events-none absolute inset-0 rounded-[inherit] opacity-0 [--mx:50%] [--my:50%] transition-opacity duration-500 group-hover/panel:opacity-100 bg-[radial-gradient(380px_circle_at_var(--mx)_var(--my),rgba(99,102,241,0.13),transparent_70%)] dark:bg-[radial-gradient(380px_circle_at_var(--mx)_var(--my),rgba(165,180,252,0.12),transparent_70%)]"></div>
            <p class="relative m-0 text-[26px] font-bold leading-tight tabular-nums" :class="stat.color">{{ stat.display }}</p>
            <p class="relative m-0 mt-0.5 text-[12px] text-[#6b7590] dark:text-[#8f99c0]">{{ stat.label }}</p>
          </div>
        </div>
      </section>

      <!-- 主体：词云 + 分类 -->
      <main class="grid flex-1 gap-6 lg:grid-cols-[minmax(0,2fr)_minmax(0,1fr)]">
        <section v-spotlight :class="glassClass" class="group/panel relative flex flex-col rounded-3xl p-5 animate-fade-in-up [animation-delay:450ms] sm:p-6">
          <div aria-hidden="true" data-spotlight class="pointer-events-none absolute inset-0 rounded-[inherit] opacity-0 [--mx:50%] [--my:50%] transition-opacity duration-500 group-hover/panel:opacity-100 bg-[radial-gradient(380px_circle_at_var(--mx)_var(--my),rgba(99,102,241,0.13),transparent_70%)] dark:bg-[radial-gradient(380px_circle_at_var(--mx)_var(--my),rgba(165,180,252,0.12),transparent_70%)]"></div>
          <div class="relative mb-2 flex items-baseline justify-between gap-3">
            <h2 class="m-0 flex items-center gap-2 text-[18px] font-semibold!">
              <span class="relative flex size-2" aria-hidden="true">
                <span class="absolute inline-flex size-full animate-ping rounded-full bg-[#c026d3] opacity-60 dark:bg-[#f0abfc]"></span>
                <span class="relative inline-flex size-2 rounded-full bg-[#c026d3] dark:bg-[#f0abfc]"></span>
              </span>
              标签词云
            </h2>
            <p class="m-0 text-[12px] text-[#8a93ad] dark:text-[#7f89b0]">字号越大，文章越多 · 点击查看</p>
          </div>
          <div class="relative h-[360px] sm:h-[440px] lg:h-auto lg:min-h-[460px] lg:flex-1">
            <div v-if="isLoading" class="absolute inset-0 grid place-items-center">
              <p class="m-0 animate-pulse text-[14px] text-[#8a93ad]">正在绘制星图…</p>
            </div>
            <p v-else-if="!tags.length" class="absolute inset-0 m-0 grid place-items-center text-[14px] text-[#8a93ad]">还没有标签</p>
            <KnowledgeWordCloud v-else :words="tags" class="absolute! inset-0" @select="(word, origin) => openModal(word, origin)" @hover="key => activeKey = key" />
          </div>
        </section>

        <section v-spotlight :class="glassClass" class="group/panel relative flex flex-col rounded-3xl p-5 animate-fade-in-up [animation-delay:600ms] sm:p-6">
          <div aria-hidden="true" data-spotlight class="pointer-events-none absolute inset-0 rounded-[inherit] opacity-0 [--mx:50%] [--my:50%] transition-opacity duration-500 group-hover/panel:opacity-100 bg-[radial-gradient(380px_circle_at_var(--mx)_var(--my),rgba(99,102,241,0.13),transparent_70%)] dark:bg-[radial-gradient(380px_circle_at_var(--mx)_var(--my),rgba(165,180,252,0.12),transparent_70%)]"></div>
          <div class="relative mb-4 flex items-baseline justify-between gap-3">
            <h2 class="m-0 flex items-center gap-2 text-[18px] font-semibold!">
              <span class="relative flex size-2" aria-hidden="true">
                <span class="absolute inline-flex size-full animate-ping rounded-full bg-[#0d9488] opacity-60 [animation-delay:500ms] dark:bg-[#5eead4]"></span>
                <span class="relative inline-flex size-2 rounded-full bg-[#0d9488] dark:bg-[#5eead4]"></span>
              </span>
              分类星系
            </h2>
            <p class="m-0 text-[12px] text-[#8a93ad] dark:text-[#7f89b0]">{{ categories.length }} 个分类</p>
          </div>
          <div v-if="isLoading" class="relative space-y-3">
            <div v-for="n in 5" :key="n" class="h-12 animate-pulse rounded-xl bg-black/5 dark:bg-white/5"></div>
          </div>
          <p v-else-if="!categories.length" class="m-0 py-10 text-center text-[14px] text-[#8a93ad]">还没有分类</p>
          <ul v-else class="relative m-0 -mx-1 list-none space-y-1.5 overflow-y-auto p-1 lg:max-h-[520px]">
            <li v-for="(category, index) in categories" :key="category.key"
              class="animate-tag-item-in" :style="{ animationDelay: `${700 + Math.min(index, 12) * 60}ms` }">
              <button type="button"
                class="group relative w-full cursor-pointer overflow-hidden rounded-xl px-4 py-3 text-left transition-[background-color,translate] duration-200 hover:translate-x-1 hover:bg-[#0d9488]/8 dark:hover:bg-[#5eead4]/10 focus-visible:outline-2 focus-visible:outline-[#0d9488]"
                :aria-label="`分类 ${category.name}，${category.count} 篇文章`"
                @click="openModal(category, centreOf($event))"
                @mouseenter="activeKey = category.key" @mouseleave="activeKey = ''"
                @focus="activeKey = category.key" @blur="activeKey = ''">
                <div class="flex items-center justify-between gap-3">
                  <span class="flex min-w-0 items-center gap-2.5">
                    <span class="size-2 shrink-0 rounded-full bg-[#0d9488] shadow-[0_0_0_4px_rgba(13,148,136,0.15)] transition-shadow duration-300 group-hover:shadow-[0_0_0_6px_rgba(13,148,136,0.22)] dark:bg-[#5eead4] dark:shadow-[0_0_0_4px_rgba(94,234,212,0.15)]"></span>
                    <span class="truncate text-[15px] font-medium transition-colors group-hover:text-[#0d9488] dark:group-hover:text-[#5eead4]">{{ category.name }}</span>
                  </span>
                  <span class="shrink-0 text-[13px] tabular-nums text-[#8a93ad] dark:text-[#8f99c0]">{{ category.count }} 篇</span>
                </div>
                <!-- 占比条：以最多的分类为满格 -->
                <div class="mt-2 h-1 overflow-hidden rounded-full bg-black/5 dark:bg-white/8">
                  <div class="relative h-full overflow-hidden rounded-full bg-linear-to-r from-[#0d9488] to-[#4f46e5] transition-[width] duration-1000 ease-out dark:from-[#5eead4] dark:to-[#a5b4fc]"
                    :style="{ width: barsReady ? `${Math.max(4, (category.count / maxCategoryCount) * 100)}%` : '0%', transitionDelay: `${700 + Math.min(index, 12) * 60}ms` }">
                    <!-- 悬停时一道高光沿条掠过 -->
                    <span class="absolute inset-0 -translate-x-full bg-linear-to-r from-transparent via-white/70 to-transparent opacity-0 group-hover:animate-shimmer group-hover:opacity-100"></span>
                  </div>
                </div>
              </button>
            </li>
          </ul>
        </section>
      </main>
    </div>

    <!-- 文章列表弹层 -->
    <Teleport to="body">
      <Transition enter-active-class="transition-opacity duration-300" leave-active-class="transition-opacity duration-300"
        enter-from-class="opacity-0" leave-to-class="opacity-0" @enter="flyIn" @leave="flyOut">
        <div v-if="modal.visible" class="fixed inset-0 z-[1000] flex items-center justify-center bg-[#0b1020]/25 p-4 backdrop-blur-sm dark:bg-black/55"
          @click.self="closeModal">
          <div role="dialog" aria-modal="true" :aria-label="`${modal.title} 的文章`" :style="{ '--accent': modal.color }"
            class="flex max-h-[80vh] w-full max-w-[560px] flex-col overflow-hidden rounded-3xl border border-white/70 bg-white/90 shadow-[0_24px_60px_rgba(30,40,90,0.25)] backdrop-blur-xl dark:border-white/10 dark:bg-[#121630]/92 dark:shadow-[0_24px_60px_rgba(0,0,0,0.6)]">
            <div class="flex items-start justify-between gap-4 border-b border-(--accent)/15 bg-linear-to-br from-(--accent)/15 to-transparent px-6 py-5">
              <div class="min-w-0">
                <p class="m-0 text-[12px] text-[#8a93ad] dark:text-[#8f99c0]">{{ modal.type === 'tag' ? '标签' : '分类' }}</p>
                <h3 class="m-0 mt-1 truncate text-[22px] font-bold! text-(--accent) dark:text-[color-mix(in_oklab,var(--accent)_62%,white)]">
                  <span v-if="modal.type === 'tag'" class="mr-0.5 opacity-60" aria-hidden="true">#</span>{{ modal.title }}
                </h3>
              </div>
              <div class="flex shrink-0 items-center gap-3">
                <span v-if="modal.articles" class="rounded-full bg-(--accent)/15 px-2.5 py-0.5 text-[12px] text-(--accent) animate-fade-in dark:text-[color-mix(in_oklab,var(--accent)_62%,white)]">{{ modal.articles.length }} 篇</span>
                <button ref="closeButtonRef" type="button" aria-label="关闭"
                  class="grid size-8 cursor-pointer place-items-center rounded-full text-[#6b7590] transition-[background-color,rotate] duration-300 hover:rotate-90 hover:bg-black/5 dark:text-[#aab4d8] dark:hover:bg-white/10 focus-visible:outline-2 focus-visible:outline-(--accent)"
                  @click="closeModal">
                  <Close class="size-4" />
                </button>
              </div>
            </div>
            <div class="overflow-y-auto overscroll-contain p-3">
              <div v-if="!modal.articles && !modal.failed" aria-busy="true" aria-label="加载中">
                <div v-for="n in 4" :key="n" class="flex animate-pulse items-center gap-3 px-3 py-3">
                  <div class="size-7 shrink-0 rounded-lg bg-black/5 dark:bg-white/10"></div>
                  <div class="flex-1 space-y-2">
                    <div class="h-3.5 w-3/4 rounded bg-black/5 dark:bg-white/10"></div>
                    <div class="h-2.5 w-1/3 rounded bg-black/5 dark:bg-white/10"></div>
                  </div>
                </div>
              </div>
              <p v-else-if="modal.failed" class="m-0 py-10 text-center text-[14px] text-[#8a93ad]">加载失败，请稍后重试</p>
              <p v-else-if="!modal.articles?.length" class="m-0 py-10 text-center text-[14px] text-[#8a93ad]">暂无文章</p>
              <ul v-else class="m-0 list-none p-0">
                <li v-for="(article, index) in modal.articles" :key="article.id"
                  class="animate-tag-item-in" :style="{ animationDelay: `${Math.min(index, 10) * 40}ms` }">
                  <button type="button" @click="goToArticle(article.id)"
                    class="group flex w-full cursor-pointer items-center gap-3 rounded-xl px-3 py-3 text-left transition-colors duration-200 hover:bg-(--accent)/10 dark:hover:bg-(--accent)/20 focus-visible:outline-2 focus-visible:outline-(--accent)">
                    <span class="grid size-7 shrink-0 place-items-center rounded-lg bg-black/5 text-[13px] tabular-nums text-[#8a93ad] transition-colors duration-200 group-hover:bg-(--accent) group-hover:text-white dark:bg-white/10">{{ index + 1 }}</span>
                    <div class="min-w-0 flex-1">
                      <p class="m-0 line-clamp-2 text-[15px] leading-snug transition-colors duration-200 group-hover:text-(--accent) dark:group-hover:text-[color-mix(in_oklab,var(--accent)_62%,white)]">{{ article.title }}</p>
                      <p class="m-0 mt-1 text-[12px] text-[#9aa1b8] dark:text-[#7f89b0]">
                        {{ article.createTime?.slice(0, 10) }} · {{ article.views ?? 0 }} 次阅读 · {{ article.wordNum ?? 0 }} 字
                      </p>
                    </div>
                    <ArrowRight class="size-4 shrink-0 text-(--accent) opacity-0 -translate-x-1 transition-[opacity,translate] duration-200 group-hover:translate-x-0 group-hover:opacity-100 group-focus-visible:translate-x-0 group-focus-visible:opacity-100" aria-hidden="true" />
                  </button>
                </li>
              </ul>
            </div>
          </div>
        </div>
      </Transition>
    </Teleport>
  </div>
</template>

<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, reactive, ref, watch } from 'vue';
import { useRouter } from 'vue-router';
import { ArrowLeft, ArrowRight, Close, Moon, Sunny } from '@element-plus/icons-vue';
import { getAllTaxonomies, getArticlesByTaxonomy } from '@/api/user';
import { vSpotlight } from '@/directives/spotlight';
import { revealThemeChange } from '@/utils/themeTransition';
import { useUserStore } from '@/store';
import { tagBaseColor } from '@/utils/tagColor';
import KnowledgeStarfield from '@/components/frontend/KnowledgeStarfield.vue';
import KnowledgeWordCloud from '@/components/frontend/KnowledgeWordCloud.vue';

interface Taxonomy {
  key: string;
  name: string;
  type: string;
  count: number;
}

type TaxonomyArticle = Awaited<ReturnType<typeof getArticlesByTaxonomy>>[number];

// No backdrop-blur here: the starfield behind repaints every frame, and blur over a moving
// backdrop is recomputed every frame too, which was the main source of dropped frames.
// A higher-opacity fill keeps text readable while stars still show through faintly.
const glassClass = 'border border-white/70 bg-white/70 shadow-[0_8px_32px_rgba(45,55,110,0.08)] dark:border-white/10 dark:bg-[#10142b]/70 dark:shadow-[0_8px_32px_rgba(0,0,0,0.35)]';
const CATEGORY_COLOR = '#0d9488';

const router = useRouter();
const reading = useUserStore().aritcleModel;
const isDark = computed(() => reading.isDarkMode);
const reducedMotion = matchMedia('(prefers-reduced-motion: reduce)').matches;
const toggleTheme = (event: MouseEvent) => {
  revealThemeChange(event.currentTarget as HTMLElement, () => { reading.isDarkMode = !reading.isDarkMode; });
};

const allData = ref<Taxonomy[]>([]);
const isLoading = ref(true);
// Card or word under the pointer, as `${type}:${name}`; the starfield lights the matching star.
const activeKey = ref('');

const categories = computed(() => allData.value.filter(item => item.type === 'category').sort((a, b) => b.count - a.count));
const tags = computed(() => allData.value.filter(item => item.type === 'tag'));
const maxCategoryCount = computed(() => Math.max(1, ...categories.value.map(item => item.count)));
// Every article has exactly one category, so their counts add up to the article total.
const articleTotal = computed(() => categories.value.reduce((sum, item) => sum + item.count, 0));

// Bars grow from 0 once data arrives; set on the next frame so the width transition has a start value.
const barsReady = ref(false);

// Count the headline numbers up instead of showing them all at once.
const counted = reactive({ articles: 0, categories: 0, tags: 0 });
const countUp = () => {
  const targets = { articles: articleTotal.value, categories: categories.value.length, tags: tags.value.length };
  if (reducedMotion) {
    Object.assign(counted, targets);
    return;
  }
  const start = performance.now();
  const tick = (now: number) => {
    const progress = Math.min(1, (now - start) / 900);
    const eased = 1 - Math.pow(1 - progress, 3);
    counted.articles = Math.round(targets.articles * eased);
    counted.categories = Math.round(targets.categories * eased);
    counted.tags = Math.round(targets.tags * eased);
    if (progress < 1) requestAnimationFrame(tick);
  };
  requestAnimationFrame(tick);
};
const stats = computed(() => [
  { label: '篇文章', display: counted.articles, color: 'text-[#4f46e5] dark:text-[#a5b4fc]' },
  { label: '个分类', display: counted.categories, color: 'text-[#0d9488] dark:text-[#5eead4]' },
  { label: '个标签', display: counted.tags, color: 'text-[#c026d3] dark:text-[#f0abfc]' },
]);

const loadTaxonomies = async () => {
  isLoading.value = true;
  try {
    const data = await getAllTaxonomies();
    allData.value = (data ?? []).map(item => ({ ...item, key: `${item.type}:${item.name}` }));
  } catch (error) {
    console.error('获取标签和分类失败:', error);
  } finally {
    isLoading.value = false;
  }
  countUp();
  requestAnimationFrame(() => requestAnimationFrame(() => { barsReady.value = true; }));
};

const modal = reactive({
  visible: false,
  title: '',
  type: '',
  color: CATEGORY_COLOR,
  articles: null as TaxonomyArticle[] | null,
  failed: false,
});
const closeButtonRef = ref<HTMLButtonElement | null>(null);
let lastFocus: HTMLElement | null = null;
// Where the dialog flies out of (and back into): the clicked word or category row.
let modalOrigin: { x: number; y: number } | null = null;

const centreOf = (event: MouseEvent) => {
  const rect = (event.currentTarget as HTMLElement).getBoundingClientRect();
  return { x: rect.left + rect.width / 2, y: rect.top + rect.height / 2 };
};

// element.animate() ignores the global reduced-motion CSS; the overlay's own fade still runs there.
const flyFrames = (el: Element) => {
  const panel = (el as HTMLElement).querySelector<HTMLElement>('[role=dialog]');
  if (!panel || reducedMotion) return null;
  const rect = panel.getBoundingClientRect();
  const dx = modalOrigin ? modalOrigin.x - (rect.left + rect.width / 2) : 0;
  const dy = modalOrigin ? modalOrigin.y - (rect.top + rect.height / 2) : 40;
  return {
    panel,
    frames: [
      { transform: `translate(${dx}px, ${dy}px) scale(0.12)`, opacity: 0, filter: 'blur(6px)' },
      { transform: 'none', opacity: 1, filter: 'blur(0)' },
    ],
  };
};
const flyIn = (el: Element) => {
  const fly = flyFrames(el);
  fly?.panel.animate(fly.frames, { duration: 520, easing: 'cubic-bezier(0.2, 0.9, 0.25, 1.12)' });
};
const flyOut = (el: Element) => {
  const fly = flyFrames(el);
  fly?.panel.animate([...fly.frames].reverse(), { duration: 280, easing: 'cubic-bezier(0.5, 0, 0.75, 0)', fill: 'forwards' });
};
// Ignores a slow response that lands after the reader already opened another tag.
let modalRequest = 0;

const openModal = async (item: { name: string; type: string }, origin?: { x: number; y: number }) => {
  modalOrigin = origin ?? null;
  lastFocus = document.activeElement instanceof HTMLElement ? document.activeElement : null;
  const request = ++modalRequest;
  Object.assign(modal, {
    visible: true,
    title: item.name,
    type: item.type,
    color: item.type === 'tag' ? tagBaseColor(item.name) : CATEGORY_COLOR,
    articles: null,
    failed: false,
  });
  try {
    const articles = await getArticlesByTaxonomy(item.name, item.type);
    if (request === modalRequest) modal.articles = articles ?? [];
  } catch {
    if (request === modalRequest) modal.failed = true;
  }
};

const closeModal = () => {
  modal.visible = false;
};

watch(() => modal.visible, async visible => {
  document.body.style.overflow = visible ? 'hidden' : '';
  await nextTick();
  if (visible) closeButtonRef.value?.focus();
  else lastFocus?.focus();
});

const onKeydown = (event: KeyboardEvent) => {
  if (event.key === 'Escape' && modal.visible) closeModal();
};

const goToArticle = (id: number) => {
  const url = router.resolve({ name: 'ArticleInfo', params: { id } }).href;
  window.open(url, '_blank');
};

onMounted(() => {
  window.addEventListener('keydown', onKeydown);
  loadTaxonomies();
});

onBeforeUnmount(() => {
  window.removeEventListener('keydown', onKeydown);
  document.body.style.overflow = '';
});
</script>
