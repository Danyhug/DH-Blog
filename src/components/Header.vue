<template>
    <div>
        <nav id="nav">
            <div class="inner">
                <ul class="menu">
                    <li class="item">
                        <a href="/">
                            <Icon iconName="icon-chengbao"></Icon>
                            首页
                        </a>
                    </li>
                    <li class="item">
                        <a href="about">
                            <Icon iconName="icon-chaoren"></Icon>
                            关于
                        </a>
                    </li>
                    <li class="item">
                        <a href="tags">
                            <Icon iconName="icon-fenlei"></Icon>
                            分类
                        </a>
                    </li>
                </ul>

                <ul class="right">
                    <li>
                        <!-- 夜间模式 -->
                        <Icon iconName="icon-richu"></Icon>
                    </li>
                    <li>
                        <!-- 搜索 -->
                        <Icon iconName="icon-sousuo"></Icon>
                    </li>
                </ul>
            </div>
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
    position: fixed;
    left: 0;
    width: 100%;
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
</style>