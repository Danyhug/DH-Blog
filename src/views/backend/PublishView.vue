<template>
  <div class="top-container">
    <div class="left">
      <el-tabs tabPosition="left" v-model="activeTag">
        <el-tab-pane label="分类选择" name="first">
          <el-radio-group v-model.number="article.categoryId">
            <el-radio size="large" border v-for="category in categories" :key="category.id" :value="category.id"
              :label="category.name">
            </el-radio>
          </el-radio-group>
        </el-tab-pane>
        <el-tab-pane label="标签选择" name="second">
          <!-- 标签： -->
          <el-checkbox-group size="large" v-model="article.tags">
            <el-checkbox-button v-for="tag in tags" :key="tag.id" :value="tag.slug"
              :label="tag.name"></el-checkbox-button>
          </el-checkbox-group>
        </el-tab-pane>
        <el-tab-pane label="附加信息" name="third">
          <el-upload class="avatar-uploader" :action="SERVER_URL + '/article/upload'" :show-file-list="false"
            :on-success="handleAvatarSuccess" :before-upload="beforeAvatarUpload">
            <img v-if="imageUrl" :src="imageUrl" class="avatar" />
            <el-icon v-else class="avatar-uploader-icon">
              <Plus />
            </el-icon>
          </el-upload>
        </el-tab-pane>
      </el-tabs>
    </div>

    <div class="right">
      <el-divider content-position="center">
        <p class="tip">文章标题</p>
      </el-divider>
      <el-input placeholder="输入文章标题" clearable class="title-input" v-model="article.title"></el-input>
      <div class="btns">
        <el-button type="warning" @click="clear(true)">清空</el-button>
        <el-button type="primary" @click="submit">发布</el-button>
      </div>
    </div>
  </div>

  <el-divider content-position="center">
    <p class="tip">文章内容</p>
  </el-divider>
  <MdEditor v-model="article.content" :toolbars="toolbars" previewTheme="github" @onUploadImg="onUploadImg">
    <template #defToolbars>
      <Emoji :emojis="emojis" :selectAfterInsert="false">
        <template #trigger>
          <span style="font-size: 1.5rem;">🐶</span>
        </template>
      </Emoji>
    </template>
    <template #defFooters>
    </template>
  </MdEditor>
</template>

<script setup lang="ts">
import { addArticle, getArticleCategoryList, getArticleInfo, getArticleTagList, updateArticle, uploadFile } from '@/api/api';
import { Article } from '@/types/Article';
import { Category } from '@/types/Category';
import { Tag } from '@/types/Tag';
import { useRoute } from 'vue-router'
import { onMounted, reactive, ref } from 'vue';
import type { UploadProps } from 'element-plus'
import { SERVER_URL } from '@/types/Constant'
import { Plus } from '@element-plus/icons-vue'
import { Emoji } from '@vavt/v3-extension'
import { toolbars, emojis } from '@/types/Constant'

const route = useRoute()
const articleId = route.query?.articleId

// 切换标签
const activeTag = ref('first')

const article = reactive<Article<String>>({
  title: '',
  content: ``,
  categoryId: -1,
  tags: [],
  thumbnailUrl: ''
});
const categories = reactive<Category[]>([]);
const tags = reactive<Tag[]>([]);

// 清空文章信息
const clear = (manual: boolean = false) => {
  function func() {
    article.title = ''
    article.content = ''
    article.categoryId = -1
    article.tags = []
  }

  if (manual) {
    ElMessageBox.confirm(
      '确定要清空所有信息吗？此操作不可恢复',
      '警告',
      {
        confirmButtonText: '确定',
        cancelButtonText: '取消',
        type: 'warning',
      }
    ).then(() => {
      func()
    })
  } else {
    func()
  }
}

// 获取分类列表
const getCategories = async () => {
  const data = await getArticleCategoryList();
  categories.push(...data);
};

// 获取标签列表
const getTags = async () => {
  const data = await getArticleTagList();
  tags.push(...data);
};

const onUploadImg = async (files: any[], callback: (arg0: any[]) => void) => {
  const res = await Promise.all(
    files.map((file) => {
      return new Promise((resolve, reject) => {
        const form = new FormData();
        form.append('file', file);
        // 上传文件
        uploadFile(form)
          .then((res) => {
            ElMessage.success('上传成功');
            resolve(res);
          })
          .catch((err) => {
            ElMessage.error('上传失败');
            reject(err);
          })
      });
    })
  ) as string[];
  callback(res.map(item => `${SERVER_URL}/articleUpload/${item}`))
};

// 提交文章
const submit = () => {
  console.log(article)
  if (article.categoryId == -1) {
    return ElMessage.error({
      message: '未选择文章分类',
      plain: true,
    })
  }

  let el: HTMLElement | null = document.querySelector('.md-editor-preview');
  let count = el !== null ? el.innerText.length : 0;
  article.wordNum = count;

  // 上传新文章
  if (articleId == null) {
    addArticle(article).then(_ => {
      ElMessage.success({
        message: '发布成功',
        plain: true,
      })
      clear()
    })
  } else {
    // 更新文章
    updateArticle(article).then(_ => {
      ElMessage.success({
        message: '已成功更新文章',
        plain: true,
      })
      clear()
    })
  }
}

onMounted(() => {
  getCategories().catch(error => {
    ElMessage.error({
      message: '获取分类失败' + error.message,
      plain: true,
    })
  });
  getTags().catch(error => {
    ElMessage.error({
      message: '获取标签失败' + error.message,
      plain: true,
    })
  })

  // 获取路由部分是否为edit?id=格式
  if (articleId) {
    // 获取文章详情
    getArticleInfo(articleId as String).then((res: Article<Tag>) => {
      let articleTemp = { ...res, tags: [] }
      Object.assign(article, articleTemp);
      if (res?.tags) {
        for (let tag of res.tags) {
          if (article.tags) {
            article.tags.push(tag.slug)
          }
        }
      }
    })
  }
})

// 上传文件

const imageUrl = ref('')

const handleAvatarSuccess: UploadProps['onSuccess'] = (
  response,
  uploadFile
) => {
  imageUrl.value = URL.createObjectURL(uploadFile.raw!)
  ElMessage.success({
    message: '上传文件成功',
    plain: true,
  })

  // 保存缩略图图片url
  article.thumbnailUrl = response.data
}

const beforeAvatarUpload: UploadProps['beforeUpload'] = (rawFile) => {
  if (rawFile.size / 1024 / 1024 > 10) {
    ElMessage.error('图片不能超过10兆!')
    return false
  }
  return true
}

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

.avatar-uploader .avatar {
  max-width: 246px;
  max-height: 138px;
  display: block;
}
</style>

<style>
.avatar-uploader .el-upload {
  border: 1px dashed var(--el-border-color);
  border-radius: 6px;
  cursor: pointer;
  position: relative;
  overflow: hidden;
  transition: var(--el-transition-duration-fast);
}

.avatar-uploader .el-upload:hover {
  border-color: var(--color-blue);
}

.el-icon.avatar-uploader-icon {
  font-size: 28px;
  color: #8c939d;
  width: 246px;
  height: 138px;
  text-align: center;
}

.emojis {
  display: grid;
  font-size: 1.5rem;
  grid-template-columns: repeat(5, 1fr);
  grid-gap: 12px;

  li {
    list-style: none;
    cursor: pointer;
  }
}
</style>