<template>
    <div id="banner">
      <div class="info">
        <slot></slot>
      </div>
    </div>
  </template>
  
  <script>
  export default {
    name: 'Banner'
  }
  </script>
  
  <style lang="less" scoped>
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
      text-align: center;
      line-height: 1.2;
      color: #fff;
      font-family: 'Fredericka the Great', Mulish, -apple-system, "PingFang SC", "Microsoft YaHei", sans-serif;
    }
  }
  </style>