<template>
  <div class="common-layout">
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
  --el-aside-width: 234px;
  height: 100vh;
  background-color: #fbfbfb;
  box-shadow: 4px 0 10px rgb(248, 248, 248);
  transition: width 0.6s;
  overflow: hidden;
}

.el-main {
  background-color: var(--dh-admin-bg);
}
</style>
