<template>
  <div class="box">
    <div style="width: 92%; margin: 0 auto">
      <Publish v-if="site.open_comment" @comment-submitted="send" />
      <p v-else class="closed-tip">评论功能已关闭</p>
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

.closed-tip {
  margin: 0 0 24px 0;
  padding: 18px 0;
  text-align: center;
  color: #999;
  font-size: 14px;
}
</style>

<script setup>
import View from "@/components/frontend/Comment/View.vue";
import Publish from "@/components/frontend/Comment/Publish.vue";
import { addComment } from '@/api/user.ts'
import { useUserStore, useSiteStore } from "@/store";
import { storeToRefs } from "pinia";
const store = useUserStore()
const siteStore = useSiteStore()
const { site } = storeToRefs(siteStore)

onMounted(() => siteStore.loadSite())

const send = async (comment) => {
  await addComment(comment)
  store.commentKey = !store.commentKey
}
</script>
