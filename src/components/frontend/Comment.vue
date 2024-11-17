<template>
  <div class="box">
    <Publish />
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
import View from '@/components/frontend/comment/View.vue'
import Publish from '@/components/frontend/Comment/Publish.vue'
</script>