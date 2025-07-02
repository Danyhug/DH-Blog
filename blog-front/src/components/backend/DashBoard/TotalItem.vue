<template>
  <!-- 总览 -->
  <ul class="total-item">
    <li>
      <div class="item-left">
        <span class="title">总文章数</span>
        <span class="num">{{ data.articleCount }}</span>
        <span class="trend">+30%</span>
      </div>
      <div class="item-right">
        <Icon iconName="icon-history" :iconSize="iconSize.itemTotal" fill="#fff"></Icon>
      </div>
    </li>
    <li>
      <div class="item-left">
        <span class="title">总分类数</span>
        <span class="num">{{ data.categoryCount }}</span>
        <span class="trend">+30%</span>
      </div>
      <div class="item-right">
        <Icon iconName="icon-layout" :iconSize="iconSize.itemTotal" fill="#fff"></Icon>
      </div>
    </li>
    <li>
      <div class="item-left">
        <span class="title">总标签数</span>
        <span class="num">{{ data.tagCount }}</span>
        <span class="trend">+30%</span>
      </div>
      <div class="item-right">
        <Icon iconName="icon-training" :iconSize="iconSize.itemTotal" fill="#fff"></Icon>
      </div>
    </li>
    <li>
      <div class="item-left">
        <span class="title">总评论数</span>
        <span class="num">{{ data.commentCount }}</span>
        <span class="trend">+30%</span>
      </div>
      <div class="item-right">
        <Icon iconName="icon-smile" :iconSize="iconSize.itemTotal" fill="#fff"></Icon>
      </div>
    </li>
  </ul>
</template>
<style lang="less" scoped>
.total-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 16px;
  margin-bottom: 10px;

  li {
    width: 23.6%;
    height: 90px;
    background: #fff;
    border-radius: var(--dh-admin-border-radius-big);
    padding: 20px 18px 0;
    list-style: none;
    box-shadow: var(--dh-admin-border-shadow-shallow);
    display: flex;
    justify-content: space-between;

    .item-left {
      width: 70%;

      .title {
        font-size: 13px;
        color: #78829d;
        display: block;
      }

      .num {
        font-size: 28px;
        font-weight: 400;
        margin-top: 5px;
      }

      .trend {
        font-size: 13px;
        color: rgb(82, 196, 26);
        margin: 13px 0 0 10px;
        font-weight: 800;
      }
    }

    .item-right {
      width: 46px;
      height: 46px;
      font-size: 20px;
      line-height: 42px;
      border-radius: var(--dh-admin-border-radius-small);
      text-align: center;
    }

    &:nth-child(1) .item-right {
      background-image: linear-gradient(310deg, rgb(80, 208, 255), rgb(80, 163, 255));
    }

    &:nth-child(2) .item-right {
      background-image: linear-gradient(310deg, rgb(59, 219, 255), rgb(97, 218, 225));
    }

    &:nth-child(3) .item-right {
      background-image: linear-gradient(310deg, rgb(245, 106, 88), rgb(245, 85, 64));
    }

    &:nth-child(4) .item-right {
      background-image: linear-gradient(310deg, rgb(163, 139, 228), rgb(130, 94, 228));
    }
  }
}
</style>
<script setup lang="ts">
import { getOverview } from '@/api/user';
import { OverView } from '@/types/DashBoard';
const iconSize = reactive({
  itemTotal: '1.2'
})
const data = reactive<OverView>({
  articleCount: 0,
  categoryCount: 0,
  commentCount: 0,
  tagCount: 0
})

onMounted(async () => {
  const overview = await getOverview()
  Object.assign(data, overview)
})
</script>