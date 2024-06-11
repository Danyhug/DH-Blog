<template>
  <div>
    <el-row>
      <el-col>
        <el-card class="box">
          <img :src="store.homeHeaderInfo.thumbnailUrl" class="image" />

          <div>
            <p class="title">{{ store.homeHeaderInfo.title }}</p>
            <div class="schedule">
              <el-progress :color="customColors" :percentage="sideInfo.process"></el-progress>
              <p>已阅读时长：1分24秒</p>
            </div>
          </div>

          <div class="links">
            <a>
              <Icon iconName="icon-31erweima" iconSize="2"></Icon>
            </a>
            <a>
              <Icon iconName="icon-home" iconSize="2"></Icon>
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
              <Icon iconName="icon-shili" iconSize="1.6"></Icon>
            </el-divider>
            <a href="" class="tag" v-for="(item, index) in 9" :style="{ backgroundColor: tags[index] }">原神</a>
          </div>
        </el-card>
      </el-col>
    </el-row>
  </div>
</template>

<script setup>
const store = useUserStore()

import { useUserStore } from '@/store';
import { reactive, onMounted } from 'vue'
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

function debounce(func, wait) {
  let timeout;
  return function () {
    const context = this;
    const args = arguments;
    clearTimeout(timeout);
    timeout = setTimeout(function () {
      func.apply(context, args);
    }, wait);
  };
}

window.addEventListener('scroll', debounce(() => {
  let scrollTop = window.scrollY

  // 获取div的高度
  let height = document.querySelector('.blog-container').scrollHeight

  // 计算阅读百分比
  let process = Math.floor((scrollTop / height) * 100)
  sideInfo.process = process > 100 ? 100 : process
}, 16))

onMounted(() => {
  scrollTo(0, 60)
})

</script>

<style lang="less" scoped>
.box {
  padding: 0 20px;
  height: 100vh;
  background-color: #fff;

  img {
    width: 300px;
  }
}

.schedule {
  p {
    font-size: 14px;
    color: #606266;
  }
}

.title {
  font-size: 1.25rem;
  margin: 0.75rem 0;
}

.box {
  width: 100%;
}

.links {
  margin-top: 30px;
  padding: 8px 0;
  display: grid;
  grid-template-columns: repeat(5, 1fr);
  justify-items: center;
  width: 100%;
  color: #909399;
  font-size: 14px;
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
