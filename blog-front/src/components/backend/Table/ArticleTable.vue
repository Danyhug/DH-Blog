<template>
  <el-table :data="props.articles" style="width: 100%;">
    <el-table-column prop="id" label="编号" width="70" />
    <el-table-column prop="title" label="文章标题" width="145" />
    <el-table-column prop="categoryName" label="分类">
      <template #default="scope">
        <el-tag type="primary">{{ scope.row.categoryName }}</el-tag>
      </template>
    </el-table-column>

    <el-table-column prop="tags" label="标签" min-width="120">
      <template #default="scope">
        <div style="display: flex; flex-wrap: wrap; gap: 4px; align-items: center;">
          <template v-if="scope.row.tags && scope.row.tags.length > 0">
            <el-tooltip 
              v-if="scope.row.tags.length > 3" 
              placement="top" 
              :content="scope.row.tags.map(t => t.name).join(', ')"
            >
              <div>
                <el-tag 
                  v-for="(tag, index) in scope.row.tags.slice(0, 3)" 
                  :key="tag.id"
                  :type="tagColors[index % tagColors.length]" 
                  effect="plain" 
                  size="small"
                  style="margin: 2px;"
                >
                  {{ tag.name }}
                </el-tag>
                <el-tag type="info" effect="plain" size="small" style="margin: 2px;">
                  +{{ scope.row.tags.length - 3 }}
                </el-tag>
              </div>
            </el-tooltip>
            <template v-else>
              <el-tag 
                v-for="(tag, index) in scope.row.tags" 
                :key="tag.id"
                :type="tagColors[index % tagColors.length]" 
                effect="plain" 
                size="small"
                style="margin: 2px;"
              >
                {{ tag.name }}
              </el-tag>
            </template>
          </template>
          <template v-else>
            <span style="color: #909399; font-size: 12px;">无标签</span>
          </template>
        </div>
      </template>
    </el-table-column>
    <el-table-column prop="views" label="浏览数" />
    <el-table-column prop="wordNum" label="字数" />
    <el-table-column prop="createTime" label="发布时间" width="170" />
    <el-table-column prop="updateTime" label="更新时间" width="170" />
    <el-table-column fixed="right" label="操作" width="150">
      <template #default="scope">
        <el-button link type="primary" size="large" @click.prevent="edit(scope.row.id)">编辑</el-button>
        <el-button link type="success" size="small" @click.prevent="generateTags(scope.row.id)" :loading="generatingTags[scope.row.id]">
          <el-icon><MagicStick /></el-icon> AI标签
        </el-button>
        <br>
        <el-button link type="danger" size="small">删除</el-button>
      </template>
    </el-table-column>
  </el-table>
</template>
<script lang="ts" setup>
import { useRouter } from 'vue-router'
import { ref, reactive } from 'vue'
import { ElMessage } from 'element-plus'
import { MagicStick } from '@element-plus/icons-vue'
import { generateAITags } from '@/api/admin'

const props = defineProps(['articles'])
const router = useRouter()
// 使用Record类型来定义generatingTags对象
const generatingTags = reactive<Record<number, boolean>>({})

const edit = (id: number) => {
  router.push({ name: 'publish', query: { articleId: id } });
}

const generateTags = async (id: number) => {
  try {
    // 设置加载状态
    generatingTags[id] = true
    
    // 调用API生成标签
    const response = await generateAITags(id)
    
    // 显示成功消息
    ElMessage({
      type: 'success',
      message: '标签生成成功，请刷新页面查看'
    })
    
    // 通知父组件刷新数据
    emit('refresh')
  } catch (error) {
    console.error('生成标签失败:', error)
    ElMessage({
      type: 'error',
      message: '标签生成失败，请稍后重试'
    })
  } finally {
    // 无论成功失败，都取消加载状态
    generatingTags[id] = false
  }
}

// 定义emit以便通知父组件刷新
const emit = defineEmits(['refresh'])

// 标签颜色数组
const tagColors = ['primary', 'success', 'warning', 'danger', 'info']
</script>