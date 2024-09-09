import { getArticleCategoryList, getArticleTagList } from '@/api/api'
import { ArticleModel } from '@/types/ArticleModel'
import { Category } from '@/types/Category'
import { Tag } from '@/types/Tag'
import { defineStore } from 'pinia'
import { reactive, ref } from 'vue'
import { MdInit } from '@/types/MdEditor'

export const useSystemStore = defineStore('system', () => {
  const mdEditorInit = reactive<MdInit>({
    codeFoldable: false,
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
    getTags
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

  return {
    homeShowComponent,
    homeHeaderInfo,
    aritcleModel
  }
})
