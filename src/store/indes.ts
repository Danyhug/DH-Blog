import { getArticleCategoryList, getArticleTagList } from '@/api/api'
import { Category } from '@/types/Category'
import { Tag } from '@/types/Tag'
import { defineStore } from 'pinia'
import { reactive } from 'vue'

const useAdminStore = defineStore('admin', () => {
  const tags = reactive<Tag[]>([])
  const categories = reactive<Category[]>([])

  const getCategories = async () => {
    const data = await getArticleCategoryList();
    categories.push(...data);
  };
  
  // 获取标签列表
  const getTags = async () => {
    const data = await getArticleTagList();
    tags.push(...data);
  };

  return {
    tags,
    categories,
    getCategories,
    getTags
  }
})

export default useAdminStore