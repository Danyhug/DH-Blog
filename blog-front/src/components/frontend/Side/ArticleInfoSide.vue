<template>
  <div class="w-full">
    <el-row>
      <el-col>
        <el-card class="box w-full min-h-screen">
          <img :src="store.homeHeaderInfo.thumbnailUrl" class="image w-[95%] mx-auto" />

          <div>
            <p class="title text-[1.15rem] my-3">{{ store.homeHeaderInfo.title }}</p>
            <div class="schedule">
              <el-progress :color="customColors" :percentage="sideInfo.process"></el-progress>
              <p class="text-[14px] text-[#606266]">已阅读时长：{{ formatSeconds(second) }}</p>
            </div>
          </div>

          <div class="links grid grid-cols-[repeat(5,1fr)] justify-items-center w-full text-[#909399] text-[12px] mt-[14px] pt-2">
            <a class="cursor-pointer">
              <Icon iconName="icon-31erweima" iconSize="2"></Icon>
            </a>
            <a class="cursor-pointer">
              <Icon iconName="icon-fangda" class="mt-[3px]" iconSize="1.56"
                @click="store.aritcleModel.isFullPreview = !store.aritcleModel.isFullPreview"></Icon>
            </a>
            <a class="cursor-pointer">
              <Icon iconName="icon-forward" iconSize="2"></Icon>
            </a>
            <a class="cursor-pointer">
              <Icon iconName="icon-share" iconSize="2"></Icon>
            </a>
            <a class="cursor-pointer">
              <Icon iconName="icon-setting" iconSize="2"></Icon>
            </a>
          </div>

          <div class="tags w-full text-[#909399] text-[14px]">
            <el-divider>
              <Icon iconName="icon-shili" iconSize="1.56"></Icon>
            </el-divider>
            <div class="tag-list flex flex-wrap gap-2 w-full">
              <span class="tag inline-flex items-center max-w-full text-white rounded-[5px] py-[3px] px-[6px] leading-[1.45] [overflow-wrap:anywhere] [word-break:break-word] whitespace-normal" v-for="(item, index) in store.homeHeaderInfo.tags" :key="item.id ?? item.name"
                :style="{ backgroundColor: tags[index % tags.length] }">{{ item.name }}</span>
            </div>
          </div>

          <div class="catelog flex-1 overflow-y-auto text-left">
            <MdCatalog editorId="dh-editor" :scrollElement="scrollElement" theme="light" />
          </div>
        </el-card>
      </el-col>
    </el-row>
  </div>
</template>

<script setup>
import { useUserStore } from '@/store';
import { reactive, onMounted, onBeforeUnmount, ref } from 'vue'
import { debounce } from '@/utils/tool'
const store = useUserStore()

const scrollElement = document.documentElement

const getRandomColor = () => {
  const tagColors = [
    "#037ef3", "#f85a40", "#00c16e", "#7552cc", "#0cb9c1", "#f48924", "#ff4f81"
  ]
  return tagColors[Math.floor(Math.random() * tagColors.length)];
}
const tags = Array.from({ length: 9 }, () => getRandomColor())

const customColors = [
  { color: 'rgb(57,157,254)', percentage: 80 },
  { color: 'rgb(0,207,102)', percentage: 100 },
]

const sideInfo = reactive({
  process: 0
})

// 计算分钟和秒
function formatSeconds(seconds) {
  if (seconds >= 60) {
    var minutes = Math.floor(seconds / 60);
    var remainingSeconds = seconds % 60;
    return minutes + '分' + remainingSeconds + '秒';
  } else {
    return seconds + '秒';
  }
}

let timer = null
let second = ref(0)
const scrollListener = debounce(() => {
  let scrollTop = window.scrollY

  // 获取div的高度
  let height = document.querySelector('.blog-container').scrollHeight

  // 计算阅读百分比
  let process = Math.floor((scrollTop / height) * 100)
  sideInfo.process = process > 100 ? 100 : process
}, 16)

onMounted(() => {
  window.addEventListener('scroll', scrollListener)
  scrollTo(0, 60)

  timer = setInterval(() => second.value++, 1000)
})

onBeforeUnmount(() => {
  clearInterval(timer)
  window.removeEventListener('scroll', scrollListener)
})

</script>

<style lang="less" scoped>
/* .el-card 自带 background-color（无层级），压过 bg-white */
.box {
  background-color: #fff;
}

/* :deep()：EP 卡片内容区加不上 class */
:deep(.el-card__body) {
  height: 100vh;
  display: flex;
  flex-direction: column;
}

/* 裸伪元素选择器，无工具类写法 */
::-webkit-scrollbar {
  width: 0;
}
</style>
