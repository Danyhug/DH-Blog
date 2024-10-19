<template>
  <div>
    <!-- 这里是文章列表 -->
    <div class="tip">
      <p>文章列表</p>
    </div>
    <div class="posts">
      <ArticleBox v-loading="show" v-for="item in store.articleList" :article="item" :key="item.id"></ArticleBox>
    </div>
    <div class="page">
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

const getPageList = () => {
  getArticleList(store.page).then(res => {
    store.articleList = res.list; // 简化数组替换

    // 如果是首次加载数据，则设置数据总数
    if (store.page.total === 0) {
      store.page.total = res.total;
      show.value = false;
    } else {
      // 无论是否首次，都在一段时间后隐藏加载动画并滚动
      setTimeout(() => {
        const bannerHeight = document.querySelector('#banner')?.scrollHeight || 0;
        scrollTo(0, bannerHeight);
        show.value = false;
      }, 320);
    }
  });
}

onMounted(() => {
  // 无缓存
  if (store.articleList.length == 0) {
    // 第一次加载，从服务器获取数据
    getPageList()
  } else {
    show.value = false;
  }
})

// 更新页面
const changePage = (curr: number) => {
  show.value = true;

  store.page.pageNum = curr;
  getPageList()
}

</script>
<style lang="less" scoped>
.tip {
  display: block;
  width: 100%;
  text-align: center;
  font-weight: 700;

  p {
    display: inline-block;
    transform: translateY(-50%);
    background-color: #fff;
    padding: 0 16px;
    font-weight: 400;
    color: #606266;
    font-size: 24px;
  }
}

/* 文章父元素 */
.posts {
  padding: 18px;
}

Article {
  margin-bottom: 46px;
}

.page {
  display: flex;
  justify-content: center;
  align-items: center;
}
</style>