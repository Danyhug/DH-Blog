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
          <el-upload class="avatar-uploader" :http-request="handleFileUpload" :show-file-list="false"
            :before-upload="beforeAvatarUpload">
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

      <div class="form-box">
        <el-input placeholder="输入文章标题" clearable class="title-input" v-model="article.title" />
        <el-switch class="article-status" style="--el-switch-on-color: #13ce66; --el-switch-off-color: #4aa0ff"
          inactive-text="公开" active-text="私密" v-model="article.isLocked" @change="changeArticleStatus" />
      </div>

      <div class="btns">
        <!-- 本地保存提示信息 -->
        <div class="save-text">{{ autoSaveText }}</div>

        <el-button type="warning" @click="clear(true)">清空</el-button>
        <el-button type="primary" @click="submit">发布</el-button>
      </div>
    </div>
  </div>

  <el-divider content-position="center">
    <p class="tip">文章内容</p>
  </el-divider>
  <MdEditor ref="editor" v-model="article.content" :toolbars="toolbars" :previewTheme="system.mdEditorInit.previewTheme"
    :codeFoldable="system.mdEditorInit.codeFoldable" @onUploadImg="onUploadImg">
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
import { useRoute } from 'vue-router'
import { onMounted, reactive, ref } from 'vue';
import type { UploadProps } from 'element-plus'
import { SERVER_URL } from '@/types/Constant'
import { Plus } from '@element-plus/icons-vue'
import { Emoji } from '@vavt/v3-extension'

import {
  addArticle, updateArticle, uploadFile
} from '@/api/admin';

import { getArticleCategoryList, getArticleTagList } from '@/api/user'
import { getArticleInfo } from '@/api/admin'

import { toolbars, emojis } from '@/types/Constant'
import { Article } from '@/types/Article';
import { Category } from '@/types/Category';
import { Tag } from '@/types/Tag';
import { useSystemStore } from '@/store';

const system = useSystemStore()
const route = useRoute()
const articleId = route.query?.articleId
// 保存文章的key
const article_save_key = `DHBlog_Article_${articleId == null ? 'Draft' : articleId.toString()}`;

// 切换标签
const activeTag = ref('first')
const autoSaveText = ref('')

const article = reactive<Article<String>>({
  title: '',
  content: ``,
  categoryId: -1,
  tags: [],
  thumbnailUrl: '',
  isLocked: false,

});
const categories = reactive<Category[]>([]);
const tags = reactive<Tag[]>([]);

// 切换文章状态 公开 > 私密
const changeArticleStatus = (val: boolean) => {
  if (!val) return article.lockPassword = '';

  ElMessageBox.prompt("请输入文章密钥：", "提示").then(res => {
    article.lockPassword = res.value
    ElMessage.success(`设置密钥为 ${article.lockPassword}`)
  }).catch(_ => article.isLocked = false)
}

// 自动保存
const autoSave = () => {
  if (article.content.length === 0) return;
  localStorage.setItem(article_save_key, JSON.stringify(article));

  // 创建一个Date对象，表示当前时间
  var now = new Date();
  // 获取当前年份、月份（+1因为月份是从0开始的）、日期、小时、分钟和秒
  var year = now.getFullYear();
  var month = String(now.getMonth() + 1).padStart(2, '0'); // 使用padStart确保月份是两位数
  var day = String(now.getDate()).padStart(2, '0'); // 使用padStart确保日期是两位数
  var hours = String(now.getHours()).padStart(2, '0'); // 使用padStart确保小时是两位数
  var minutes = String(now.getMinutes()).padStart(2, '0'); // 使用padStart确保分钟是两位数
  var seconds = String(now.getSeconds()).padStart(2, '0'); // 使用padStart确保秒是两位数
  // 使用模板字符串拼接日期和时间
  var formattedTime = `${year}/${month}/${day} ${hours}:${minutes}:${seconds}`;
  // 更新提示信息
  autoSaveText.value = `${formattedTime} 自动保存`;
}
// 30秒自动保存一次
setInterval(autoSave, 1000 * 30);

// 清空文章信息
const clear = (manual: boolean = false) => {
  function func() {
    article.title = ''
    article.content = ''
    article.categoryId = -1
    article.tags = []
    // 清空本地
    localStorage.removeItem(article_save_key)
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
  callback(res.map(item => `${SERVER_URL}/${item}`))
};

// 提交文章
const submit = () => {
  if (article.categoryId == -1) {
    return ElMessage.error({
      message: '未选择文章分类',
      plain: true,
    })
  }

  let el: HTMLElement | null = document.querySelector('.md-editor-footer-item span');
  let count = el !== null ? parseInt(el.innerText) : 0;
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

  // 尝试获取本地文章信息
  const temp = localStorage.getItem(article_save_key) || null;

  // 获取路由部分是否为edit?id=格式 此时是更新信息，获取文章信息
  if (articleId) {
    // 弹窗询问用户是否继续编辑草稿还是重新获取文章信息
    const handleArticleId = () => {
      if (temp === null) {
        getArticleById();
      } else {
        ElMessageBox.confirm(
          '是否继续使用上次草稿？',
          '提示',
          {
            confirmButtonText: '继续编辑',
            cancelButtonText: '重新获取',
            type: 'warning'
          }
        ).then(() => {
          Object.assign(article, JSON.parse(temp as string));
        }).catch(() => {
          clear();
          getArticleById();
        });
      }
    };

    const getArticleById = () => {
      getArticleInfo(articleId as string).then((res: Article<Tag>) => {
        const articleTemp = { ...res, tags: res.tags?.map(tag => tag.name) || [] };
        Object.assign(article, articleTemp);
      });
    };

    handleArticleId();
    return;
  }

  // 如果没有 articleId，则直接从本地存储加载文章信息
  if (temp) {
    Object.assign(article, JSON.parse(temp as string));
  }
})

// 上传文件
const imageUrl = ref('')

const handleFileUpload = async (event: any) => {
  const file = event.file;
  const form = new FormData();
  form.append('file', file);

  try {
    const res = await uploadFile(form);
    imageUrl.value = URL.createObjectURL(file);
    ElMessage.success('上传文件成功');

    // 保存缩略图图片url
    article.thumbnailUrl = res.toString();
  } catch (error) {
    ElMessage.error('上传文件失败');
  }
};

const beforeAvatarUpload: UploadProps['beforeUpload'] = (rawFile) => {
  if (rawFile.size / 1024 / 1024 > 10) {
    ElMessage.error('图片不能超过10兆!')
    return false
  }
  return true
}

</script>

<style scoped lang="less">
.top-container {
  display: flex;
  justify-content: space-between;

  &>div {
    height: 126px;
    width: 46%;
  }

  .right {

    .form-box {
      display: flex;
      justify-content: space-between;

      .el-input {
        width: calc(100% - 150px);
      }

      .article-status {
        width: 120px;
      }
    }

    .btns {
      .save-text {
        float: left;
        font-size: 14px;
        color: #999;
      }

      margin: 20px 0;
      text-align: right;
    }
  }
}

.el-divider {
  background-color: #1E9FFF;
}

:deep(.cm-editor .cm-line) {
  font-family: '微软雅黑', 'Georgia';
  line-height: 25px !important;
  font-size: 16px;
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
