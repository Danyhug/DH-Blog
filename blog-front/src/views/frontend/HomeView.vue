<template>
  <div :inert="store.aritcleModel.isFullPreview">
    <Header ref="headerElement" />
    <Banner ref="bannerElement">
      <template v-if="sideShowComponent.__name == 'HomeSide'">
        <h1 class="text-[3.5em]">我的个人纪录</h1>
        <h2 class="text-[2.5em] [text-shadow:rgba(0,0,0,0.5)_0rem_0.2rem_0.3rem]">DH-BLOG</h2>
      </template>
      <!-- 文章接口回来之前 store 里还是上一篇（或空）的信息，先显示骨架条，避免旧标题闪一下再换 -->
      <template v-else-if="isCurrentArticle">
        <h3 class="text-[2.2em] [text-shadow:rgba(0,0,0,0.5)_0rem_0.2rem_0.3rem] tracking-[0.125rem] animate-fade-in">{{ store.homeHeaderInfo.title }}</h3>
        <div class="top mt-[18px] animate-fade-in">
          <span class="date mr-5">发表于 {{ store.homeHeaderInfo.created }}</span>
          <span class="num-word mr-5">本文字数 {{ store.homeHeaderInfo.wordNum }} 字</span>
          <span class="time-consum mr-5">阅读时长 {{ store.homeHeaderInfo.timConSum }} 分钟</span>
        </div>
      </template>
      <div v-else class="flex flex-col items-center" aria-busy="true" aria-label="文章信息加载中">
        <div class="h-[1.6em] w-[min(18em,70vw)] rounded-md bg-white/25 animate-pulse text-[2.2em]"></div>
        <div class="mt-[18px] h-[1.2em] w-[min(26em,80vw)] rounded-md bg-white/20 animate-pulse"></div>
      </div>
    </Banner>
    <div>
      <div class="inner px-[25px] flex justify-between">
        <div class="left bg-white dark:bg-[#191919] flex flex-col items-center w-[30%] h-screen text-center sticky top-0 my-[9.6px]" ref="leftElement">
          <FadeTransition>
            <!-- key 跟随路由：文章间跳转（同一路由只换 :id）时重建侧栏，阅读时长/进度/目录随之重置 -->
            <component :is="sideShowComponent" :key="route.path" />
          </FadeTransition>
        </div>
        <div class="right bg-white dark:bg-[#191919] dark:shadow-none w-[67%] shadow-[0_0.5rem_0.75rem_0.0625rem_rgb(235,235,235)] rounded-[0.3125rem] my-[9.6px] pt-4 flex flex-col justify-between">
          <router-view v-slot="{ Component }">
            <FadeTransition>
              <!-- 同上：否则 /view/article/1 → /2 会复用 ArticleView 实例，mounted 不再执行，文章不刷新 -->
              <component :is="Component" :key="route.path" />
            </FadeTransition>
          </router-view>
        </div>
      </div>

      <Footer />
    </div>
  </div>
</template>

<script setup lang="ts">
import Header from '@/components/frontend/Header.vue';
import Banner from '@/components/frontend/Banner.vue';
import HomeSide from '@/components/frontend/Side/HomeSide.vue';
import ArticleInfoSide from '@/components/frontend/Side/ArticleInfoSide.vue';
import Footer from '@/components/frontend/Footer.vue';
import FadeTransition from '@/components/Child/FadeTransition.vue';

import { computed, shallowRef, watch } from 'vue';

import { useUserStore } from '@/store/index'
import { useRoute } from 'vue-router';

const sideShowComponent = shallowRef<any>(HomeSide);

const store = useUserStore();
const route = useRoute()

// homeHeaderInfo is filled by ArticleView once the article request returns; until then it still holds
// the previous article (or nothing), so only show it when it belongs to the article in the URL.
const isCurrentArticle = computed(() => String(store.homeHeaderInfo.id) === String(route.params.id))

if (route.path == '/view/home') {
  sideShowComponent.value = HomeSide;
  store.homeShowComponent = 'home'
} else {
  sideShowComponent.value = ArticleInfoSide;
  store.homeShowComponent = 'articleInfoSide'
}

watch(() => route.path, _ => {
  if (route.path == '/view/home') {
    sideShowComponent.value = HomeSide;
    store.homeShowComponent = 'home'
  } else {
    sideShowComponent.value = ArticleInfoSide;
    store.homeShowComponent = 'articleInfoSide'
  }
})

</script>

<style scoped>
/* 字重：style.less 全局 h1,h2,h3{font-weight:normal} 是无层级规则；scoped 的 h3 特异性更高本来能赢，换成工具类就会输 */
h3 {
  font-weight: bold;
}
</style>
