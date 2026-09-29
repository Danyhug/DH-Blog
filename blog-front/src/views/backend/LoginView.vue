<template>
  <div class="flex h-screen">
    <div class="relative h-full w-[520px] p-[20px] [@media(max-width:960px)]:hidden">
      <div class="absolute left-[-58%] top-0 h-[200px] w-[400px] bg-[rgba(0,0,0,0.8)] [transform:rotate(-38deg)_translateX(50%)]">
        <img src="@/assets/images/logo-nofill.png" alt="" class="h-[130px] [transform:rotate(38deg)_translateX(56%)_translateY(24%)]">
        DH-Blog
      </div>
      <img src="@/assets/images/admin-login-bg.png" alt="" class="absolute left-0 top-0 w-full h-full z-[-1]">
      <img src="@/assets/images/admin-login-img.svg" alt="" class="relative block w-[600px] mt-[180px] ml-[-40px] z-[4]">
    </div>
    <div class="flex h-full w-[calc(100%-520px)] items-center justify-center [@media(max-width:960px)]:w-full">
      <div class="w-[440px] px-[5px] mb-[20px]">
        <div>
          <h3 class="login-title ml-[-2px] text-[34px] text-[#252F4A]">欢迎回来</h3>
          <p class="mt-[10px] text-[16px] text-[#8c8c8c]">输入您的账号密码登录</p>

          <div class="mt-[25px] rounded-[8px]">
            <el-input v-model="user.username" placeholder="请输入账号" size="large" />
          </div>
          <div class="mt-[25px] rounded-[8px]">
            <el-input v-model="user.password" placeholder="请输入密码" size="large" type="password" />
          </div>

          <!-- TODO 验证滑块 -->
          <div class="mt-[18px] mb-[30px]">
            <el-checkbox v-model="user.remember" name="type">记住密码</el-checkbox>
            <a href="忘记密码？"></a>
          </div>

          <div>
            <el-button type="primary" size="large" class="login-btn-main w-full" @click="login">登录</el-button>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped lang="less">
/* 字重：style.less 全局 h1,h2,h3{font-weight:400} 是无层级规则，压过工具类 */
.login-title {
  font-weight: 600;
}

/* .el-button 自带 display/background-color，--large 又设了 height（无层级），压过工具类 */
.login-btn-main {
  display: block;
  height: 44px;
  background-color: #2a8aff;
}
</style>

<script setup lang="ts">
import { UserLogin } from "@/types/User";
import { onMounted, ref } from "vue";
import { userLogin } from "@/api/user";
import router from "@/router";
import { notify } from "@/utils/notification";

const user = ref<UserLogin>({
  username: "admin",
  password: "admin",
  valid: true,
  remember: false,
});

const login = () => {
  if (user.value.valid) {
    if (user.value.username.length == 0 || user.value.password.length == 0) {
      notify.error("账号或密码不能为空");
    } else {
      userLogin(user.value).then(token => {
        localStorage.setItem('token', token);

        notify.success('登录成功')
        const redirect = router.currentRoute.value.query.redirect
router.replace(redirect ? redirect.toString() : { name: "Admin" })
      })
    }
  }
}

onMounted(() => {
  if (localStorage.getItem('token')) {
    notify.success('登录状态校验成功')
    router.replace({ name: "Admin" })
  }
})
</script>
