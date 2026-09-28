<template>
  <div>
    <nav id="nav"
      class="py-[0.625rem] px-[100px] absolute top-0 h-[50px] text-grey-0 [text-shadow:0_0.2rem_0.3rem_rgba(0,0,0,0.5)] leading-[50px]">
      <ul class="menu w-full flex list-none">
        <li class="item px-5 ml-[1.875rem]">
          <router-link to="/">
            <Icon iconName="icon-chengbao"></Icon>
            内容中心
          </router-link>
        </li>
        <li class="item px-5 ml-[1.875rem]">
          <router-link to="/knowledge">
            <Icon iconName="icon-chaoren"></Icon>
            知识星图
          </router-link>
        </li>
        <li class="item px-5 ml-[1.875rem]">
          <router-link to="/webdav">
            <Icon iconName="icon-wenjian"></Icon>
            我的网盘
          </router-link>
        </li>
        <li class="item px-5 ml-[1.875rem]">
          <router-link to="/login">
            <Icon iconName="icon-fenlei"></Icon>
            后台管理
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
