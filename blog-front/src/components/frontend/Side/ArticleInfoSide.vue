<template>
  <div style="width: 100%;">
    <el-row>
      <el-col>
        <el-card class="box">
          <img :src="store.homeHeaderInfo.thumbnailUrl" class="image" />

          <div>
            <p class="title">{{ store.homeHeaderInfo.title }}</p>
            <div class="schedule">
              <el-progress :color="customColors" :percentage="sideInfo.process"></el-progress>
              <p>已阅读时长：{{ formatSeconds(second) }}</p>
            </div>
          </div>

          <div class="links">
            <a>
              <Icon iconName="icon-31erweima" iconSize="2"></Icon>
            </a>
            <a>
              <Icon iconName="icon-fangda" style="margin-top: 3px;" iconSize="1.56"
                @click="store.aritcleModel.isFullPreview = !store.aritcleModel.isFullPreview"></Icon>
            </a>
            <a>
              <Icon iconName="icon-forward" iconSize="2"></Icon>
            </a>
            <a>
              <Icon iconName="icon-share" iconSize="2"></Icon>
            </a>
            <a>
              <Icon iconName="icon-setting" iconSize="2"></Icon>
            </a>
          </div>

          <div class="tags">
            <el-divider>
              <Icon iconName="icon-shili" iconSize="1.56"></Icon>
            </el-divider>
            <a href="" class="tag" v-for="(item, index) in store.homeHeaderInfo.tags"
              :style="{ backgroundColor: tags[index] }">{{ item.name }}</a>
          </div>

          <div class="catelog">
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
.box {
  background-color: #fff;

  img {
    width: 95%;
    margin: 0 auto;
  }
}

:deep(.el-card__body) {
  height: 100vh;
  display: flex;
  flex-direction: column;
}

::-webkit-scrollbar {
  width: 0;
}

.catelog {
  flex: 1;
  overflow-y: auto;
  text-align: left;
}

.schedule {
  p {
    font-size: 14px;
    color: #606266;
  }
}

.title {
  font-size: 1.15rem;
  margin: 0.75rem 0;
}

.box {
  width: 100%;
  min-height: 100vh;
}

.links {
  margin-top: 14px;
  padding-top: 8px;
  display: grid;
  grid-template-columns: repeat(5, 1fr);
  justify-items: center;
  width: 100%;
  color: #909399;
  font-size: 12px;

  a {
    cursor: pointer;
  }
}

.tags {
  width: 100%;
  display: flex;
  flex-wrap: wrap;
  justify-content: space-between;
  color: #909399;
  font-size: 14px;

  a {
    width: 21%;
    margin-bottom: 10px;
    margin-top: 0;
    color: #fff;
    border-radius: 5px;
    padding: 3px 6px;
  }
}
</style>
