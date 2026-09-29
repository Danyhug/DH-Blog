<template>
  <div class="common-layout">
    <el-container class="relative">
      <el-aside class="sticky top-0 h-screen bg-[#fbfbfb] shadow-[4px_0_10px_rgb(248,248,248)] transition-[width] duration-[600ms] [--el-aside-width:234px]">
        <AdminSide></AdminSide>
      </el-aside>
      <el-container>
        <el-main class="bg-[var(--dh-admin-bg)]">
          <router-view></router-view>
        </el-main>
        <el-footer class="flex items-center justify-center">
          <AdminFooter></AdminFooter>
        </el-footer>
      </el-container>
    </el-container>
  </div>
</template>

<script setup lang="ts">
import AdminSide from '@/components/backend/AdminSide.vue';
import AdminFooter from '@/components/backend/AdminFooter.vue';
</script>

<style scoped>
/* .el-aside 自带 overflow:auto（无层级），压过工具类 */
.el-aside {
  overflow: hidden;
}
</style>
