<template>
  <div>
    <div class="btn-group">
      <div class="left">
        <el-button round type="primary" plain :icon="DArrowRight" class="downTree" />
        <el-button circle type="primary" plain :icon="Refresh" :loading="isLoading" @click="isLoading = true" />
      </div>

      <el-button-group class="right">
        <el-button type="primary" :icon="Edit" @click="edit">编辑</el-button>
        <el-button type="success" :icon="ChatDotRound" @click="reply">回复</el-button>
        <el-button type="danger" :icon="Delete" />
      </el-button-group>
    </div>


    <el-table :data="commentList" stripe height="80vh" style="width: 100%" row-key="id" highlight-current-row
      @current-change="handleCurrentChange" border indent="8">
      <el-table-column label="ID" prop="id" width="100"></el-table-column>
      <el-table-column label="文章标题" width="85">
        <template #default="scope">
          <el-popover placement="top-start" trigger="click" @show="getTitle(scope.row.articleId)">
            <template #reference>
              <el-button size="small">SHOW</el-button>
            </template>
            <div style="font-size: 1.2em;">
              {{ articleTitleCache[scope.row.articleId] }}&nbsp;
              <el-button type="warning" size="small" @click="openArticle(scope.row.articleId)">去看看</el-button>
            </div>
          </el-popover>
        </template>
      </el-table-column>
      <el-table-column label="作者" prop="author" width="120"></el-table-column>
      <el-table-column label="邮箱" prop="email"></el-table-column>
      <el-table-column label="内容" prop="content" min-width="150"></el-table-column>
      <el-table-column label="环境" width="120">
        <template #default="scope">
          <el-tag type="success" style="margin-bottom: 6px;">{{ scope.row.ua.split(';')[0] }}</el-tag>
          <el-tag type="primary" round>{{ scope.row.ua.split(';')[1].split(' ')[1] }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column label="时间" width="110">
        <template #default="scope">
          <span v-html="scope.row.createTime.replace('T', '<br />')" style="text-align: center;"></span>
        </template>
      </el-table-column>
    </el-table>
  </div>
</template>

<script setup lang="ts">
import { ref } from "vue";
import { getAllComment } from "@/api/admin";
import { Comment } from "@/types/Comment";
import { Delete, Edit, ChatDotRound, Refresh, DArrowRight } from "@element-plus/icons-vue";
import { getArticleTitleById } from "@/api/user";
import { useRouter } from "vue-router";
const router = useRouter();
const isLoading = ref(false)
const openArticle = (articleId: number) => {
  // 使用 window.open 方法打开文章页
  window.open(router.resolve({ name: 'ArticleInfo', params: { id: articleId } }).href);
};// 定义评论列表数据
let commentList = ref<Comment[]>([
  {
    id: 1,
    articleId: 23,
    author: "Faker",
    email: "faker@qq.com",
    content: "巅峰的你就连我也得避其锋芒🤡",
    isPublic: true,
    createTime: "2024-11-18T20:18:05",
    parentId: null,
    ua: "Windows 10; Edge 13.0.0.0",
    isAdmin: false,
    children: [
      {
        id: 16,
        articleId: 3,
        author: "Lovevivi",
        email: "love@qq.com",
        content: "菜就多练😅",
        isPublic: true,
        createTime: "2024-11-19T14:13:10",
        parentId: 1,
        ua: "Windows 10; Chrome 130.0.0.0",
        isAdmin: false,
        children: [],
      },
    ],
  },
  {
    id: 2,
    articleId: 3,
    author: "Bang",
    email: "bang@qq.com",
    content: "如果S7你来打ADC，或许SKT能拿下三星👍",
    isPublic: true,
    createTime: "2024-11-18T20:21:48",
    parentId: null,
    ua: "Windows 10; Edge 13.0.0.0",
    isAdmin: false,
    children: [],
  },
  {
    id: 3,
    articleId: 3,
    author: "RYL.White",
    email: "white@qq.com",
    content: "如果S3让你和飞科对线，或许英雄联盟的历史就会改写了💪",
    isPublic: true,
    createTime: "2024-11-18T20:23:11",
    parentId: null,
    ua: "Windows 10; Chrome 130.0.0.0",
    isAdmin: false,
    children: [],
  },
  {
    id: 4,
    articleId: 3,
    author: "RNG.UZI",
    email: "uzi@qq.com",
    content: "我愿意称你为世界第一VN🤗",
    isPublic: true,
    createTime: "2024-11-18T20:23:45",
    parentId: null,
    ua: "Windows 10; Chrome 130.0.0.0",
    isAdmin: false,
    children: [],
  },
  {
    id: 5,
    articleId: 3,
    author: "IG.Theshy",
    email: "1457191996@qq.com",
    content: "如同天上降魔主，真是人间太岁神🤡",
    isPublic: true,
    createTime: "2024-11-19T09:06:40",
    parentId: null,
    ua: "Windows 10; Chrome 109.0.0.0",
    isAdmin: false,
    children: [],
  },
  {
    id: 6,
    articleId: 3,
    author: "王多多",
    email: "1234567890@qq.com",
    content: "把头埋低，这是就是Thyshy",
    isPublic: true,
    createTime: "2024-11-19T09:07:28",
    parentId: null,
    ua: "Windows 10; Chrome 109.0.0.0",
    isAdmin: false,
    children: [],
  },
  {
    id: 7,
    articleId: 3,
    author: "刘备",
    email: "1111112334@qq.com",
    content: "穿上草鞋，飞一般的感觉🤔",
    isPublic: true,
    createTime: "2024-11-19T09:08:20",
    parentId: null,
    ua: "Windows 10; Chrome 109.0.0.0",
    isAdmin: false,
    children: [],
  },
  {
    id: 8,
    articleId: 3,
    author: "Bin",
    email: "1233332111@qq.com",
    content: "我会把你打回原型😘",
    isPublic: true,
    createTime: "2024-11-19T09:20:04",
    parentId: null,
    ua: "Windows 10; Chrome 109.0.0.0",
    isAdmin: false,
    children: [],
  },
  {
    id: 9,
    articleId: 3,
    author: "Doinb",
    email: "1232323@qq.com",
    content: "洲际赛，将韩国国籍打没的人😉",
    isPublic: true,
    createTime: "2024-11-19T09:21:37",
    parentId: null,
    ua: "Windows 10; Chrome 109.0.0.0",
    isAdmin: false,
    children: [
      {
        id: 15,
        articleId: 3,
        author: "zhulin",
        email: "11111111111@qq.com",
        content: "虚空的神-s1-s14冠军掠夺者------Uzi！",
        isPublic: true,
        createTime: "2024-11-19T14:12:38",
        parentId: 9,
        ua: "Windows 10; Chrome 130.0.0.0",
        isAdmin: false,
        children: [],
      },
    ],
  },
]);

let page = {
  pageSize: 10,
  pageNum: 1,
};
const currentRow = ref()

// 文章标题缓存
const articleTitleCache = reactive<{ [key: number]: string }>({
  1: "S7和S3谁更厉害",
})

const getTitle = (articleId: number) => {
  if (articleId in articleTitleCache) return

  getArticleTitleById(articleId).then(res => {
    articleTitleCache[articleId] = res
  })
}

const handleCurrentChange = (val: Comment | undefined) => {
  currentRow.value = val
}

const edit = () => { }

const reply = () => { }

onMounted(() => {
  getAllComment(page.pageSize, page.pageNum).then((res) => {
    // commentList.value.push(...res.list);
  });
});
</script>

<style scoped lang="less">
.btn-group {
  margin-bottom: 12px;
  display: flex;
  justify-content: space-between;
  padding: 0 16px;

  .left {
    .downTree {
      :deep(.el-icon) {
        transform: rotate(90deg) !important;
      }
    }
  }

  .right {}
}
</style>
