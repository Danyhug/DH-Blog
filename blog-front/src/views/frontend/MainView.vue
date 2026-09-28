<template>
  <div>
    <!-- 这里是文章列表 -->
    <div class="p-[18px]">
      <ArticleBox v-loading="show" v-for="item in store.articleList" :article="item" :key="item.id" class="mb-[46px]"></ArticleBox>
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
import { onMounted, ref } from 'vue'
import { useUserStore } from '@/store';

const store = useUserStore()
const show = ref(true)

const getPageList = async (animate = false) => {
  const res = await getArticleList(store.page)
  store.articleList = res.list; // 简化数组替换
  store.page.total = res.total;

  if (animate) {
    setTimeout(() => {
      const bannerHeight = document.querySelector('#banner')?.scrollHeight || 0
      scrollTo(0, bannerHeight)
      show.value = false
    }, 320)
  } else {
    show.value = false
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
