<template>
  <div class="pt-6 relative h-screen">
    <img class="block mx-auto w-[9.375rem] h-[9.375rem] rounded-[50%]" :src="site.avatar || defaultAvatar" :alt="site.blog_title">
    <div class="leading-[2.2em]">
      <p class="sub-title">{{ site.signature }}</p>
      <ul class="mt-6 mb-5 flex justify-between">
        <li class="text-grey-7 dark:text-[#d4d4d4] list-none">
          <a href="">
            <p class="num font-bold text-[22px]">{{ data.articleCount }}</p>
            文章
          </a>
        </li>
        <span class="block border-l border-grey-4 dark:border-[#444]"></span>
        <li class="text-grey-7 dark:text-[#d4d4d4] list-none">
          <a href="">
            <p class="num font-bold text-[22px]">{{ data.categoryCount }}</p>
            分类
          </a>
        </li>
        <span class="block border-l border-grey-4 dark:border-[#444]"></span>
        <li class="text-grey-7 dark:text-[#d4d4d4] list-none">
          <a href="">
            <p class="num font-bold text-[22px]">{{ data.tagCount }}</p>
            标签
          </a>
        </li>
      </ul>

      <ul class="text-center">
        <li v-if="site.github_link" class="inline-block px-[0.9375rem]">
          <a :href="site.github_link" target="_blank">
            <Icon iconName="icon-github1" iconSize="2.3"></Icon>
          </a>
        </li>
        <li v-if="site.bilibili_link" class="inline-block px-[0.9375rem]">
          <a :href="site.bilibili_link" target="_blank">
            <Icon iconName="icon-bilibili" iconSize="2.2" class="fill-[rgb(250,116,153)]"></Icon>
          </a>
        </li>
      </ul>
    </div>

    <Pet />
  </div>
</template>
<script setup lang="ts">
import Pet from '@/components/frontend/Pet.vue'
import { getOverview } from '@/api/user';
import { OverView } from '@/types/DashBoard';
import { useSiteStore } from '@/store';
import defaultAvatar from '@/assets/images/logo.jpg'
import { storeToRefs } from 'pinia';

const siteStore = useSiteStore()
const { site } = storeToRefs(siteStore)

const data = reactive<OverView>({
  articleCount: 0,
  categoryCount: 0,
  commentCount: 0,
  tagCount: 0
})

onMounted(async () => {
  await siteStore.loadSite()
  const overview = await getOverview()
  Object.assign(data, overview)
})
</script>
