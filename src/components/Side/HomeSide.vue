<template>
  <div class="info">
    <img src="@/assets/images/logo.jpg" alt="">
    <div class="info-text">
      <p class="title">Danyhug</p>
      <p class="sub-title">我们原神玩家是这样的</p>
      <ul class="links">
        <li>
          <a href="">
            <p class="num">46</p>
            文章
          </a>
        </li>
        <span></span>
        <li>
          <a href="">
            <p class="num">17</p>
            分类
          </a>
        </li>
        <span></span>
        <li>
          <a href="">
            <p class="num">53</p>
            标签
          </a>
        </li>
      </ul>

      <ul class="external-links">
        <li>
          <a href="https://github.com/Danyhug" target="_blank">
            <Icon iconName="icon-github1" iconSize="2.3"></Icon>
          </a>
        </li>
        <li>
          <a href="https://space.bilibili.com/38630127" target="_blank">
            <Icon iconName="icon-bilibili" iconSize="2.2" style="fill: rgb(250,116,153)"></Icon>
          </a>
        </li>
      </ul>
    </div>
  </div>

</template>
<script>

export default {
  name: 'HomeSide',
  data() {
    return {

    }
  }
}
</script>
<style scoped>
.info {
  padding-top: 24px;
}

.links {
  margin-top: 1.5rem;
  margin-bottom: 1.25rem;
  display: flex;
  justify-content: space-between;

  li {
    color: var(--grey-7);

    .num {
      font-weight: bold;
      font-size: 22px;
    }
  }

  span {
    display: block;
    border-left: 1px solid var(--grey-4);
  }
}

img {
  width: 9.375rem;
  height: 9.375rem;
  border-radius: 50%;
}

.info-text {
  line-height: 2.2em;
}

.external-links {
  text-align: center;

  li {
    display: inline-block;
    padding: 0 .9375rem;

  }
}
</style>