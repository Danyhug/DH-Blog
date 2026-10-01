<template>
  <div>
    <!-- 这里是文章列表 -->
    <div ref="listElement" class="p-[18px]">
      <ArticleBox v-reveal v-loading="show" v-for="item in store.articleList" :article="item" :key="item.id" class="mb-[46px]"></ArticleBox>
    </div>
    <div class="flex justify-center items-center px-[18px] pt-0 pb-8">
      <Pagination :pageSize="store.page.pageSize" :currentPage="store.page.pageNum" :total="store.page.total"
        @update:currentPage="changePage"></Pagination>
    </div>
  </div>
</template>
<script lang="ts" setup>
import { getArticleList } from '@/api/user';
import ArticleBox from '@/components/frontend/ArticleBox.vue'
import Pagination from '@/components/frontend/Pagination.vue';
import { vReveal } from '@/directives/reveal';
import { nextTick, onMounted, ref } from 'vue'
import { useUserStore } from '@/store';

const store = useUserStore()
const show = ref(true)
const listElement = ref<HTMLElement>()

const getPageList = async (animate = false) => {
  const res = await getArticleList(store.page)
  store.articleList = res.list; // 简化数组替换
  store.page.total = res.total;
  // 与换数据同一帧关掉 loading：新卡片挂载时不带遮罩，滚动途中的淡入才看得见
  show.value = false

  if (animate) {
    // 翻页：从页码处滚回本页第一篇。不传 behavior，跟随 html 的 scroll-behavior——
    // 平时是 smooth，系统开启「减少动态效果」时被 tailwind.css 的全局规则改成 auto，直接跳到位。
    // 目标取列表容器而不是首张卡片：卡片淡入前带着 translate-y，按它定位会偏。
    await nextTick()
    listElement.value?.scrollIntoView({ block: 'start' })
  }
}

onMounted(() => {
  // 每次进入首页都按当前登录状态刷新，避免复用访客或管理员的旧列表。
  getPageList()
})

// 更新页面
const changePage = (curr: number) => {
  show.value = true;

  store.page.pageNum = curr;
  getPageList(true)
}

</script>
