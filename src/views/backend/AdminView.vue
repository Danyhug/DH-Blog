<template>
  <div class="common-layout">
    <el-container>
      <el-container style="position: relative;">
        <el-aside>
          <AdminSide></AdminSide>
        </el-aside>
        <el-container>
          <el-main>
            <router-view></router-view>
          </el-main>
          <el-footer style="display: flex; align-items: center; justify-content: center;">
            <AdminFooter></AdminFooter>
          </el-footer>
        </el-container>
      </el-container>
    </el-container>
  </div>
</template>

<script setup lang="ts">
import AdminSide from '@/components/backend/AdminSide.vue';
import AdminFooter from '@/components/backend/AdminFooter.vue';
</script>

<style scoped>
.el-aside {
  position: sticky;
  top: 0;
  height: 100vh;
  width: 200px;
}

/* 手机适配 */
@media (max-width: 768px) {
  .el-aside {
    display: none;
  }
}
</style>
