<template>
  <div>
    <Header ref="headerElement" />
    <Banner ref="bannerElement">
      <template v-if="sideShowComponent.__name == 'HomeSide'">
        <h1 class="text-[3.5em]">我的个人纪录</h1>
        <h2 class="text-[2.5em] [text-shadow:rgba(0,0,0,0.5)_0rem_0.2rem_0.3rem]">DH-BLOG</h2>
      </template>
      <template v-else>
        <h3 class="text-[2.2em] [text-shadow:rgba(0,0,0,0.5)_0rem_0.2rem_0.3rem] tracking-[0.125rem]">{{ store.homeHeaderInfo.title }}</h3>
        <div class="top mt-[18px]">
          <span class="date mr-5">发表于 {{ store.homeHeaderInfo.created }}</span>
          <span class="num-word mr-5">本文字数 {{ store.homeHeaderInfo.wordNum }} 字</span>
          <span class="time-consum mr-5">阅读时长 {{ store.homeHeaderInfo.timConSum }} 分钟</span>
        </div>
      </template>
    </Banner>
    <div class="inner px-[25px] flex justify-between">
      <div class="left bg-white flex flex-col items-center w-[30%] h-screen text-center sticky top-0 my-[9.6px]" ref="leftElement">
        <transition mode="out-in">
          <component :is="sideShowComponent" />
        </transition>
      </div>
      <div class="right bg-white w-[67%] shadow-[0_0.5rem_0.75rem_0.0625rem_rgb(235,235,235)] rounded-[0.3125rem] my-[9.6px] pt-4 flex flex-col justify-between">
        <transition mode="out-in">
          <router-view />
        </transition>
      </div>
    </div>

    <Footer />
  </div>
</template>

<script setup lang="ts">
import Header from '@/components/frontend/Header.vue';
import Banner from '@/components/frontend/Banner.vue';
import HomeSide from '@/components/frontend/Side/HomeSide.vue';
import ArticleInfoSide from '@/components/frontend/Side/ArticleInfoSide.vue';
import Footer from '@/components/frontend/Footer.vue';

import { shallowRef, watch } from 'vue';

import { useUserStore } from '@/store/index'
import { useRoute } from 'vue-router';

const sideShowComponent = shallowRef<any>(HomeSide);

const store = useUserStore();
const route = useRoute()

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
/*
  §4.8：style.less 的 `h1, h2, h3 { font-weight: normal }` 是无层级规则。
  `h3 { font-weight: bold }`（scoped，特异性 0,1,1 > 0,0,1）本来赢过它；
  换成 `font-bold` 工具类（在 @layer utilities 里）就会输给它，所以这一条必须留在这里。
*/
h3 {
  font-weight: bold;
}

/* Vue <transition> 运行时生成的类，模板里静态写不出来 */
.v-enter-from,
.v-leave-to {
  opacity: 0;
  transform: scale(.5);
}

.v-leave-from,
.v-enter-to {
  opacity: 1;
  transform: scale(1);
}

.v-enter-active,
.v-leave-active {
  transition: all .6s ease;
}
</style>
