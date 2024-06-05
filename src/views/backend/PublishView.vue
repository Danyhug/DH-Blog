<template>
  <div class="top-container">
    <div class="left">
      <el-tabs class="demo-tabs" tabPosition="left">
        <el-tab-pane label="分类选择" name="first">
          <el-radio-group>
            <el-radio value="1" size="large" border>原神</el-radio>
            <el-radio value="2" size="large" border>崩坏三</el-radio>
          </el-radio-group>
        </el-tab-pane>
        <el-tab-pane label="标签选择" name="second">
          <!-- 标签： -->
          <el-checkbox-group size="large">
            <el-checkbox-button v-for="(city, index) in 5" :key="index" :value="city">
              {{ "游戏" }}
            </el-checkbox-button>
          </el-checkbox-group>
        </el-tab-pane>
        <el-tab-pane label="附加信息" name="third">
          我是个信息
        </el-tab-pane>
      </el-tabs>
    </div>

    <div class="right">
      <el-divider content-position="center">
        <p class="tip">文章标题</p>
      </el-divider>
      <el-input placeholder="输入文章标题" clearable class="title-input" v-model="article.title"></el-input>
      <div class="btns">
        <el-button type="primary">发布</el-button>
        <el-button type="danger">清空</el-button>
      </div>
    </div>
  </div>


  <el-divider content-position="center">
    <p class="tip">文章内容</p>
  </el-divider>
  <MdEditor v-model="article.content" previewTheme="github" />
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
.top-container {
  display: flex;
  justify-content: space-between;

  &>div {
    height: 126px;
    width: 46%;
  }

  .right {
    .btns {
      margin: 20px 0;
      text-align: right;
    }
  }
}

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