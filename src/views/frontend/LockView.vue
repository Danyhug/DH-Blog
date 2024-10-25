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

        <div class="title">已被设为私密文章 / 输入秘钥解锁</div>
        <div class="form">
          <input type="text">
          <el-icon size="2em" style="vertical-align: text-bottom; position: relative; left: -30px;">
            <Search />
          </el-icon>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
</script>

<style scoped lang="less">
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
    img {
      width: 360px;
      height: 471px;
    }
  }

  .right {
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
        padding-right: 60px;
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
  }
}
</style>