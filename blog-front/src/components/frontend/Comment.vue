<template>
  <div class="box">
    <div style="width: 92%; margin: 0 auto">
      <Publish @comment-submitted="send" />
      <View :key="store.commentKey" />
    </div>
  </div>
</template>

<style lang="less" scoped>
.box {
  margin-top: 60px;
  background-color: rgb(250, 250, 250);
  padding: 30px 0 36px 0;
}
</style>

<script setup>
import View from "@/components/frontend/Comment/View.vue";
import Publish from "@/components/frontend/Comment/Publish.vue";
import { addComment } from '@/api/user.ts'
import { useUserStore } from "@/store";
const store = useUserStore()

const key = ref(true)
const send = (comment) => {
  addComment(comment)
  store.commentKey = !store.commentKey
}
</script>
