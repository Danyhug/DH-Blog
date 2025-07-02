<template>
  <div class="dash-container">
    <TotalItem />

    <!-- 中间图表，左侧显示 访客数、用户数、总浏览数、总评论数，右侧显示具体图表数据 -->
    <div class="chart-item">
      <VisitTable />
      
      <VisitChart />
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

  :deep(>div) {
    height: 100%;
    border-radius: var(--dh-admin-border-radius-big);
    padding: 20px 0 30px;
    box-shadow: var(--dh-admin-border-shadow-shallow);
    background-color: #fff;

    .item-top {
      padding: 0 18px;
      width: 100%;
      display: flex;
      justify-content: space-between;
      align-items: center;

      .item-title {
        font-size: 18px;
      }

      .item-sub {
        font-size: 13px;
        margin-top: 3px;
        color: rgb(120, 130, 157);

        span {
          color: #7a82ff;
        }

        span.add {
          color: var(--dh-admin-color-success);
        }

        span.sub {
          color: var(--dh-admin-color-error);
        }
      }
    }
  }
}
</style>
<script setup>
import TotalItem from "@/components/backend/DashBoard/TotalItem.vue";
import VisitTable from "@/components/backend/DashBoard/VisitTable.vue";
import VisitChart from "@/components/backend/DashBoard/VisitChart.vue";
</script>
