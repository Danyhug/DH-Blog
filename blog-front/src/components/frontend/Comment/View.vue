<template>
  <div class="relative w-full my-[30px] mx-auto text-[rgb(49,49,49)] dark:text-[#ccc]">
    <Loading v-if="isLoading" />

    <div>
      <div class="count">{{ length }} 条评论</div>
      <ul class="my-6 w-full">
        <CommentItem :commentList="commentList" />
      </ul>
    </div>
  </div>
</template>

<script setup>
import { reactive } from 'vue';
import CommentItem from '@/components/frontend/Comment/CommentItem.vue';
import { formatDate } from '@/utils/tool'
import { getCommentList } from '@/api/user'
import { useRoute } from 'vue-router';
import Loading from '@/components/frontend/Loading.vue'
// 用路由上的文章 id 而不是 store.homeHeaderInfo.id：后者要等文章接口返回才更新，
// 文章接口慢于下面的 1.5s 时会拉到上一篇文章的评论。
// 挂载时就取下来：等到定时器触发时可能已经离开文章页（回首页、私密文章被转去 /lock），
// 那时 route.params.id 是 undefined，会请求 /comment/NaN 得到 400
const route = useRoute()
const articleId = Number(route.params.id)
const commentList = ref([]);
const length = ref(0);

const isLoading = ref(true)

let timer
onMounted(() => {
  timer = setTimeout(async () => {
    await changComment()
    isLoading.value = false
  }, 1500)
})
onBeforeUnmount(() => clearTimeout(timer))

const changComment = async () => {
  if (!Number.isInteger(articleId)) return
  const data = await getCommentList(articleId);
  // 没有评论时后端的 list 是 null（Go 的 nil 切片），CommentItem 的 prop 要求数组
  commentList.value = data.list ?? []
  length.value = data.total
}
</script>
