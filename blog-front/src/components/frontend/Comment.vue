<template>
  <div class="mt-[60px] bg-[rgb(250,250,250)] dark:bg-[#141414] px-0 pt-[30px] pb-9">
    <div class="w-[92%] mx-auto">
      <Publish v-if="site.open_comment" @comment-submitted="send" />
      <p v-else class="mt-0 mb-6 py-[18px] text-center text-[#999] text-[14px]">评论功能已关闭</p>
      <View :key="store.commentKey" />
    </div>
  </div>
</template>

<script setup>
import View from "@/components/frontend/Comment/View.vue";
import Publish from "@/components/frontend/Comment/Publish.vue";
import { useUserStore, useSiteStore } from "@/store";
import { storeToRefs } from "pinia";
const store = useUserStore()
const siteStore = useSiteStore()
const { site } = storeToRefs(siteStore)

onMounted(() => siteStore.loadSite())

const send = () => {
  store.commentKey = !store.commentKey
}
</script>
