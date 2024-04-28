<script setup lang="ts">
import Header from './components/Header.vue';
import Banner from './components/Banner.vue';
import Side from './components/Side.vue';
import Footer from './components/Footer.vue';

import { onMounted } from 'vue'
import { useRoute } from 'vue-router'

const router = useRoute()

onMounted(() => {
})

</script>

<template>
  <div class="container">
    <Header></Header>
    <Banner>

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
      <Side></Side>
      <div class="right">
        <router-view></router-view>
      </div>
    </div>
    <Footer></Footer>
  </div>
</template>

<style scoped>
.inner>* {
  padding: 2.5rem 1.5625rem;
}

.top {
  margin-top: 1.125rem;

  span {
    margin-right: 1.25rem;
  }
}

.right {
  width: 70%;
  box-shadow: 0 3px .625rem .125rem var(--grey-2);
}
</style>
