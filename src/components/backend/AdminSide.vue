<template>
  <div class="container">
    <h1>DH-Blog</h1>
    <el-menu router :default-active="$route.path">
      <el-menu-item index="/admin/dashboard">
        仪表盘
      </el-menu-item>
      <el-menu-item index="/admin/publish">
        博客发布
      </el-menu-item>
      <el-menu-item index="/admin/manager">
        博客管理
      </el-menu-item>
      <el-menu-item index="/admin/system">
        系统配置
      </el-menu-item>
    </el-menu>
  </div>
</template>
<style scoped>
h1 {
  font-family: "Varela Round";
  font-weight: bold;
  font-size: 28px;
  background-image: linear-gradient(120deg, rgb(119, 183, 247), rgb(98, 224, 161));
  line-height: 80px;
  color: #FFF;
  text-align: center;
}

.container {
  height: 100vh;
  display: flex;
  flex-direction: column;
}
</style>
<style>
.el-divider__text {
  background-color: rgb(253, 253, 253);
}

.el-menu {
  flex: 1;
}
</style>