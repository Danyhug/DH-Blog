<template>
  <div class="lock-view-container">
    <Header></Header>

    <div class="lock-view">
      <div class="left">
        <img src="@/assets/images/lock-robot.png" alt="锁定文章">
      </div>
      <div class="right">
        <div class="logo">
          <div class="logo-text">
            <span>D</span>any<span>h</span>ug's <span>Blog</span>
          </div>
        </div>

        <div class="title">已被设为私密文章 / 输入密钥解锁</div>
        <div class="form">
          <input type="text" autofocus v-model="data.password" @keyup.enter="check" />
          <el-icon size="2em" style="vertical-align: text-bottom; position: relative; left: -50px; cursor: pointer;"
            @click="check">
            <Unlock />
          </el-icon>
        </div>

        <button @click="goBack">
          <span class="text">返回</span>
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
</script>

<style scoped lang="less">
// 适配手机
@media (max-width: 1024px) {
  .left {
    display: none;
  }

  .lock-view .right .logo .logo-text {
    font-size: 2.4em!important;
  }

  .lock-view {
    .right {
      margin-right: 0 !important;
    font-size: 12px!important;

      justify-content: center;
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

.lock-view-container {
  padding-top: 10%;
  height: 100vh;
  width: 100%;
  background-color: #18191a;
}

.lock-view {
  display: flex;
  justify-content: center;
  color: #f5f5f5;

  .left {
    padding-right: 5%;

    img {
      width: 360px;
      height: 471px;
    }
  }

  .right {
    position: relative;
    margin-right: 7%;
    height: 100%;
    display: flex;
    flex-direction: column;
    justify-content: flex-start;
    color: rgb(245, 245, 245);

    &>div {
      margin: 30px;
    }

    .logo {
      font-size: 18px;

      .logo-text {
        font-weight: bold;
        font-size: calc(2.5em + 1.45vw);
        text-shadow: 1px 2px 1px #444742;

        span {
          color: #2AA2F7;
        }
      }

      img {
        width: 160px;
      }
    }

    .title {
      font-size: 1.6em;
    }

    .form {
      input {
        background-color: #2b2b2f;
        color: #f5f5f5;
        border: none;
        padding: 10px;
        flex-grow: 1;
        z-index: 2;
        margin-right: -10px;
        padding-top: 15px;
        padding-bottom: 11px;
        font-size: 1.8em;
        padding-right: 80px;
        text-align: center;
        width: 460px;
        // 字符之间隔2px
        letter-spacing: 1.5em;

        &:focus {
          border-color: #66afe9;
          outline: 0;
          -webkit-box-shadow: inset 0 1px 1px rgba(0, 0, 0, 0.075), 0 0 8px rgba(102, 175, 233, 0.6);
          box-shadow: inset 0 1px 1px rgba(0, 0, 0, 0.075), 0 0 8px rgba(102, 175, 233, 0.6);
        }
      }
    }

    button {
      position: absolute;
      bottom: -20%;
      right: 14%;

      align-self: flex-end;
      background-image: linear-gradient(144deg, #af40ff, #5b42f3 50%, #00ddeb);
      border: 0;
      border-radius: 8px;
      box-shadow: rgba(151, 65, 252, 0.2) 0 15px 30px -5px;
      box-sizing: border-box;
      color: #f5f5f5;
      display: flex;
      font-size: 18px;
      justify-content: center;
      line-height: 1em;
      width: 23%;
      padding: 3px;
      text-decoration: none;
      user-select: none;
      -webkit-user-select: none;
      touch-action: manipulation;
      white-space: nowrap;
      cursor: pointer;
      transition: all 0.3s;
    }

    button:active,
    button:hover {
      outline: 0;
    }

    button span {
      background-color: rgb(5, 6, 45);
      padding: 16px 24px;
      border-radius: 6px;
      width: 100%;
      height: 100%;
      transition: 300ms;
    }

    button:hover span {
      background: none;
    }

    button:active {
      transform: scale(0.9);
    }

  }
}
</style>