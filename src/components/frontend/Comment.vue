<template>
  <div class="box">
    <div style="width: 92%; margin: 0 auto">
      <Publish />
      <View />
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

<script>
import View from "@/components/frontend/Comment/View.vue";
import Publish from "@/components/frontend/Comment/Publish.vue";
</script>
