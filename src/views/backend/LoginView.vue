<template>
  <div class="login">
    <div class="left-wrap">
      <div class="logo">
        <img src="@/assets/images/logo-nofill.png" alt="" class="logo-img">
        DH-Blog
      </div>
      <img src="@/assets/images/admin-login-bg.png" alt="" class="left-bg">
      <img src="@/assets/images/admin-login-img.svg" alt="" class="left-img">
    </div>
    <div class="right-wrap">
      <div class="login-wrap">
        <div class="form">
          <h3 class="title">欢迎回来</h3>
          <p class="sub-title">输入您的账号密码登录</p>

          <div class="input-wrap">
            <el-input v-model="user.username" placeholder="请输入账号" size="large" />
          </div>
          <div class="input-wrap">
            <el-input v-model="user.password" placeholder="请输入密码" size="large" type="password" />
          </div>

          <!-- TODO 验证滑块 -->
          <div class="forget-password">
            <el-checkbox v-model="user.remember" name="type">记住密码</el-checkbox>
            <a href="忘记密码？"></a>
          </div>

          <div class="login-btn">
            <el-button type="primary" size="large" @click="login">登录</el-button>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<style lang="less">
.login {
  display: flex;
  height: 100vh;

  .left-wrap {
    position: relative;
    height: 100%;
    width: 520px;
    padding: 20px;

    .logo {
      width: 400px;
      height: 200px;
      position: absolute;
      left: -58%;
      top: 0;
      background-color: rgba(0, 0, 0, .8);
      transform: rotate(-38deg) translateX(50%);

      img {
        transform: rotate(38deg) translateX(56%) translateY(24%);
        height: 130px;
      }
    }

    .left-bg {
      position: absolute;
      top: 0;
      left: 0;
      width: 100%;
      height: 100%;
      z-index: -1;
    }

    .left-img {
      position: relative;
      margin: 300px auto auto;
      z-index: 4;
      display: block;
    }
  }

  .right-wrap {
    width: calc(100% - 520px);
    height: 100%;
    display: flex;
    justify-content: center;
    align-items: center;

    .login-wrap {
      width: 440px;
      padding: 0 5px;
      margin-bottom: 20px;

      .title {
        margin-left: -2px;
        font-size: 34px;
        font-weight: 600;
        color: #252F4A;
      }

      .sub-title {
        margin-top: 10px;
        font-size: 16px;
        color: #8c8c8c;
      }

      .input-wrap {
        margin-top: 25px;
        border-radius: 8px;
      }

      .forget-password {
        margin: 18px 0 30px;
      }

      .login-btn {
        button {
          display: block;
          width: 100%;
          height: 44px;
          background-color: #2a8aff;
        }
      }
    }
  }
}

// 平板适配
@media screen and (max-width: 960px) {
  .login {
    .left-wrap {
      display: none;
    }

    .right-wrap {
      width: 100%;
    }
  }
}
</style>

<script setup lang="ts">
import { UserLogin } from "@/types/User";
import { onMounted, ref } from "vue";
import { userLogin } from "@/api/user";
import router from "@/router";

const user = ref<UserLogin>({
  username: "admin",
  password: "admin",
  valid: true,
  remember: false,
});

const login = () => {
  if (user.value.valid) {
    if (user.value.username.length == 0 || user.value.password.length == 0) {
      ElMessage.error("账号或密码不能为空");
    } else {
      userLogin(user.value).then(token => {
        localStorage.setItem('token', token);

        ElMessage.success('登录成功')
        router.replace({ name: "Admin" })
      })
    }
  }
}

onMounted(() => {
  if (localStorage.getItem('token')) router.replace({ name: "Admin" })
})
</script>