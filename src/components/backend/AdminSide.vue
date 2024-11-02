<template>
  <div class="container">
    <h1>DH-Blog</h1>
    <el-menu router :default-active="$route.path">
      <el-menu-item index="/admin/dashboard">
        <el-icon :size="iconSize">
          <Odometer />
        </el-icon>
        仪表盘
      </el-menu-item>
      <el-menu-item index="/admin/publish">
        <el-icon :size="iconSize">
          <Edit />
        </el-icon>
        博客发布
      </el-menu-item>
      <el-menu-item index="/admin/manager">
        <el-icon :size="iconSize">
          <EditPen />
        </el-icon>
        博客管理
      </el-menu-item>
      <el-menu-item index="/admin/system">
        <el-icon :size="iconSize">
          <Setting />
        </el-icon>
        系统配置
      </el-menu-item>
      <el-menu-item index="/admin/system">
        <el-icon :size="iconSize">
          <User />
        </el-icon>
        评论管理
      </el-menu-item>
    </el-menu>
  </div>
</template>
<style scoped lang="less">
h1 {
  font-weight: bold;
  font-size: 30px;
  font-style: italic;
  line-height: 80px;
  color: #3F8CFF;
  text-shadow: 1px 1px 2px rgba(0, 0, 0, 0.3);
  background-color: #FFF;
  text-align: center;
}

.container {
  height: 100%;
  display: flex;
  flex-direction: column;
  background-color: #fafbfc;
}

:deep(.el-menu) {
  padding: 0 10px;
  background-color: #fff;
  flex: 1;
  border: none;
}

.el-icon {
  margin-right: 15px;
}

.el-menu-item {
  --el-menu-hover-bg-color: rgb(245, 245, 245);
  border-radius: 10px;
  margin-bottom: 10px;
}

.is-active {
  --el-menu-active-color: #3F8CFF;
  background-color: var(--el-menu-bg-color);
  box-shadow: 0 0 27px #0000000f;
}
</style>
<script setup>
import { ref } from 'vue';
const iconSize = ref(20);
</script>