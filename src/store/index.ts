import { getArticleCategoryList, getArticleTagList } from '@/api/user'
import { ArticleModel } from '@/types/ArticleModel'
import { Category } from '@/types/Category'
import { Tag } from '@/types/Tag'
import { defineStore } from 'pinia'
import { reactive, ref } from 'vue'
import { MdInit } from '@/types/MdEditor'
import { Article } from '@/types/Article'
import { Page } from '@/types/Page'

export const useSystemStore = defineStore('system', () => {
  const mdEditorInit = reactive<MdInit>({
    codeFoldable: true,
    editorId: 'dh-editor',
    previewTheme: 'cyanosis',
    theme: 'light'
  })

  return {
    mdEditorInit
  }
})

export const useAdminStore = defineStore('admin', () => {
  const tags = reactive<Tag[]>([])
  const categories = reactive<Category[]>([])
  const online = ref(0)
  
  const getCategories = async () => {
    const data = await getArticleCategoryList();
    if (data.length === 0) {
      categories.length = 0
    } else {
      Object.assign(categories, data)
    }
  };

  // 获取标签列表
  const getTags = async () => {
    const data = await getArticleTagList();
    if (data.length === 0) {
      tags.length = 0
    } else {
      Object.assign(tags, data)
    }
  };

  return {
    tags,
    categories,
    getCategories,
    getTags,
    online
  }
})

export const useUserStore = defineStore('user', () => {
  const homeShowComponent = ref('home')

  // 首页上方展示内容（文章详情上面）
  interface HomeHeaderInfo {
    title: string
    created: string,
    wordNum: number,
    tags: any,
    thumbnailUrl: string,
    timConSum: string
  }
  const homeHeaderInfo = reactive<HomeHeaderInfo>({
    title: '',
    created: '',
    // 总字数
    wordNum: 0,
    // 阅读时长
    timConSum: '0',
    thumbnailUrl: '',
    tags: []
  })

  // 文章状态控制
  const aritcleModel = reactive<ArticleModel>({
    isDarkMode: false,
    isFullPreview: false
  })

  // 首页文章列表
  const articleList = reactive<Article<Tag[]>[]>([])
  const page = reactive<Page>({
    pageNum: 1,
    pageSize: 7,
    total: 0
  })

  return {
    homeShowComponent,
    homeHeaderInfo,
    aritcleModel,
    articleList,
    page
  }
})
