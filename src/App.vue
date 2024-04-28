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
        <component :is="sideShowComponent"></component>
      </div>
      <div class="right">
        <router-view></router-view>
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

import { onMounted, ref } from 'vue'
import { useRoute } from 'vue-router'

const router = useRoute()

// const sideShowComponent = ref(HomeSide)
const sideShowComponent = ref(ArticleInfoSide)

onMounted(() => {

})
</script>

<style scoped>
.left {
  display: flex;
  flex-direction: column;
  align-items: center;
  width: 30%;
  height: 100vh;
  text-align: center;
  position: sticky;
  top: 0;
  left: 0;
}

.inner {
  padding: 2.5rem 1.5625rem;
  display: flex;
  justify-content: space-between;;
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
  box-shadow: 0 0 12px 1px rgb(235, 235, 235);
}
</style>
