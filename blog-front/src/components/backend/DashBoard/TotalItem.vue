<template>
  <!-- 总览 -->
  <ul class="flex items-center justify-between p-[16px] mb-[10px]">
    <li class="flex justify-between w-[23.6%] h-[90px] bg-white rounded-[var(--dh-admin-border-radius-big)] px-[18px] pt-[20px] list-none shadow-[var(--dh-admin-border-shadow-shallow)]">
      <div class="w-[70%]">
        <span class="block text-[13px] text-[#78829d]">总文章数</span>
        <span class="text-[28px] font-normal mt-[5px]">{{ data.articleCount }}</span>
        <span class="text-[13px] text-[rgb(82,196,26)] mt-[13px] ml-[10px] font-extrabold">+30%</span>
      </div>
      <div class="w-[46px] h-[46px] text-[20px] leading-[42px] text-center rounded-[var(--dh-admin-border-radius-small)] bg-[linear-gradient(310deg,rgb(80,208,255),rgb(80,163,255))]">
        <Icon iconName="icon-history" :iconSize="iconSize.itemTotal" fill="#fff"></Icon>
      </div>
    </li>
    <li class="flex justify-between w-[23.6%] h-[90px] bg-white rounded-[var(--dh-admin-border-radius-big)] px-[18px] pt-[20px] list-none shadow-[var(--dh-admin-border-shadow-shallow)]">
      <div class="w-[70%]">
        <span class="block text-[13px] text-[#78829d]">总分类数</span>
        <span class="text-[28px] font-normal mt-[5px]">{{ data.categoryCount }}</span>
        <span class="text-[13px] text-[rgb(82,196,26)] mt-[13px] ml-[10px] font-extrabold">+30%</span>
      </div>
      <div class="w-[46px] h-[46px] text-[20px] leading-[42px] text-center rounded-[var(--dh-admin-border-radius-small)] bg-[linear-gradient(310deg,rgb(59,219,255),rgb(97,218,225))]">
        <Icon iconName="icon-layout" :iconSize="iconSize.itemTotal" fill="#fff"></Icon>
      </div>
    </li>
    <li class="flex justify-between w-[23.6%] h-[90px] bg-white rounded-[var(--dh-admin-border-radius-big)] px-[18px] pt-[20px] list-none shadow-[var(--dh-admin-border-shadow-shallow)]">
      <div class="w-[70%]">
        <span class="block text-[13px] text-[#78829d]">总标签数</span>
        <span class="text-[28px] font-normal mt-[5px]">{{ data.tagCount }}</span>
        <span class="text-[13px] text-[rgb(82,196,26)] mt-[13px] ml-[10px] font-extrabold">+30%</span>
      </div>
      <div class="w-[46px] h-[46px] text-[20px] leading-[42px] text-center rounded-[var(--dh-admin-border-radius-small)] bg-[linear-gradient(310deg,rgb(245,106,88),rgb(245,85,64))]">
        <Icon iconName="icon-training" :iconSize="iconSize.itemTotal" fill="#fff"></Icon>
      </div>
    </li>
    <li class="flex justify-between w-[23.6%] h-[90px] bg-white rounded-[var(--dh-admin-border-radius-big)] px-[18px] pt-[20px] list-none shadow-[var(--dh-admin-border-shadow-shallow)]">
      <div class="w-[70%]">
        <span class="block text-[13px] text-[#78829d]">总评论数</span>
        <span class="text-[28px] font-normal mt-[5px]">{{ data.commentCount }}</span>
        <span class="text-[13px] text-[rgb(82,196,26)] mt-[13px] ml-[10px] font-extrabold">+30%</span>
      </div>
      <div class="w-[46px] h-[46px] text-[20px] leading-[42px] text-center rounded-[var(--dh-admin-border-radius-small)] bg-[linear-gradient(310deg,rgb(163,139,228),rgb(130,94,228))]">
        <Icon iconName="icon-smile" :iconSize="iconSize.itemTotal" fill="#fff"></Icon>
      </div>
    </li>
  </ul>
</template>
<script setup lang="ts">
import { getOverview } from '@/api/user';
import { OverView } from '@/types/DashBoard';
const iconSize = reactive({
  itemTotal: '1.2'
})
const data = reactive<OverView>({
  articleCount: 0,
  categoryCount: 0,
  commentCount: 0,
  tagCount: 0
})

onMounted(async () => {
  const overview = await getOverview()
  Object.assign(data, overview)
})
</script>
