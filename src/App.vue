<template>
  <div class="container">
    <Header ref="headerElement"></Header>
    <Banner ref="bannerElement">
      <template v-if="router.path === '/'">
        <h1>我的个人纪录</h1>
        <h2>DH-BLOG</h2>
      </template>

      <template v-else>
        <h3>{{ "文章标题3" }}</h3>
        <div class="top">
          <span class="date">发表于 2023/7/21 09:15:47</span>
          <span class="num-word">本文字数 6 字</span>
          <span class="time-consum">阅读时长 1 分钟</span>
        </div>
      </template>
    </Banner>

    <div class="inner">
      <div class="left" ref="leftElement">
        <Transition mode="out-in">
          <component :is="sideShowComponent"></component>
        </Transition>
      </div>
      <div class="right"> 
        <Transition mode="out-in">
          <keep-alive>
            <router-view></router-view>
          </keep-alive>
        </Transition>
      </div>
    </div>
    <Footer></Footer>
  </div>
</template>

<script setup lang="ts">
import Header from './components/Header.vue';
import Banner from './components/Banner.vue';
import HomeSide from './components/Side/HomeSide.vue';
import ArticleInfoSide from './components/Side/ArticleInfoSide.vue';
import Footer from './components/Footer.vue';

import { onMounted, ref, watch } from 'vue'
import { useRoute } from 'vue-router'

const router = useRoute()
const sideShowComponent = ref(HomeSide)

// 监听路由
onMounted(() => {
  watch(() => router.path, (newVal, _) => {
    if (newVal === '/') {
      sideShowComponent.value = HomeSide
    } else {
      sideShowComponent.value = ArticleInfoSide
    }
  })
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

.left {
  display: flex;
  flex-direction: column;
  align-items: center;
  width: 30%;
  height: 100vh;
  text-align: center;
  position: sticky;
  top: 0;
  margin: .6rem 0;
}

.inner {
  padding: 0 1.5625rem;
  display: flex;
  justify-content: space-between;
}

.inner>* {
  background-color: #fff;
}

.top {
  margin-top: 1.125rem;

  span {
    margin-right: 1.25rem;
  }
}

.right {
  width: 67%;
  box-shadow: 0 8px 12px 1px rgb(235, 235, 235);
  border-radius: 5px;
  padding: 2.5rem 0;
}
</style>
