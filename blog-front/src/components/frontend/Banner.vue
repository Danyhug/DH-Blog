<template>
  <div id="banner" ref="banner"
    class="relative z-[-999] w-full h-[80vh] overflow-hidden bg-[#2b3590] dark:bg-[#151a40] filter-[contrast(88%)] dark:filter-none flex justify-center items-center">
    <!-- 原图 500KB+，慢网下会一行行往下画。先铺一张 1KB 的缩略图（小于 Vite 内联阈值，随 CSS 一起到）模糊放大顶住，
         原图完整下载后再整张淡入盖上去 -->
    <div class="absolute inset-0 bg-[url(@/assets/images/banner-placeholder.jpg)] bg-cover bg-center blur-2xl scale-110 dark:brightness-60"></div>
    <img ref="bannerImage" :src="bannerUrl" alt="" fetchpriority="high" decoding="async"
      class="absolute inset-0 size-full object-cover transition-opacity duration-700 ease-out dark:brightness-60 dark:saturate-80"
      :class="bannerLoaded ? 'opacity-100' : 'opacity-0'" @load="bannerLoaded = true">
    <!-- 夜间：图片压暗后，底部再渐隐到页面底色（index.html 的 body #141414），避免亮色横幅与下方深色内容硬切。
         夜间去掉容器的 contrast 滤镜，否则它会把这里的 #141414 提亮成 #212121，和 body 对不上 -->
    <div class="absolute inset-0 hidden dark:block bg-linear-to-b from-black/10 via-transparent via-55% to-[#141414]"></div>
    <!-- 文章页压暗一层，让标题更醒目。从列表深处点进来时横幅还在视口外，等页面滚回顶部、横幅进入视口才开始变暗，
         否则过渡在看不见的地方就播完了 -->
    <div class="absolute inset-0 bg-black pointer-events-none transition-opacity duration-700 ease-out motion-reduce:transition-none"
      :class="dimmed ? 'opacity-30' : 'opacity-0'"></div>
    <div class="z-[2] text-center leading-[1.2] text-white dark:text-[#e8e8e8] [font-family:'Fredericka_the_Great',Mulish,-apple-system,'PingFang_SC','Microsoft_YaHei',sans-serif]">
      <slot></slot>
    </div>
  </div>
</template>

<script setup>
import { onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useUserStore } from '@/store/index'
import bannerUrl from '@/assets/images/banner.png'

const store = useUserStore();
const banner = ref(null)
const bannerImage = ref(null)
const bannerLoaded = ref(false)

const isArticle = () => store.homeShowComponent == 'articleInfoSide'
// Latched once the banner is seen on an article page, so scrolling away and back doesn't replay it;
// only returning home clears it.
const dimmed = ref(false)
const inView = ref(false)
const syncDimmed = () => {
  if (!isArticle()) dimmed.value = false
  else if (inView.value) dimmed.value = true
}

// Mostly visible: the fade starts once the smooth scroll back to top has all but arrived.
const observer = new IntersectionObserver(([entry]) => {
  inView.value = entry.isIntersecting
  syncDimmed()
}, { threshold: 0.8 })

watch(() => store.homeShowComponent, syncDimmed)

onMounted(() => {
  // 已在缓存里时直接显示，不必再等一轮淡入
  if (bannerImage.value?.complete && bannerImage.value.naturalWidth) bannerLoaded.value = true
  observer.observe(banner.value)
})

onBeforeUnmount(() => observer.disconnect())

</script>
