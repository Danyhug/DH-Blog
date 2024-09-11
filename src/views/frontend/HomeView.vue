<template>
  <div class="container">
    <Header ref="headerElement" />
    <Banner ref="bannerElement">
      <template v-if="sideShowComponent.name == 'HomeSide'">
        <h1>我的个人纪录</h1>
        <h2>DH-BLOG</h2>
      </template>
      <template v-else>
        <h3>{{ store.homeHeaderInfo.title }}</h3>
        <div class="top">
          <span class="date">发表于 {{ store.homeHeaderInfo.created }}</span>
          <span class="num-word">本文字数 {{ store.homeHeaderInfo.wordNum }} 字</span>
          <span class="time-consum">阅读时长 {{ store.homeHeaderInfo.timConSum }} 分钟</span>
        </div>
      </template>
    </Banner>
    <div class="inner">
      <div class="left" ref="leftElement">
        <transition mode="out-in">
          <component :is="sideShowComponent" />
        </transition>
      </div>
      <div class="right">
        <transition mode="out-in">
          <router-view />
        </transition>
      </div>
    </div>
    <Footer />
  </div>
</template>

<script setup lang="ts">
import '@/assets/css/style.less'
import Header from '@/components/frontend/Header.vue';
import Banner from '@/components/frontend/Banner.vue';
import HomeSide from '@/components/frontend/Side/HomeSide.vue';
import ArticleInfoSide from '@/components/frontend/Side/ArticleInfoSide.vue';
import Footer from '@/components/frontend/Footer.vue';
import { shallowRef } from 'vue';

import { useUserStore } from '@/store/index'
import { useRouter, useRoute } from 'vue-router';

const sideShowComponent = shallowRef(HomeSide);

const store = useUserStore();
const router = useRouter()
const route = useRoute()

if (route.path == '/view/home') {
  sideShowComponent.value = HomeSide;
  store.homeShowComponent = 'home'
} else {
  sideShowComponent.value = ArticleInfoSide;
  store.homeShowComponent = 'articleInfoSide'
}
router.beforeEach((_, __, next) => {
  if (store.homeShowComponent == 'articleInfoSide') {
    sideShowComponent.value = HomeSide;
    store.homeShowComponent = 'home'
  } else if (store.homeShowComponent == 'home') {
    sideShowComponent.value = ArticleInfoSide;
    store.homeShowComponent = 'articleInfoSide'
  }

  next()
})
</script>

<style scoped>
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

h1 {
  font-size: 3.5em;
}

h2 {
  font-size: 2.5em;
  text-shadow: rgba(0, 0, 0, 0.5) 0rem 0.2rem 0.3rem;
}

h3 {
  font-size: 2.2em;
  font-weight: bold;
  text-shadow: rgba(0, 0, 0, 0.5) 0rem 0.2rem 0.3rem;
  letter-spacing: 0.125rem;
}

.left {
  display: flex;
  flex-direction: column;
  align-items: center;
  width: 30%;
  height: 100vh;
  text-align: center;
  position: sticky;
  top: 0;
  margin: 9.6px 0;
}

.inner {
  padding: 0 25px;
  display: flex;
  justify-content: space-between;
}

.inner>* {
  background-color: #fff;
}

.top {
  margin-top: 18px;

  span {
    margin-right: 20px;
  }
}

.date {
  text-shadow: rgba(0, 0, 0, .5) 0rem .2rem .3rem;
}

.right {
  width: 67%;
  box-shadow: 0 .5rem .75rem .0625rem rgb(235, 235, 235);
  border-radius: .3125rem;
  margin: 9.6px 0;
  padding: 16px 0 32px;
}
</style>
