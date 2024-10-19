<template>
  <div id="banner" ref="banner">
    <div class="info">
      <slot></slot>
    </div>
  </div>
</template>

<script setup>
import { onMounted, ref, watch } from 'vue'
import { useUserStore } from '@/store/index'

const store = useUserStore();
const banner = ref(null)

function articleAnimate() {
  if (store.homeShowComponent == 'articleInfoSide') {
    banner.value.classList.add('fade-in-article')
  } else if (store.homeShowComponent == 'home') {
    banner.value.classList.remove('fade-in-article')
  }
}

onMounted(() => {
  articleAnimate()
  watch(() => store.homeShowComponent, _ => articleAnimate())
})

</script>

<style lang="less" scoped>
// 进入文章时，上面的黑色背景淡入
.fade-in-article {
  &::after {
    content: "";
    position: absolute;
    top: 0;
    left: 0;
    width: 100%;
    height: 100%;
    background-color: rgb(0, 0, 0);
    opacity: 0;
    animation: fadeIn 1s ease-in-out forwards;
    animation-delay: 1s;
  }

  @keyframes fadeIn {
    from {
      opacity: 0;
    }

    to {
      opacity: .3;
    }
  }
}

#banner {
  position: relative;
  z-index: -999;
  width: 100%;
  height: 80vh;
  background: url('@/assets/images/banner.png') no-repeat;
  background-size: 100% 100%;
  filter: contrast(88%);
  display: flex;
  justify-content: center;
  align-items: center;

  .info {
    z-index: 2;
    text-align: center;
    line-height: 1.2;
    color: #fff;
    font-family: 'Fredericka the Great', Mulish, -apple-system, "PingFang SC", "Microsoft YaHei", sans-serif;
  }
}
</style>