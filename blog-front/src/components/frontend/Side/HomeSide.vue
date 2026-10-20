<template>
  <div class="pt-6 h-screen">
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

    <!--
      桌宠贴侧栏底部。心情气泡不在文档流里：Pet.vue 每帧把她头顶的坐标写进 usePetMood，
      PetMood.vue 照着那个锚点绝对定位，所以她走动、蹲下、跳起来时气泡都跟着头走。
      两个组件都不吃点击，只有角色本身接收双击。

      这一层不带 relative：根节点不能加 w-full（会把上面「文章/分类/标签」那行从 150px 撑成整栏），
      所以让它往上找定位祖先 —— 侧栏本身是 sticky，天然是 positioned，桌宠就拿到整栏宽度。
    -->
    <div class="pointer-events-none absolute inset-x-0 bottom-[2%]">
      <div class="relative w-full">
        <PetMood />
        <Pet class="pointer-events-auto" />
      </div>
    </div>
  </div>
</template>
<script setup lang="ts">
import Pet from '@/components/frontend/Pet.vue'
import PetMood from '@/components/frontend/PetMood.vue'
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
