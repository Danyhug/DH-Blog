<template>
  <div class="pt-[10%] h-screen w-full bg-[#18191a]">
    <Header></Header>

    <div class="lock-view flex justify-center text-[#f5f5f5]">
      <div class="left pr-[5%]">
        <img src="@/assets/images/lock-robot.png" alt="锁定文章" class="w-[360px] h-[471px]">
      </div>
      <div class="right relative mr-[7%] h-full flex flex-col justify-start text-[rgb(245,245,245)]">
        <div class="logo m-[30px] text-[18px]">
          <div class="logo-text font-bold text-[calc(2.5em+1.45vw)] [text-shadow:1px_2px_1px_#444742]">
            <span class="text-[#2AA2F7]">D</span>any<span class="text-[#2AA2F7]">h</span>ug's <span class="text-[#2AA2F7]">Blog</span>
          </div>
        </div>

        <div class="title m-[30px] text-[1.6em]">已被设为私密文章 / 输入密钥解锁</div>
        <div class="form m-[30px]">
          <input type="text" ref="input" autofocus v-model="data.password" @keyup.enter="check"
            class="bg-[#2b2b2f] text-[#f5f5f5] [border:none] p-[10px] grow z-[2] -mr-[10px] pt-[15px] pb-[11px] text-[1.8em] pr-[80px] text-center w-[460px] tracking-[1.5em] focus:border-[#66afe9] focus:[outline:0] focus:[box-shadow:inset_0_1px_1px_rgba(0,0,0,0.075),0_0_8px_rgba(102,175,233,0.6)]" />
          <el-icon size="2em" class="align-text-bottom relative -left-[50px] cursor-pointer"
            @click="check">
            <Unlock />
          </el-icon>
        </div>

        <button @click="goBack" ref="button" style="transform: translateX(-100%);"
          class="absolute bottom-[-20%] right-[14%] self-end bg-[linear-gradient(144deg,#af40ff,#5b42f3_50%,#00ddeb)] [border:0] rounded-lg [box-shadow:rgba(151,65,252,0.2)_0_15px_30px_-5px] box-border text-[#f5f5f5] flex text-[18px] justify-center leading-[1em] w-[23%] p-[3px] no-underline select-none touch-manipulation whitespace-nowrap cursor-pointer [transition:all_0.3s] hover:[outline:0] active:[outline:0] active:[transform:scale(0.9)]">
          <span class="text bg-[rgb(5,6,45)] py-4 px-6 rounded-md w-full h-full [transition:300ms]">返回</span>
        </button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { unLockArticle } from '@/api/user';
import router from '@/router';
import { useRoute } from 'vue-router';
import { reactive } from 'vue';

const input = ref<HTMLElement>()
const button = ref<HTMLElement>()
const route = useRoute()
const goBack = () => router.back()

const data = reactive({
  id: route.query.id as unknown as number,
  password: ''
})

const check = () => {
  unLockArticle(data.id, data.password).then(res => {
    // 携带数据返回文章页
    localStorage.setItem('unlockArticle', JSON.stringify(res))
    router.replace({ name: 'ArticleInfo', params: { id: data.id } })
  })
}

onMounted(() => {
  if (button.value && input.value) {
    button.value.style.left = `${input.value.offsetWidth + input.value.offsetLeft}px`;
  }
});
</script>

<style scoped lang="less">
/* 移动端覆盖块：无层级 + !important，正好维持原有的断点行为 */
@media (max-width: 1024px) {
  .left {
    display: none;
  }

  .lock-view .right .logo .logo-text {
    font-size: 2.4em !important;
  }

  .lock-view {
    .right {
      margin-right: 0 !important;
      font-size: 12px !important;

      align-items: center;

      &>div {
        margin-left: 5px !important;
        margin-right: 5px !important;
      }

      .form input {
        width: 94% !important;
      }
    }
  }
}

/* 祖先 :hover + 后代 span；改成 span:hover 不等价（按钮有 3px padding） */
.lock-view .right button:hover span {
  background: none;
}
</style>
