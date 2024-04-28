<template>
  <!-- 文章浏览页 -->
  <div>
    <router-link to="/">首页</router-link>
    <div class="blog-container">
      <v-md-preview :text="'# 测试文本'"></v-md-preview>
    </div>
    <div class="info">
      <span>
        更新于 {{ update }}
      </span>
      <span>
        阅读次数 {{ viewnum }} 次
      </span>
    </div>
  </div>
</template>

<script>
export default {
  name: 'HomeView',
  data() {
    return {
      // 文章信息
      id: 1,
      title: '',
      content: '测试文章信息',
      created: 0,
      update: 0,
      viewnum: 0
    }
  },
  data() {
    return {
      
    }
  },
}
</script>
<style lang="less" scoped>
.blog-container {
  padding: 1.25rem 1.5rem;
}

.info {
  padding: 10px 0;
  font-size: 12px;
  color: var(--gery-6);
  text-align: right;
  border-top: 1px solid var(--grey-4);

  span {
    margin: 0 10px;
  }
}
</style>
