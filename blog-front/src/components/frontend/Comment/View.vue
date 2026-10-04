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
// 文章接口慢于下面的 1.5s 时会拉到上一篇文章的评论
const route = useRoute()
const commentList = ref([]);
const length = ref(0);

const isLoading = ref(true)

onMounted(() => {
  setTimeout(async () => {
    await changComment()
    isLoading.value = false
  }, 1500)
})

const changComment = async () => {
  const data = await getCommentList(Number(route.params.id));
  commentList.value = data.list
  length.value = data.total
}
</script>
