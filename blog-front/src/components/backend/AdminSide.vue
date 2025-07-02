<template>
  <div ref="container" class="container">
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
    </el-menu>

    <div class="tool">
      <div>
        <el-icon
          size="26"
          color="#666"
          :style="{
            transform: isFold ? 'rotate(-180deg)' : 'rotate(0deg)',
            transition: 'all 0.3s',
          }"
        >
          <Fold @click="fold()" style="cursor: pointer;" />
        </el-icon>
      </div>
    </div>
  </div>
</template>
<style scoped lang="less">
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

.container {
  height: 100%;
  display: flex;
  flex-direction: column;
  position: relative;
  background-color: #fff;

  :deep(.el-menu) {
    padding: 0 12px;
    background-color: #fff;
    flex: 1;
    border: none;
  }

  .el-icon {
    margin-left: 10px;
    margin-right: 15px;
  }

  .tool {
    .el-icon {
      margin: 0;
    }
  }

  .el-menu-item {
    --el-menu-hover-bg-color: rgb(245, 245, 245);
    border-radius: 10px;
    margin-bottom: 10px;
    transition: all 0.3s;
  }

  .is-active {
    --el-menu-active-color: #3f8cff;
    background-color: var(--el-menu-bg-color);
    box-shadow: -2px 2px 26px #0000001b;
  }

  .tool {
    height: 60px;
    position: absolute;
    bottom: 20px;
    left: 12px;
  }
}

.fold-container {
  h1 {
    font-size: 0;
    opacity: 0;
  }

  :deep(.el-menu) {
    padding: 0 5px;
  }

  .el-icon {
    margin: 0;
  }

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
