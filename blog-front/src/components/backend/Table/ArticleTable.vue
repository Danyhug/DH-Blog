<template>
  <el-table :data="props.articles" style="width: 100%;">
    <el-table-column prop="id" label="编号" width="70" />
    <el-table-column label="文章标题" min-width="220">
      <template #default="scope">
        <!-- 站长手写是常态，只有 agent 写的才标出来，省下一整列 -->
        <div class="flex flex-wrap items-center gap-1.5">
          <span>{{ scope.row.title }}</span>
          <el-tag v-if="scope.row.authorType === 'agent'" size="small" type="warning" effect="plain">
            AI · {{ scope.row.authorName }}
          </el-tag>
        </div>
      </template>
    </el-table-column>
    <el-table-column prop="categoryName" label="分类">
      <template #default="scope">
        <el-tag type="primary">{{ scope.row.categoryName }}</el-tag>
      </template>
    </el-table-column>

    <el-table-column prop="views" label="浏览数" />
    <el-table-column prop="wordNum" label="字数" />
    <el-table-column prop="createTime" label="发布时间" width="170" />
    <el-table-column prop="updateTime" label="更新时间" width="170" />
    <el-table-column fixed="right" label="操作" width="190">
      <template #default="scope">
        <el-button link type="primary" size="large" @click.prevent="edit(scope.row.id)">编辑</el-button>
        <el-button link type="success" size="small" @click.prevent="generateTags(scope.row.id)" :loading="generatingTags[scope.row.id]">
          <el-icon><MagicStick /></el-icon> AI标签
        </el-button>
        <br>
        <el-button link type="warning" size="small" @click.prevent="generateSummary(scope.row.id)" :loading="generatingSummary[scope.row.id]">
          <el-icon><MagicStick /></el-icon> AI摘要
        </el-button>
        <el-button link type="danger" size="small" @click.prevent="removeArticle(scope.row)"
          :loading="deleting[scope.row.id]">删除</el-button>
      </template>
    </el-table-column>
  </el-table>
</template>
<script lang="ts" setup>
import { useRouter } from 'vue-router'
import { reactive } from 'vue'
import { MagicStick } from '@element-plus/icons-vue'
import { ElMessageBox } from 'element-plus'
import { generateAITags, generateAISummary, deleteArticle } from '@/api/admin'
import { notify } from '@/utils/notification'
import { Article } from '@/types/Article'
import { Tag } from '@/types/Tag'

const props = defineProps(['articles'])
const router = useRouter()
// 使用Record类型来定义generatingTags对象
const generatingTags = reactive<Record<number, boolean>>({})
const generatingSummary = reactive<Record<number, boolean>>({})
const deleting = reactive<Record<number, boolean>>({})

const edit = (id: number) => {
  router.push({ name: 'publish', query: { articleId: id } });
}

const generateTags = async (id: number) => {
  try {
    // 设置加载状态
    generatingTags[id] = true
    
    // 调用API生成标签
    await generateAITags(id)
    
    // 显示成功消息
    notify.success({
      message: '标签生成成功，请刷新页面查看'
    })
    
    // 通知父组件刷新数据
    emit('refresh')
  } catch (error) {
    console.error('生成标签失败:', error)
    notify.error({
      message: '标签生成失败，请稍后重试'
    })
  } finally {
    // 无论成功失败，都取消加载状态
    generatingTags[id] = false
  }
}

// 定义emit以便通知父组件刷新
const emit = defineEmits(['refresh'])

const generateSummary = async (id: number) => {
  try {
    generatingSummary[id] = true
    await generateAISummary(id)
    notify.success({
      message: '摘要生成任务已提交，稍后刷新页面查看'
    })
    emit('refresh')
  } catch (error) {
    console.error('生成摘要失败:', error)
    notify.error({
      message: '摘要生成失败，请稍后重试'
    })
  } finally {
    generatingSummary[id] = false
  }
}
const removeArticle = async (article: Article<Tag>) => {
  const id = article.id as number
  try {
    await ElMessageBox.confirm(
      `确定要删除文章「${article.title}」吗？`,
      '警告',
      { confirmButtonText: '确定删除', cancelButtonText: '取消', type: 'warning' }
    )
  } catch {
    // 用户取消，不做处理
    return
  }

  try {
    deleting[id] = true
    await deleteArticle(id)
    notify.success({ message: '文章已删除' })
    emit('refresh')
  } catch (error) {
    // 失败原因由 axios 拦截器统一提示（如「文章不存在」），这里不再重复弹窗
    console.error('删除文章失败:', error)
  } finally {
    deleting[id] = false
  }
}

</script>
