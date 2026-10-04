<template>
  <!-- 影视（Netflix 风格）：顶部导航 + 大幅海报 + 一行行横向片单 -->
  <div class="min-h-screen pb-16 text-zinc-900 dark:text-white" :class="musicPlayer.current && 'pb-32'">
    <!-- ===== 顶部导航：悬浮的液态玻璃胶囊，滚动后玻璃加厚 ===== -->
    <header class="fixed inset-x-0 top-0 z-50 px-[3%] pt-4 max-md:px-3 max-md:pt-2">
      <div
        class="flex h-14 items-center gap-8 rounded-full px-3 pl-6 transition-all duration-500 max-lg:gap-4 max-md:h-12 max-md:pl-4"
        :class="scrolled || tab !== 'home' || keyword ? 'glass-thick' : 'glass'"
      >
        <button class="cursor-pointer border-none bg-transparent p-0 text-[24px] font-black tracking-tight text-[#e50914] drop-shadow-[0_0_12px_rgba(229,9,20,0.55)] max-md:text-lg" @click="setTab('home')">
          DH影院
        </button>
        <!-- 分段标签：选中项是一块更亮的玻璃 -->
        <nav class="flex items-center gap-1 rounded-full bg-black/5 p-1 dark:bg-black/20 text-sm max-md:hidden">
          <button
            v-for="item in tabs"
            :key="item.value"
            :class="['cursor-pointer rounded-full px-4 py-1.5 transition-all duration-300', tab === item.value && !keyword ? 'glass-button font-semibold text-zinc-900 dark:text-white' : 'border border-transparent bg-transparent text-black/70 dark:text-white/70 hover:text-zinc-900 dark:hover:text-white']"
            @click="setTab(item.value)"
          >
            {{ item.label }}
          </button>
          <router-link :to="{ name: 'TheaterMusic' }" class="flex items-center gap-1.5 rounded-full px-4 py-1.5 text-black/70! dark:text-white/70! no-underline transition-colors hover:text-zinc-900! dark:hover:text-white!">
            <MusicNoteIcon class="size-4" />音乐
          </router-link>
        </nav>
        <!-- 窄屏：标签收进下拉 -->
        <el-dropdown class="md:hidden" trigger="click" popper-class="glass-popup">
          <button class="flex cursor-pointer items-center gap-1 border-none bg-transparent p-0 text-sm text-zinc-900 dark:text-white">
            {{ tabs.find(item => item.value === tab)?.label }}<ChevronDownIcon class="size-4" />
          </button>
          <template #dropdown>
            <el-dropdown-menu>
              <el-dropdown-item v-for="item in tabs" :key="item.value" @click="setTab(item.value)">{{ item.label }}</el-dropdown-item>
              <el-dropdown-item divided @click="router.push({ name: 'TheaterMusic' })">音乐</el-dropdown-item>
            </el-dropdown-menu>
          </template>
        </el-dropdown>

        <div class="ml-auto flex items-center gap-2">
          <div class="flex items-center rounded-full transition-all duration-300" :class="searchOpen ? 'glass-button pr-3' : ''">
            <button :class="navIcon" title="搜索" @click="openSearch"><SearchIcon class="size-[18px]" /></button>
            <input
              v-if="searchOpen"
              ref="searchRef"
              v-model="keyword"
              placeholder="片名、剧集、文件名"
              class="w-52 border-none bg-transparent py-1.5 text-sm text-zinc-900 dark:text-white outline-none placeholder:text-black/50 dark:placeholder:text-white/50 max-md:w-24"
              @blur="!keyword && (searchOpen = false)"
              @keyup.esc="keyword = ''; searchOpen = false"
            />
          </div>
          <ThemeToggle :class="navIcon" />
          <button :class="navIcon" title="媒体库设置" @click="theater.openSettings()"><SettingsIcon class="size-[18px]" /></button>
          <router-link to="/webdav" :class="navIcon" title="返回网盘"><FolderIcon class="size-[18px]" /></router-link>
        </div>
      </div>
    </header>

    <!-- 加载中 -->
    <div v-if="loading && !videos.length" class="flex h-screen flex-col items-center justify-center gap-4">
      <span class="glass flex size-20 items-center justify-center rounded-3xl"><span class="size-9 animate-spin rounded-full border-[3px] border-black/10 dark:border-white/15 border-t-[#e50914]"></span></span>
      <p class="m-0 text-sm text-black/50 dark:text-white/50">正在整理你的片库…</p>
    </div>

    <!-- 加载失败 -->
    <div v-else-if="loadFailed && !videos.length" class="flex h-screen flex-col items-center justify-center gap-4">
      <p class="m-0 text-lg">片库加载失败</p>
      <button :class="whiteButton" @click="load">重试</button>
    </div>

    <!-- 空库 -->
    <div v-else-if="!videos.length" class="flex h-screen flex-col items-center justify-center gap-4 px-6 text-center">
      <span class="glass flex size-24 items-center justify-center rounded-[28px]"><FilmIcon class="size-11 text-[#e50914]" /></span>
      <h2 class="m-0 text-3xl font-bold">片库还是空的</h2>
      <p class="m-0 max-w-lg text-[15px] leading-relaxed text-black/60 dark:text-white/60">
        {{ scoped ? '指定的影视文件夹里没有找到视频。' : '把电影、剧集上传到网盘任意位置即可。同一文件夹里带「S01E02 / 第2集」的视频会自动合成一部剧；同名的 poster.jpg 与 .srt 字幕也会被识别。' }}
      </p>
      <MissingNotice :missing="missing" kind="视频" class="mt-2 max-w-xl" />
      <div class="mt-2 flex gap-3">
        <router-link to="/webdav" :class="whiteButton" class="no-underline">去网盘上传</router-link>
        <button :class="greyButton" @click="theater.openSettings()">媒体库设置</button>
      </div>
    </div>

    <!-- 搜索结果 -->
    <section v-else-if="keyword.trim()" class="px-[4%] pt-28 max-md:pt-20">
      <p class="m-0 mb-6 text-[#808080]">搜索「<span class="text-zinc-900 dark:text-white">{{ keyword.trim() }}</span>」的结果</p>
      <p v-if="!searchResults.length" class="py-16 text-center text-black/60 dark:text-white/60">没有找到匹配的影片或剧集</p>
      <div :class="gridClass">
        <TitleCard v-for="item in searchResults" :key="item.key" :title="item" :in-row="false" @open="selected = item" @play="playTitle(item)" />
      </div>
    </section>

    <!-- 剧集 / 电影 / 最近添加：网格 -->
    <section v-else-if="tab !== 'home'" class="px-[4%] pt-28 max-md:pt-20">
      <h1 class="m-0 mb-8 text-[38px] font-bold max-md:text-2xl">{{ tabs.find(item => item.value === tab)?.label }}</h1>
      <p v-if="!tabTitles.length" class="py-16 text-center text-black/60 dark:text-white/60">这里还没有内容</p>
      <div :class="gridClass">
        <TitleCard v-for="item in tabTitles" :key="item.key" :title="item" :in-row="false" @open="selected = item" @play="playTitle(item)" />
      </div>
    </section>

    <!-- 首页 -->
    <template v-else>
      <section v-if="featured" class="relative h-[56.25vw] max-h-[88vh] min-h-[480px] overflow-hidden">
        <VideoThumb :image-id="featured.imageId" :frame-id="featured.videos[0].id" :seed="featured.name" class="absolute inset-0 size-full" />
        <!-- 底部融进氛围背景，而不是切到一块纯色 -->
        <div class="absolute inset-0 bg-gradient-to-r from-black/70 via-black/20 to-transparent"></div>
        <div class="absolute inset-x-0 bottom-0 h-[45%] bg-gradient-to-t from-[#f5f5f7] via-[#f5f5f7]/60 to-transparent dark:from-[#07070a] dark:via-[#07070a]/60"></div>
        <!-- 作品信息放在一块液态玻璃卡片上；卡片压在剧照上，日夜都用深色玻璃与白字 -->
        <div class="dark glass absolute text-white bottom-[28%] left-[4%] w-[min(560px,44%)] rounded-[28px] p-7 max-lg:bottom-[20%] max-lg:w-[70%] max-md:bottom-[16%] max-md:left-3 max-md:w-[calc(100%-24px)] max-md:p-5">
          <p v-if="featured.kind === 'series'" class="m-0 mb-3 flex items-center gap-2 text-xs font-semibold tracking-[0.3em] text-black/75 dark:text-white/75">
            <span class="text-lg font-black tracking-normal text-[#e50914]">N</span>剧集
          </p>
          <h1 class="m-0 text-[clamp(28px,3.8vw,64px)] font-black leading-[0.95] tracking-tight drop-shadow-2xl">{{ featured.name }}</h1>
          <p class="m-0 mt-4 flex flex-wrap items-center gap-3 text-[clamp(13px,1vw,16px)] text-black/85 dark:text-white/85">
            <span v-if="featured.year" class="font-semibold text-green-600 dark:text-[#46d369]">{{ featured.year }}</span>
            <span v-if="featured.kind === 'series'">{{ featured.videos.length }} 集</span>
            <span v-else-if="featured.videos[0].duration">{{ formatRuntime(featured.videos[0].duration) }}</span>
            <span class="truncate text-black/55 dark:text-white/55">{{ featured.videos[0].folder_path || '我的网盘' }}</span>
          </p>
          <div class="mt-6 flex gap-3">
            <button :class="whiteButton" @click="playTitle(featured)"><PlayIcon class="size-5" />{{ heroPlayLabel }}</button>
            <button :class="greyButton" @click="selected = featured"><InfoIcon class="size-5" />更多信息</button>
          </div>
        </div>
      </section>

      <div class="relative z-10" :class="featured ? '-mt-[9vw] max-lg:-mt-16' : 'pt-24'">
        <VideoRow v-if="continueRow.length" title="继续观看">
          <TitleCard
            v-for="item in continueRow"
            :key="item.video.id"
            :title="item.title"
            :video="item.video"
            removable
            @open="selected = item.title"
            @play="openPlayer(item.video, item.title)"
            @remove="removeFromContinue(item.title)"
          />
        </VideoRow>
        <VideoRow v-if="recentTitles.length" title="最近添加">
          <TitleCard v-for="item in recentTitles" :key="item.key" :title="item" @open="selected = item" @play="playTitle(item)" />
        </VideoRow>
        <VideoRow v-if="seriesTitles.length" title="剧集">
          <TitleCard v-for="item in seriesTitles" :key="item.key" :title="item" @open="selected = item" @play="playTitle(item)" />
        </VideoRow>
        <VideoRow v-for="row in folderRows" :key="row.name" :title="row.name">
          <TitleCard v-for="item in row.titles" :key="item.key" :title="item" @open="selected = item" @play="playTitle(item)" />
        </VideoRow>
        <MissingNotice :missing="missing" kind="视频" class="mx-[4%] mt-4" />
      </div>
    </template>

    <TitleDetail
      v-if="selected"
      :key="selected.key"
      :title="selected"
      @close="selected = null"
      @play="(video, fromStart) => openPlayer(video, selected!, fromStart ? 0 : undefined)"
      @remove-progress="removeFromContinue"
    />

    <VideoPlayer
      v-if="session"
      :video="session.video"
      :title="session.title"
      :start-at="session.startAt"
      @close="session = null"
      @play="video => openPlayer(video, session!.title)"
      @progress="(video, progress) => (video.progress = progress)"
    />
  </div>
</template>

<script setup lang="ts">
import { computed, nextTick, onActivated, onDeactivated, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useMusicPlayerStore } from '@/store'
import { getVideoLibrary, mediaStreamUrl, removeProgress, type MissingMedia, type Video } from '@/api/media'
import { captureFrame } from './frameCapture'
import { notify } from '@/utils/notification'
import { useTheater } from '../context'
import TitleCard from './TitleCard.vue'
import TitleDetail from './TitleDetail.vue'
import VideoPlayer from './VideoPlayer.vue'
import VideoRow from './VideoRow.vue'
import VideoThumb from './VideoThumb.vue'
import MissingNotice from '../components/MissingNotice.vue'
import ThemeToggle from '@/components/Child/ThemeToggle.vue'
import { buildTitles, continueWatching, matchesTitle, resumeTarget, type Title } from './catalog'
import { formatRuntime } from '../utils/format'
import {
  ChevronDownIcon, FilmIcon, FolderIcon, InfoIcon, MusicNoteIcon, PlayIcon, SearchIcon, SettingsIcon
} from '../components/icons'

type Tab = 'home' | 'series' | 'movies' | 'recent'

const route = useRoute()
const router = useRouter()
const theater = useTheater()
const musicPlayer = useMusicPlayerStore()

// 日间是深色实心胶囊，夜间（含挂了 dark 的海报卡片）是 Netflix 的白色播放键
const whiteButton = 'flex cursor-pointer items-center gap-2 rounded-full border-none bg-zinc-900 px-7 py-2.5 text-base font-bold text-white! no-underline shadow-[0_8px_24px_-8px_rgba(0,0,0,0.45)] transition-all hover:bg-black active:scale-95 dark:bg-white/95 dark:text-black! dark:shadow-[0_8px_30px_rgba(255,255,255,0.25),inset_0_1px_0_white] dark:hover:bg-white max-md:px-5 max-md:py-2 max-md:text-sm'
const greyButton = 'glass-button flex cursor-pointer items-center gap-2 rounded-full px-7 py-2.5 text-base font-bold text-zinc-900 dark:text-white max-md:px-5 max-md:py-2 max-md:text-sm'
const navIcon = 'flex size-9 cursor-pointer items-center justify-center rounded-full border-none bg-transparent text-zinc-900! dark:text-white! transition-colors hover:bg-black/10 dark:hover:bg-white/15'
const gridClass = 'grid grid-cols-[repeat(auto-fill,minmax(250px,1fr))] gap-x-2 gap-y-8 max-md:grid-cols-2 max-md:gap-y-4'

const tabs: { value: Tab; label: string }[] = [
  { value: 'home', label: '首页' },
  { value: 'series', label: '剧集' },
  { value: 'movies', label: '电影' },
  { value: 'recent', label: '最近添加' }
]
const tab = computed<Tab>(() => (tabs.some(item => item.value === route.query.tab) ? (route.query.tab as Tab) : 'home'))

function setTab(value: Tab) {
  keyword.value = ''
  router.push({ name: 'TheaterVideos', query: value === 'home' ? {} : { tab: value } })
  window.scrollTo({ top: 0 })
}

// ---- 数据 ----
const videos = ref<Video[]>([])
const scoped = ref(false)
const missing = ref<MissingMedia>({ count: 0, samples: [] })
const loading = ref(false)
const loadFailed = ref(false)

async function load() {
  loading.value = true
  loadFailed.value = false
  try {
    const library = await getVideoLibrary()
    videos.value = library.videos
    scoped.value = library.scoped
    missing.value = library.missing ?? { count: 0, samples: [] }
    pickFeatured()
  } catch {
    loadFailed.value = true
  } finally {
    loading.value = false
  }
}

const titles = computed(() => buildTitles(videos.value))
const continueRow = computed(() => continueWatching(titles.value))
const recentTitles = computed(() => titles.value.slice().sort((a, b) => b.addedAt - a.addedAt).slice(0, 24))
const seriesTitles = computed(() => titles.value.filter(item => item.kind === 'series'))
const movieTitles = computed(() => titles.value.filter(item => item.kind === 'movie'))

// 电影按所在文件夹分行（「纪录片」「动画电影」…）；零散的归到「电影」一行
const folderRows = computed(() => {
  const byFolder = new Map<string, Title[]>()
  const loose: Title[] = []
  for (const item of movieTitles.value) {
    if (!item.collectionId) {
      loose.push(item)
      continue
    }
    const group = byFolder.get(item.collectionId)
    if (group) group.push(item)
    else byFolder.set(item.collectionId, [item])
  }
  const rows: { name: string; titles: Title[] }[] = []
  for (const group of byFolder.values()) {
    if (group.length >= 2) rows.push({ name: group[0].collectionName, titles: group })
    else loose.push(...group)
  }
  rows.sort((a, b) => b.titles.length - a.titles.length)
  if (loose.length) rows.push({ name: rows.length ? '更多电影' : '电影', titles: loose })
  return rows
})

const tabTitles = computed(() => {
  if (tab.value === 'series') return seriesTitles.value
  if (tab.value === 'movies') return movieTitles.value
  return titles.value.slice().sort((a, b) => b.addedAt - a.addedAt)
})

// 首页大图：优先挑有剧照/海报的作品，每次进入片库随机一部
const featuredKey = ref('')
const featured = computed(() => titles.value.find(item => item.key === featuredKey.value) ?? titles.value[0])
function pickFeatured() {
  const withImage = titles.value.filter(item => item.imageId)
  const pool = withImage.length ? withImage : titles.value
  featuredKey.value = pool.length ? pool[Math.floor(Math.random() * pool.length)].key : ''
}
const heroPlayLabel = computed(() => {
  const item = featured.value
  if (!item) return '播放'
  const target = resumeTarget(item)
  return target.progress && !target.progress.finished && target.progress.position > 5 ? '继续播放' : '播放'
})

// ---- 搜索 ----
const keyword = ref('')
const searchOpen = ref(false)
const searchRef = ref<HTMLInputElement | null>(null)
const searchResults = computed(() => {
  const k = keyword.value.trim()
  return k ? titles.value.filter(item => matchesTitle(item, k)) : []
})

async function openSearch() {
  searchOpen.value = true
  await nextTick()
  searchRef.value?.focus()
}

// ---- 详情与播放 ----
const selected = ref<Title | null>(null)
const session = ref<{ video: Video; title: Title; startAt?: number } | null>(null)

function openPlayer(video: Video, title: Title, startAt?: number) {
  // 影片和音乐不该同时出声
  musicPlayer.pause()
  selected.value = null
  session.value = { video, title, startAt }
}

function playTitle(title: Title) {
  openPlayer(resumeTarget(title), title)
}

async function removeFromContinue(title: Title) {
  const watched = title.videos.filter(video => video.progress)
  await Promise.all(watched.map(video => removeProgress(video.id)))
  watched.forEach(video => (video.progress = undefined))
  notify.success(`已清除「${title.name}」的观看记录`)
}

// 从网盘预览页「在影院中播放」跳过来：?play=<文件ID>
watch([videos, () => route.query.play], ([list, play]) => {
  if (!play || !list.length) return
  const target = list.find(video => video.id === String(play))
  if (target) {
    const title = titles.value.find(item => item.videos.some(video => video.id === target.id))
    if (title) openPlayer(target, title)
  } else {
    notify.warning('该视频不在影视库中，可能被媒体库设置排除了')
  }
  router.replace({ name: 'TheaterVideos', query: { ...route.query, play: undefined } })
})

// ---- 导航栏随滚动变实色 ----
const scrolled = ref(false)
function onScroll() {
  scrolled.value = window.scrollY > 10
}

// ---- 氛围背景：推荐作品的剧照（没有就截一帧），被外壳大幅模糊后铺满整页 ----
const active = ref(false)
async function updateAmbient() {
  if (!active.value) return
  const item = featured.value
  if (!item) return void (theater.ambient.value = '')
  const url = item.imageId ? mediaStreamUrl(item.imageId) : await captureFrame(item.videos[0].id)
  if (active.value && featured.value?.key === item.key) theater.ambient.value = url
}
watch(featured, updateAmbient)

onMounted(load)
onActivated(() => {
  active.value = true
  updateAmbient()
  window.addEventListener('scroll', onScroll, { passive: true })
})
onDeactivated(() => {
  active.value = false
  window.removeEventListener('scroll', onScroll)
})
watch(theater.libraryVersion, load)
</script>
