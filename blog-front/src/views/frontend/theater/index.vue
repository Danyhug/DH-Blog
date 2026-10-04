<template>
  <!-- 个人影院外壳：影视（Netflix 风格）与音乐（Apple Music 风格）两个页面共用。
       音乐播放器挂在这一层，所以在两个页面之间切换时音乐不会断。
       页面本身都是透明的，叠在氛围背景上，玻璃材质才有东西可以模糊。
       主要在电视上用遥控器操作：方向键移动焦点（utils/tv.ts），所以焦点框要醒目，隔着几米也看得见。 -->
  <div
    data-nav-scope
    class="relative isolate min-h-screen text-zinc-900 antialiased dark:text-white [&_:focus-visible]:outline-[3px] [&_:focus-visible]:outline-offset-[3px] [&_:focus-visible]:outline-[#fa2d48] [&_:focus-visible]:outline-solid"
  >
    <AmbientBackdrop :image="ambient" :mode="route.name === 'TheaterMusic' ? 'music' : 'cinema'" />

    <div class="relative z-[1]">
      <router-view v-slot="{ Component }">
        <keep-alive>
          <component :is="Component" />
        </keep-alive>
      </router-view>

      <MusicPlayerBar v-if="player.current" />
      <transition
        enter-active-class="transition duration-300 ease-out"
        enter-from-class="translate-y-full opacity-0"
        leave-active-class="transition duration-200 ease-in"
        leave-to-class="translate-y-full opacity-0"
      >
        <NowPlaying v-if="player.expanded && player.current" />
      </transition>
    </div>

    <LibrarySettingsDialog v-model="settingsOpen" @saved="libraryVersion++" />
  </div>
</template>

<script setup lang="ts">
import { computed, onUnmounted, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import { useMusicPlayerStore } from '@/store'
import { provideTheater } from './context'
import { tvMode, useHistoryLayer, useSpatialNavigation } from './utils/tv'
import AmbientBackdrop from './components/AmbientBackdrop.vue'
import LibrarySettingsDialog from './components/LibrarySettingsDialog.vue'
import MusicPlayerBar from './music/PlayerBar.vue'
import NowPlaying from './music/NowPlaying.vue'

const route = useRoute()
const player = useMusicPlayerStore()
const settingsOpen = ref(false)
const libraryVersion = ref(0)
const ambient = ref('')

provideTheater({
  openSettings: () => (settingsOpen.value = true),
  libraryVersion,
  ambient
})

useSpatialNavigation()
// 遥控器的返回键先收起「正在播放」，而不是直接离开影院
useHistoryLayer('playing', computed(() => player.expanded && !!player.current), () => (player.expanded = false))

// tv 类挂在 <html> 上：Element Plus 的弹层被传送到 body 下，也要跟着去掉毛玻璃
watch(tvMode, on => document.documentElement.classList.toggle('tv', on), { immediate: true })

// 离开影院（回网盘/博客）时停掉音乐：播放条只在影院里渲染，继续出声就没地方暂停了
onUnmounted(() => {
  player.pause()
  player.expanded = false
  document.documentElement.classList.remove('tv')
})
</script>
