<template>
  <div class="comment-manager">
    <div class="page-toolbar">
      <div class="title-block">
        <h2>文章评论</h2>
        <p>按文章查看评论与回复，共 {{ total }} 篇文章有评论</p>
      </div>

      <div class="toolbar-actions">
        <el-button round type="primary" plain :icon="DArrowRight" @click="expandAllRows">
          {{ expanded ? '收起全部' : '展开全部' }}
        </el-button>
        <el-button circle plain :icon="Refresh" :loading="isLoading" aria-label="刷新评论" @click="refreshData" />

        <el-button-group class="comment-actions">
          <el-button :icon="Edit" :disabled="!selectedComment" @click="openEdit">编辑</el-button>
          <el-button :icon="ChatDotRound" :disabled="!selectedComment" @click="openReply">回复</el-button>
          <el-button :icon="Delete" type="danger" :disabled="!selectedComment" @click="deleteSelectedComment">
            删除
          </el-button>
        </el-button-group>
      </div>
    </div>

    <el-table
      ref="tableRef"
      v-loading="isLoading"
      :data="commentGroups"
      :row-key="getRowKey"
      :row-class-name="getRowClassName"
      :tree-props="{ children: 'children' }"
      stripe
      highlight-current-row
      class="comment-table"
      height="calc(100vh - 238px)"
      indent="22"
      empty-text="还没有收到评论"
      @current-change="handleCurrentChange"
    >
      <el-table-column label="文章 / 评论" min-width="340">
        <template #default="scope">
          <div v-if="isArticleGroup(scope.row)" class="article-cell">
            <span class="article-accent"></span>
            <div class="article-copy">
              <div class="article-title-line">
                <el-button link type="primary" class="article-title" @click.stop="openArticle(scope.row.articleId)">
                  {{ scope.row.articleTitle }}
                </el-button>
                <el-tag size="small" effect="plain" round>{{ scope.row.commentCount }} 条</el-tag>
              </div>
              <span class="article-id">文章 #{{ scope.row.articleId }}</span>
            </div>
          </div>

          <div v-else class="comment-cell">
            <div class="comment-labels">
              <span class="comment-id">#{{ scope.row.id }}</span>
              <el-tag v-if="scope.row.isAdmin" size="small" type="warning" effect="plain">博主</el-tag>
              <el-tag v-if="scope.row.parentId" size="small" type="info" effect="plain">回复</el-tag>
            </div>
            <p>{{ scope.row.content }}</p>
          </div>
        </template>
      </el-table-column>

      <el-table-column label="作者" width="100">
        <template #default="scope">
          <span v-if="!isArticleGroup(scope.row)" class="author-name">{{ scope.row.author }}</span>
        </template>
      </el-table-column>

      <el-table-column label="邮箱" min-width="160">
        <template #default="scope">
          <span v-if="!isArticleGroup(scope.row)" class="muted-text">{{ scope.row.email }}</span>
        </template>
      </el-table-column>

      <el-table-column label="可见性" width="82" align="center">
        <template #default="scope">
          <el-tag
            v-if="!isArticleGroup(scope.row)"
            size="small"
            :type="scope.row.isPublic ? 'success' : 'info'"
            effect="light"
          >
            {{ scope.row.isPublic ? '公开' : '私密' }}
          </el-tag>
        </template>
      </el-table-column>

      <el-table-column label="环境" width="115">
        <template #default="scope">
          <div v-if="!isArticleGroup(scope.row)" class="environment">
            <span>{{ parseUserAgent(scope.row.ua).os }}</span>
            <small>{{ parseUserAgent(scope.row.ua).browser }}</small>
          </div>
        </template>
      </el-table-column>

      <el-table-column label="时间" width="145">
        <template #default="scope">
          <div v-if="isArticleGroup(scope.row)" class="latest-time">
            <small>最近评论</small>
            <span>{{ scope.row.latestCommentTime }}</span>
          </div>
          <span v-else class="comment-time">{{ scope.row.createTime }}</span>
        </template>
      </el-table-column>
    </el-table>

    <div class="pagination-bar">
      <el-pagination
        v-model:current-page="page.pageNum"
        v-model:page-size="page.pageSize"
        :page-sizes="[10, 20, 50]"
        :total="total"
        layout="total, sizes, prev, pager, next"
        background
        @size-change="handlePageSizeChange"
        @current-change="getData"
      />
    </div>

    <el-drawer v-model="editVisible" title="评论详情" size="500px" destroy-on-close>
      <el-form v-if="selectedComment" :model="editForm" label-width="80px">
        <el-form-item label="作者">
          <el-input v-model="editForm.author" />
        </el-form-item>
        <el-form-item label="邮箱">
          <el-input v-model="editForm.email" />
        </el-form-item>
        <el-form-item label="内容">
          <el-input v-model="editForm.content" type="textarea" :rows="6" />
        </el-form-item>
        <el-form-item label="是否公开">
          <el-switch v-model="editForm.isPublic" />
        </el-form-item>
        <el-form-item label="父评论 ID">
          <el-input :model-value="editForm.parentId || '无'" disabled />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="editVisible = false">关闭</el-button>
        <el-button type="primary" :disabled="!selectedComment" @click="edit">保存修改</el-button>
      </template>
    </el-drawer>

    <el-dialog v-model="replyDialogVisible" title="回复评论" width="560px" destroy-on-close>
      <el-form :model="replyForm" label-width="90px">
        <el-form-item label="回复内容">
          <el-input v-model="replyForm.content" type="textarea" :rows="5" placeholder="输入回复内容" />
        </el-form-item>
        <el-form-item label="是否公开">
          <el-switch v-model="replyForm.isPublic" />
        </el-form-item>

        <div class="emoji-picker">
          <button v-for="emoji in emojis" :key="emoji" type="button" @click="addEmoji(emoji)">{{ emoji }}</button>
        </div>
      </el-form>
      <template #footer>
        <el-button @click="replyDialogVisible = false">取消</el-button>
        <el-button type="primary" :disabled="!replyForm.content.trim()" @click="reply">发送回复</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { nextTick, onMounted, reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import { ChatDotRound, DArrowRight, Delete, Edit, Refresh } from '@element-plus/icons-vue'
import { deleteComment, editComment, getAllComment, replyComment } from '@/api/admin'
import type { ArticleCommentGroup, Comment } from '@/types/Comment'
import { emojis } from '@/types/Constant'
import { notify } from '@/utils/notification'

type CommentTableRow = ArticleCommentGroup | Comment

const router = useRouter()
const tableRef = ref<any>()
const isLoading = ref(false)
const expanded = ref(false)
const commentGroups = ref<ArticleCommentGroup[]>([])
const total = ref(0)
const selectedComment = ref<Comment>()
const editVisible = ref(false)
const replyDialogVisible = ref(false)

const page = reactive({
  pageSize: 10,
  pageNum: 1,
})

const replyForm = reactive({
  content: '',
  isPublic: true,
})

const editForm = reactive<Comment>({
  articleId: 0,
  author: '',
  email: '',
  content: '',
  isPublic: true,
  parentId: null,
})

const isArticleGroup = (row: CommentTableRow): row is ArticleCommentGroup => 'articleTitle' in row

const getRowKey = (row: CommentTableRow) => isArticleGroup(row)
  ? `article-${row.articleId}`
  : `comment-${row.id}`

const getRowClassName = ({ row }: { row: CommentTableRow }) => isArticleGroup(row) ? 'article-row' : 'comment-row'

const openArticle = (articleId: number) => {
  window.open(router.resolve({ name: 'ArticleInfo', params: { id: articleId } }).href)
}

const parseUserAgent = (ua?: string) => {
  const [os = '未知环境', browser = '未知浏览器'] = (ua || '').split(';').map(item => item.trim())
  return { os, browser }
}

const handleCurrentChange = (row?: CommentTableRow) => {
  selectedComment.value = row && !isArticleGroup(row) ? row : undefined
}

const getData = async () => {
  isLoading.value = true
  selectedComment.value = undefined
  try {
    const res = await getAllComment(page.pageNum, page.pageSize)
    commentGroups.value = res.list
    total.value = res.total
    expanded.value = false
  } finally {
    isLoading.value = false
  }
}

const refreshData = async () => {
  await getData()
  notify.success('评论信息已刷新')
}

const handlePageSizeChange = () => {
  page.pageNum = 1
  getData()
}

const toggleRowRecursively = (row: CommentTableRow) => {
  if (!row.children?.length) return
  tableRef.value?.toggleRowExpansion(row, expanded.value)
  row.children.forEach(toggleRowRecursively)
}

const expandAllRows = async () => {
  expanded.value = !expanded.value
  await nextTick()
  commentGroups.value.forEach(toggleRowRecursively)
}

const openEdit = () => {
  if (!selectedComment.value) return
  Object.assign(editForm, selectedComment.value)
  editVisible.value = true
}

const edit = async () => {
  if (!selectedComment.value) return
  await editComment(editForm)
  editVisible.value = false
  notify.success('评论已更新')
  await getData()
}

const openReply = () => {
  if (!selectedComment.value) return
  replyForm.content = ''
  replyForm.isPublic = true
  replyDialogVisible.value = true
}

const addEmoji = (value: string) => {
  replyForm.content += value
}

const reply = async () => {
  const comment = selectedComment.value
  if (!comment?.id || !replyForm.content.trim()) return
  await replyComment(replyForm.content.trim(), replyForm.isPublic, comment.id, comment.articleId)
  replyDialogVisible.value = false
  notify.success('回复已发送')
  await getData()
}

const deleteSelectedComment = () => {
  const comment = selectedComment.value
  if (!comment?.id) return

  ElMessageBox.confirm('删除后，该评论下的所有回复也会一并删除。', '删除评论', {
    confirmButtonText: '删除',
    cancelButtonText: '取消',
    confirmButtonClass: 'el-button--danger',
    type: 'warning',
  }).then(async () => {
    await deleteComment(comment.id!)
    notify.success('评论已删除')
    await getData()
  }).catch(() => undefined)
}

onMounted(getData)
</script>

<style scoped lang="less">
.comment-manager {
  --comment-ink: #1f2a3d;
  --comment-muted: #7d889a;
  --article-wash: #f2f7ff;
  --article-line: #b9d3f7;
  min-width: 880px;
}

.page-toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 24px;
  padding: 2px 16px 14px;
}

.title-block {
  h2 {
    margin: 0;
    color: var(--comment-ink);
    font-size: 20px;
    font-weight: 650;
    letter-spacing: .02em;
  }

  p {
    margin: 5px 0 0;
    color: var(--comment-muted);
    font-size: 13px;
  }
}

.toolbar-actions {
  display: flex;
  align-items: center;
  gap: 10px;
}

.comment-actions {
  margin-left: 6px;
}

.comment-table {
  width: 100%;
  border: 1px solid #e9edf3;
  border-radius: 8px;
  background: #fff;
  box-shadow: 0 5px 22px rgb(31 42 61 / 5%);
  overflow: hidden;
}

.article-cell {
  display: flex;
  min-width: 0;
  flex: 1;
  align-items: stretch;
  gap: 12px;
  min-height: 42px;
}

.article-accent {
  width: 3px;
  flex: 0 0 3px;
  border-radius: 3px;
  background: #4f8edc;
}

.article-copy {
  display: flex;
  min-width: 0;
  flex-direction: column;
  justify-content: center;
  gap: 3px;
}

.article-title-line {
  display: flex;
  min-width: 0;
  align-items: center;
  gap: 10px;
}

.article-title {
  display: block;
  max-width: 260px;
  height: auto;
  padding: 0;
  overflow: hidden;
  color: #245f9f;
  font-size: 15px;
  font-weight: 650;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.article-id,
.muted-text,
.comment-time {
  color: var(--comment-muted);
  font-size: 12px;
}

.comment-cell {
  display: flex;
  min-width: 0;
  flex: 1;
  flex-direction: column;
  justify-content: center;
  padding: 2px 0;

  p {
    display: -webkit-box;
    margin: 5px 0 0;
    overflow: hidden;
    color: #344054;
    line-height: 1.55;
    -webkit-box-orient: vertical;
    -webkit-line-clamp: 2;
  }
}

.comment-labels {
  display: flex;
  align-items: center;
  gap: 6px;
}

.comment-id {
  color: #98a2b3;
  font-size: 11px;
  font-variant-numeric: tabular-nums;
}

.author-name {
  color: var(--comment-ink);
  font-weight: 550;
}

.environment,
.latest-time {
  display: flex;
  flex-direction: column;
  gap: 3px;

  span {
    color: #475467;
    font-size: 12px;
  }

  small {
    color: #98a2b3;
    font-size: 11px;
  }
}

.latest-time {
  span {
    font-variant-numeric: tabular-nums;
  }
}

.pagination-bar {
  display: flex;
  justify-content: flex-end;
  padding: 14px 8px 0;
}

.emoji-picker {
  display: grid;
  grid-template-columns: repeat(10, 1fr);
  gap: 5px;
  margin-left: 90px;

  button {
    display: grid;
    width: 32px;
    height: 32px;
    padding: 0;
    border: 1px solid transparent;
    border-radius: 6px;
    background: #f5f7fa;
    cursor: pointer;
    font-size: 17px;
    place-items: center;

    &:hover,
    &:focus-visible {
      border-color: #9ec5f4;
      background: #edf5ff;
      outline: none;
    }
  }
}

:deep(.article-row > td.el-table__cell) {
  border-bottom-color: var(--article-line) !important;
  background: var(--article-wash) !important;
}

:deep(.article-row:hover > td.el-table__cell) {
  background: #eaf3ff !important;
}

:deep(.comment-table .el-table__body td:first-child > .cell) {
  display: flex;
  align-items: center;
}

:deep(.comment-table .el-table__expand-icon) {
  flex: 0 0 auto;
  margin-right: 4px;
}

:deep(.comment-table .el-table__placeholder),
:deep(.comment-table .el-table__indent) {
  flex: 0 0 auto;
}

@media (prefers-reduced-motion: reduce) {
  :deep(*) {
    scroll-behavior: auto !important;
    transition-duration: .01ms !important;
  }
}
</style>
