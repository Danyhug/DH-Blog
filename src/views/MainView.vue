<template>
  <div>
    <!-- 这里是文章列表 -->
    <div class="tip">
      <p>文章列表</p>
    </div>
    <div class="posts">
      <Article v-for="item in articleList" :id="item.ID" :title="item.Title" :content="item.Content"
        :created="item.Created"></Article>
    </div>
  </div>
</template>
<script>
export default {
  name: 'MainView',
  data() {
    return {
      articleList: [
        {
          ID: 1,
          Title: '文章标题1',
          Content: '文章内容1',
          Created: '2022-01-01'
        },
        {
          ID: 1,
          Title: '文章标题1',
          Content: '文章内容1',
          Created: '2022-01-01'
        }
        ,
        {
          ID: 1,
          Title: '文章标题1',
          Content: '文章内容1',
          Created: '2022-01-01'
        },
        {
          ID: 1,
          Title: '文章标题1',
          Content: '文章内容1',
          Created: '2022-01-01'
        },
        {
          ID: 1,
          Title: '文章标题1',
          Content: '文章内容1',
          Created: '2022-01-01'
        },
        {
          ID: 1,
          Title: '文章标题1',
          Content: '文章内容1',
          Created: '2022-01-01'
        }
      ]
    }
  },
}
</script>
<style lang="less" scoped>
.tip {
  display: block;
  width: 100%;
  text-align: center;
  font-weight: 700;
  border-top: 1px solid var(--grey-5);

  p {
    display: inline-block;
    transform: translateY(-50%);
    background-color: #fff;
    padding: 0 16px;
    font-weight: 400;
    color: #606266;
    font-size: 24px;
  }
}

/* 文章父元素 */
.posts {
  padding: 16px 10px;
}

Article {
  margin-bottom: 40px;
}
</style>