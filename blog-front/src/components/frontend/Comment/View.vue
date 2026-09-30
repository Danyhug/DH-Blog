<template>
  <div class="relative w-full my-[30px] mx-auto text-[rgb(49,49,49)] [.article-night_&]:text-[#ccc]">
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
import { useUserStore } from '@/store/index';
import Loading from '@/components/frontend/Loading.vue'
const store = useUserStore()
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
  const data = await getCommentList(store.homeHeaderInfo.id);
  commentList.value = data.list
  length.value = data.total
}
</script>
