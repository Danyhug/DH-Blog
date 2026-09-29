<template>
  <div ref="container" class="container h-full flex flex-col relative bg-white">
    <h1 @click="router.push({ name: 'Home' })">DH-Blog</h1>
    <el-menu router :default-active="$route.path">
      <el-menu-item index="/admin/dashboard">
        <el-icon :size="iconSize">
          <Odometer />
        </el-icon>
        数据总览
      </el-menu-item>
      <el-menu-item index="/admin/publish">
        <el-icon :size="iconSize">
          <Edit />
        </el-icon>
        博客发布
      </el-menu-item>
      <el-menu-item index="/admin/manager">
        <el-icon :size="iconSize">
          <EditPen />
        </el-icon>
        博客管理
      </el-menu-item>
      <el-menu-item index="/admin/gateway">
        <el-icon :size="iconSize">
          <Connection />
        </el-icon>
        AI 网关
      </el-menu-item>
      <el-menu-item index="/admin/system">
        <el-icon :size="iconSize">
          <Setting />
        </el-icon>
        系统配置
      </el-menu-item>
      <el-menu-item index="/admin/comment">
        <el-icon :size="iconSize">
          <User />
        </el-icon>
        评论管理
      </el-menu-item>
      <el-menu-item index="/admin/events">
        <el-icon :size="iconSize">
          <Monitor />
        </el-icon>
        运行日志
      </el-menu-item>
    </el-menu>

    <div class="tool h-[60px] absolute bottom-[20px] left-[12px]">
      <div>
        <el-icon
          size="26"
          color="#666"
          :style="{
            transform: isFold ? 'rotate(-180deg)' : 'rotate(0deg)',
            transition: 'all 0.3s',
          }"
        >
          <Fold @click="fold()" class="cursor-pointer" />
        </el-icon>
      </div>
    </div>
  </div>
</template>
<style scoped lang="less">
/* 字重：style.less 全局 h1,h2,h3{font-weight:400} 是无层级规则，压过工具类 */
h1 {
  opacity: 1;
  font-weight: bold;
  font-size: 30px;
  font-style: italic;
  line-height: 80px;
  color: #3f8cff;
  text-shadow: 1px 1px 2px rgba(0, 0, 0, 0.3);
  text-align: center;
  cursor: pointer;
  transition: all 0.6s;
}

.container :deep(.el-menu) {
  padding: 0 12px;
  background-color: #fff;
  flex: 1;
  border: none;
}

.container .el-icon {
  margin-left: 10px;
  margin-right: 15px;
}

.container .tool .el-icon {
  margin: 0;
}

.container .el-menu-item {
  --el-menu-hover-bg-color: rgb(245, 245, 245);
  border-radius: 10px;
  margin-bottom: 10px;
  transition: all 0.3s;
}

.container .is-active {
  --el-menu-active-color: #3f8cff;
  background-color: var(--el-menu-bg-color);
  box-shadow: -2px 2px 26px #0000001b;
}

.fold-container {
  h1 {
    font-size: 0;
    opacity: 0;
  }

  /* .el-menu 自带 background-color/border（无层级），压过工具类 */
  :deep(.el-menu) {
    padding: 0 5px;
  }

  .el-icon {
    margin: 0;
  }

  /* 库内部类与运行期状态类，工具类够不着；.fold-container 同理 */
  .el-menu-item {
    font-size: 0;
    --el-menu-base-level-padding: 12px;
    --el-menu-item-height: 46px;
  }

  .is-active {
    background-color: var(--el-menu-bg-color);
    box-shadow: 0 0 16px #eee;
  }
}
</style>
<script setup>
import { ref } from "vue";
import { useRouter } from "vue-router";
import { debounce } from "@/utils/tool";
const router = useRouter();
const iconSize = ref(20);
const container = ref(null);
// 折叠元素的宽度
const foldFather = ref(-1);
const isFold = ref(false);
let previousWidthState = window.innerWidth >= 1200; // 初始状态

const fold = () => {
  container.value.classList.toggle("fold-container");
  isFold.value = !isFold.value;

  if (foldFather.value == -1 || isFold.value) {
    foldFather.value = document.querySelector(".el-aside").offsetWidth;
    // 折叠
    document.querySelector(".el-aside").style.width = "60px";
  } else {
    // 展开
    document.querySelector(".el-aside").style.width = foldFather.value + "px";
  }
};
function resize() {
  const currentWidthState = window.innerWidth >= 1200;

  if (currentWidthState !== previousWidthState) {
    previousWidthState = currentWidthState; // 更新状态

    if (window.innerWidth >= 1200 && !isFold.value) {
      return;
    } else if (window.innerWidth < 1200 && isFold.value) {
      return;
    }

    fold();
  }
}

const debouncedResize = debounce(resize, 100);
onMounted(() => {
  // 初始调用一次以确保初始状态正确
  resize();

  // 使用 addEventListener 添加事件处理程序
  window.addEventListener("resize", debouncedResize);
});

onBeforeUnmount(() => {
  // 移除事件监听器
  window.removeEventListener("resize", debouncedResize);
});
</script>
