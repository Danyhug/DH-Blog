<template>
  <div ref="rootElement" class="relative w-full my-[30px] mx-auto text-[rgb(49,49,49)] dark:text-[#ccc]">
    <Loading v-if="isLoading" />

    <div>
      <div class="count">{{ total }} 条评论</div>
      <ul class="my-6 w-full">
        <CommentItem :commentList="commentList" />
      </ul>
      <Pagination
        :current-page="page"
        :page-size="PAGE_SIZE"
        :total="rootTotal"
        label="评论分页"
        @update:current-page="changePage"
      />
    </div>
  </div>
</template>

<script setup>
import { ref } from 'vue';
import CommentItem from '@/components/frontend/Comment/CommentItem.vue';
import { formatDate } from '@/utils/tool'
import { getCommentList } from '@/api/user'
import { useRoute } from 'vue-router';
import Loading from '@/components/frontend/Loading.vue'
import Pagination from '@/components/frontend/Pagination.vue'

// 用路由上的文章 id 而不是 store.homeHeaderInfo.id：后者要等文章接口返回才更新，
// 文章接口慢于下面的 1.5s 时会拉到上一篇文章的评论。
// 挂载时就取下来：等到定时器触发时可能已经离开文章页（回首页、私密文章被转去 /lock），
// 那时 route.params.id 是 undefined，会请求 /comment/NaN 得到 400
const route = useRoute()
const articleId = Number(route.params.id)
const rootElement = ref(null)

// 分页切的是顶层评论：一条对话连同它的回复总在同一页，所以页数按 rootTotal 算
const PAGE_SIZE = 10
const page = ref(1)
const commentList = ref([]);
// total 是全部公开评论数（含回复），rootTotal 是顶层评论数
const total = ref(0);
const rootTotal = ref(0);

const isLoading = ref(true)

let timer
onMounted(() => {
  timer = setTimeout(async () => {
    await changComment()
    isLoading.value = false
  }, 1500)
})
onBeforeUnmount(() => clearTimeout(timer))

const changComment = async (animate = false) => {
  if (!Number.isInteger(articleId)) return
  const data = await getCommentList(articleId, page.value, PAGE_SIZE);
  // 没有评论时后端的 list 是 null（Go 的 nil 切片），CommentItem 的 prop 要求数组
  commentList.value = data.list ?? []
  total.value = data.total
  rootTotal.value = data.rootTotal

  if (animate) {
    // 翻页后回到评论区顶部，否则读者会停在上一页末尾的空白处
    // 不传 behavior，跟随 html 的 scroll-behavior（系统开了「减少动态效果」时会被改成 auto）
    rootElement.value?.scrollIntoView({ block: 'start' })
  }
}

const changePage = async (next) => {
  page.value = next
  await changComment(true)
}
</script>
