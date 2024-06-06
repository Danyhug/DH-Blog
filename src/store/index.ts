import { getArticleCategoryList, getArticleTagList } from '@/api/api'
import { Category } from '@/types/Category'
import { Tag } from '@/types/Tag'
import { defineStore } from 'pinia'
import { reactive } from 'vue'

export const useAdminStore = defineStore('admin', () => {
  const tags = reactive<Tag[]>([])
  const categories = reactive<Category[]>([])

  const getCategories = async () => {
    const data = await getArticleCategoryList();
    Object.assign(categories, data)
  };

  // 获取标签列表
  const getTags = async () => {
    const data = await getArticleTagList();
    Object.assign(tags, data)
  };

  return {
    tags,
    categories,
    getCategories,
    getTags
  }
})

export const useUserStore = defineStore('user', () => { })
