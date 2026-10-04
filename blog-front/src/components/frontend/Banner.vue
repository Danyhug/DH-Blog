<template>
  <div id="banner" ref="banner"
    class="relative z-[-999] w-full h-[80vh] overflow-hidden bg-[#2b3590] filter-[contrast(88%)] flex justify-center items-center">
    <!-- 原图 500KB+，慢网下会一行行往下画。先铺一张 1KB 的缩略图（小于 Vite 内联阈值，随 CSS 一起到）模糊放大顶住，
         原图完整下载后再整张淡入盖上去 -->
    <div class="absolute inset-0 bg-[url(@/assets/images/banner-placeholder.jpg)] bg-cover bg-center blur-2xl scale-110"></div>
    <img ref="bannerImage" :src="bannerUrl" alt="" fetchpriority="high" decoding="async"
      class="absolute inset-0 size-full object-cover transition-opacity duration-700 ease-out"
      :class="bannerLoaded ? 'opacity-100' : 'opacity-0'" @load="bannerLoaded = true">
    <div class="z-[2] text-center leading-[1.2] text-white [font-family:'Fredericka_the_Great',Mulish,-apple-system,'PingFang_SC','Microsoft_YaHei',sans-serif]">
      <slot></slot>
    </div>
  </div>
</template>

<script setup>
import { onMounted, ref, watch } from 'vue'
import { useUserStore } from '@/store/index'
import bannerUrl from '@/assets/images/banner.png'

const store = useUserStore();
const banner = ref(null)
const bannerImage = ref(null)
const bannerLoaded = ref(false)

function articleAnimate() {
  if (store.homeShowComponent == 'articleInfoSide') {
    banner.value.classList.add('fade-in-article')
  } else if (store.homeShowComponent == 'home') {
    banner.value.classList.remove('fade-in-article')
  }
}

onMounted(() => {
  // 已在缓存里时直接显示，不必再等一轮淡入
  if (bannerImage.value?.complete && bannerImage.value.naturalWidth) bannerLoaded.value = true
  articleAnimate()
  watch(() => store.homeShowComponent, _ => articleAnimate())
})

</script>

<style lang="less" scoped>
/* 由 JS 运行时 add/remove；animation 引用本块 @keyframes fadeIn（Vue 会把两者一起改名，拆开即失效） */
.fade-in-article {
  &::after {
    content: "";
    position: absolute;
    top: 0;
    left: 0;
    width: 100%;
    height: 100%;
    background-color: rgb(0, 0, 0);
    opacity: 0;
    animation: fadeIn 1s ease-in-out forwards;
    animation-delay: 1s;
  }

  @keyframes fadeIn {
    from {
      opacity: 0;
    }

    to {
      opacity: .3;
    }
  }
}
</style>
