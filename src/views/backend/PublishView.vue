<template>
  <el-divider content-position="center">
    <p class="tip">文章标题</p>
  </el-divider>
  <el-input placeholder="输入文章标题" clearable class="title-input" v-model="article.title"></el-input>

  <el-divider content-position="center">
    <p class="tip">文章内容</p>
  </el-divider>
  <MdEditor v-model="article.content" previewTheme="github" />

  <el-divider content-position="center">
    <p class="tip">附加信息</p>
  </el-divider>

  <!-- 分类： -->
  <el-radio-group>
    <el-radio value="1" size="large" border>原神</el-radio>
    <el-radio value="2" size="large" border>崩坏三</el-radio>
  </el-radio-group>

  <!-- 标签： -->
  <el-checkbox-group size="large">
    <el-checkbox-button v-for="(city, index) in 5" :key="index" :value="city">
      {{ "游戏" }}
    </el-checkbox-button>
  </el-checkbox-group>

  <el-button type="primary" round>发布</el-button>
  <el-button type="danger" round>清空</el-button>
</template>

<script setup lang="ts">
import { reactive } from 'vue';
const article = reactive({
  title: '',
  content: '',
  created: 0,
  update: 0,
  viewnum: 0,
})
</script>

<style scoped>
.el-divider {
  background-color: #1E9FFF;
}

.tip {
  font-size: 18px;
  margin: 10px 0;
}

.title-input {
  font-size: 16px;
}
</style>