<template>
  <div>
    <nav id="nav">
      <ul class="menu">
        <li class="item">
          <router-link to="/">
            <Icon iconName="icon-chengbao"></Icon>
            首页
          </router-link>
        </li>
        <li class="item">
          <router-link to="/about">
            <Icon iconName="icon-chaoren"></Icon>
            关于
          </router-link>
        </li>
        <li class="item">
          <router-link to="/tags">
            <Icon iconName="icon-fenlei"></Icon>
            分类
          </router-link>
        </li>
        <li class="item">
          <router-link to="/webdav">
            <Icon iconName="icon-wenjian"></Icon>
            我的网盘
          </router-link>
        </li>
        <li class="item">
          <router-link to="/webdav-demo">
            <Icon iconName="icon-cangku"></Icon>
            WebDAV演示
          </router-link>
        </li>
        <li class="item">
          <router-link to="/login">
            <Icon iconName="icon-fenlei"></Icon>
            后台
          </router-link>
        </li>
      </ul>
    </nav>
  </div>
</template>

<script>
export default {
  name: "Header"
}
</script>

<style lang="less" scoped>
#nav {
  padding: .625rem 100px;
  position: absolute;
  top: 0;
  height: 50px;
  color: var(--grey-0);
  text-shadow: 0 .2rem .3rem rgba(0, 0, 0, .5);
  line-height: 50px;
}

#nav .inner {
  display: flex;
  flex-wrap: nowrap;
}

.menu {
  width: 100%;

  .item {
    padding: 0 20px;
  }
}

.menu,
.right {
  display: flex;

  li {
    margin-left: 1.875rem;
  }
}

ul {
  list-style: none;
}
</style>