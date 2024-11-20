<template>
  <div>
    <div class="btn-group">
      <div class="left">
        <el-button round :icon="DArrowRight" type="primary" class="downTree" @click="expandAllRows" />
        <el-button circle plain :icon="Refresh" :loading="isLoading" @click="getData" />
      </div>

      <el-button-group class="right">
        <el-button :icon="Edit" @click="currentRow && (editVisible = true)" />
        <el-button :icon="ChatDotRound" @click="currentRow && (replyDialogVisible = true)">回复</el-button>
        <el-button :icon="Delete" type="danger" @click="currentRow && deleteCom(currentRow.id)" />
      </el-button-group>
    </div>

    <el-table ref="myTable" :data="commentList" stripe height="79vh" class="table" row-key="id" highlight-current-row
      @current-change="handleCurrentChange" indent="'8'">
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

    <!-- 编辑评论 -->
    <el-drawer title="评论详情" v-model="editVisible" width="500">
      <el-form :model="currentRow" label-width="80px">
        <el-form-item label="作者">
          <el-input v-model="currentRow.author"></el-input>
        </el-form-item>
        <el-form-item label="邮箱">
          <el-input v-model="currentRow.email"></el-input>
        </el-form-item>
        <el-form-item label="内容">
          <el-input type="textarea" v-model="currentRow.content"></el-input>
        </el-form-item>
        <el-form-item label="是否公开">
          <el-switch v-model="currentRow.isPublic"></el-switch>
        </el-form-item>
        <el-form-item label="父评论ID">
          <el-input v-model="currentRow.parentId"></el-input>
        </el-form-item>
      </el-form>
      <template #footer>
        <span class="dialog-footer">
          <el-button @click="editVisible = false">关闭</el-button>
          <el-button type="primary" @click="edit">修改</el-button>
        </span>
      </template>
    </el-drawer>

    <!-- 回复评论 -->
    <el-dialog title="回复评论" v-model="replyDialogVisible">
      <el-form :model="replyForm" label-width="120px">
        <el-form-item label="回复内容" prop="content">
          <el-input type="textarea" v-model="replyForm.content" :rows="4" placeholder="请输入回复内容"></el-input>
        </el-form-item>
        <el-form-item label="是否公开" prop="isPublic">
          <el-switch v-model="replyForm.isPublic"></el-switch>
        </el-form-item>

        <div class="emoji">
          <ul>
            <li v-for="ji in emojis" :key="ji" @click="addEmj(ji)">{{ ji }}</li>
          </ul>
        </div>
      </el-form>
      <template #footer>
        <span class="dialog-footer">
          <el-button @click="replyDialogVisible = false">取消</el-button>
          <el-button type="primary" @click="reply">提交</el-button>
        </span>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref } from "vue";
import { deleteComment, getAllComment } from "@/api/admin";
import { Comment } from "@/types/Comment";
import { Edit, ChatDotRound, Refresh, DArrowRight, Delete } from "@element-plus/icons-vue";
import { getArticleTitleById } from "@/api/user";
import { useRouter } from "vue-router";
import { emojis } from '@/types/Constant';
import { editComment, replyComment } from '@/api/admin'

const router = useRouter();
const isLoading = ref(false)
const myTable = ref<any>()
// 定义评论列表数据
const commentList = ref<Comment[]>([]);
const editVisible = ref(false);
const replyDialogVisible = ref(false)
const replyForm = reactive({
  content: "",
  isPublic: true,
  parentId: null,
  articleId: null,
})
let page = reactive({
  pageSize: 10,
  pageNum: 1,
});
const currentRow = ref()

const openArticle = (articleId: number) => {
  window.open(router.resolve({ name: 'ArticleInfo', params: { id: articleId } }).href);
};

// 文章标题缓存
const articleTitleCache = reactive<{ [key: number]: string }>({})

const getTitle = (articleId: number) => {
  if (articleId in articleTitleCache) return

  getArticleTitleById(articleId).then(res => {
    articleTitleCache[articleId] = res
  })
}

const handleCurrentChange = (val: Comment | undefined) => {
  currentRow.value = val
}

const addEmj = (val: string) => {
  replyForm.content += val
}

// 编辑评论
const edit = () => {
  editComment(currentRow.value).then(res => {
    editVisible.value = false
    ElMessage.success(res)
    getData()
  })
}

// 回复评论
const reply = () => {
  replyComment(replyForm.content, replyForm.isPublic, currentRow.value.id, currentRow.value.articleId).then(res => {
    replyDialogVisible.value = false
    ElMessage.success(res)
    getData()
  })
}

const getData = () => {
  isLoading.value = true
  getAllComment(page.pageSize, page.pageNum).then((res) => {
    commentList.value = [...res.list];
    ElNotification.success({
      title: '提示信息',
      message: "已刷新评论信息",
      position: 'bottom-right',
    })

    isLoading.value = false
  });
}

// 展开/收起评论树
const expand = ref(false)
const expandAllRows = () => {
  expand.value = !expand.value
  console.log(myTable.value.store.states.data._rawValue)
  myTable.value.store.states.data._rawValue.forEach((row: Comment) => {
    myTable.value.toggleRowExpansion(row, expand);
    row.children?.forEach((child: Comment) => {
      myTable.value.toggleRowExpansion(child, expand);
    })
  });
}

const deleteCom = (id: number) => {
  // 确定删除吗
  ElMessageBox.confirm('确定删除吗？', '提示', {
    confirmButtonText: '确定',
    cancelButtonText: '取消',
    type: 'warning',
  }).then(() => {
    // 删除评论
    deleteComment(id).then(res => {
      ElMessage.success(res)
      getData()
    })
  })
}

onMounted(() => {
  getData()
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

.table {
  width: 100%;
  border-radius: 6px;
  background: #fff;
  box-shadow: 0 4px 20px #00000008;
}

.emoji {
  ul {
    list-style: none;
    font-size: 18px;
    display: grid;
    grid-template-columns: repeat(10, 1fr);
    grid-gap: 6px;

    li {
      margin-right: 10px;
      cursor: pointer;
    }
  }
}
</style>
