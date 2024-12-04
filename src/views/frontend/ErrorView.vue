<template>
  <div class="error-container">
    <img src="@/assets/images/wuwei.png" alt="无畏契约">
    <h1>发现非法访问</h1>
    <h2>访问终止</h2>
    <p>您的访问受限，解除限制请联系管理员</p>
  </div>
</template>
<style lang="less">
.error-container {
  background-color: #CE3D3F;
  display: flex;
  flex-direction: column;
  justify-content: center;
  align-items: center;
  height: 100vh;
  color: #FFFEFE;
  // 字间距
  letter-spacing: 5px;
  font-family: "幼圆";
  padding-bottom: 14%;

  img {
    width: 280px;
  }

  h1 {
    margin: 25px 0;
    font-size: 3rem;
  }

  h2 {
    font-size: 2rem;
  }

  p {
    margin-top: 12px;
    font-size: 1.1rem;
  }
}
</style>