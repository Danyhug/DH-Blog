<template>
  <div class="dash-container">
    <TotalItem />

    <!-- 中间图表，左侧显示 访客数、用户数、总浏览数、总评论数，右侧显示具体图表数据 -->
    <div class="chart-item">
      <div class="chart-left">
        <div class="item">
          <div class="item-title">访问量</div>
        </div>
      </div>

      <div class="chart-right">
        <div class="item">
          <div class="item-top">
            <div class="item-title">访问量</div>
            <div class="item-sub">
              本月增长
              <span>+15%</span>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>
<style lang="less" scoped>
.dash-container {
  width: 100%;
}



.chart-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 16px;
  margin-bottom: 10px;
  height: 450px;

  >div {
    height: 100%;
    border-radius: var(--dh-admin-border-radius-big);
    padding-top: 20px;
    padding-bottom: 30px;
    box-shadow: var(--dh-admin-border-shadow-shallow);
  }

  .chart-left {
    width: 43%;
    display: grid;
    grid-template-columns: repeat(2, 1fr);
    grid-gap: 20px;
    align-items: center;

    .item {
      width: 55%;
      background-color: green;
    }

    background: #fff;
  }

  .chart-right {
    width: 54%;
    background-color: #fff;

    .item-top {
      padding: 0 18px;

      .item-title {
        font-size: 18px;
      }

      .item-sub {
        font-size: 13px;
        margin-top: 3px;
        color: rgb(120, 130, 157);

        span {
          color: rgb(82, 196, 26);
        }
      }
    }
  }
}
</style>
<script setup>
import TotalItem from '@/components/backend/DashBoard/TotalItem.vue'
</script>