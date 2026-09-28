<template>
  <div id="banner" ref="banner"
    class="relative z-[-999] w-full h-[80vh] bg-[url(@/assets/images/banner.png)] bg-no-repeat bg-cover bg-center filter-[contrast(88%)] flex justify-center items-center">
    <div class="z-[2] text-center leading-[1.2] text-white [font-family:'Fredericka_the_Great',Mulish,-apple-system,'PingFang_SC','Microsoft_YaHei',sans-serif]">
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
/*
  `.fade-in-article` 由 JS 在运行时 add/remove（见 articleAnimate()），模板里写不出来，
  因此这条规则与它的 ::after 必须留在 CSS 里。

  同时它不能被内联的第二个原因：`animation: fadeIn` 引用的是本块内的 `@keyframes fadeIn`。
  Vue 会把两者一起改名（fadeIn-<scope>），所以这份是「配得上」的；一旦把 animation 挪到
  工具类里，引用的就是未改名的 fadeIn，会像 FilePreview 那样静默失效（§4.1）。
*/
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
</style>
