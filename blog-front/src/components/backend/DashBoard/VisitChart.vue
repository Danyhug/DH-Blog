<template>
  <div class="chart-right">
    <div class="item-top">
      <div>
        <div class="item-title">访问量</div>
        <div class="item-sub">
          本月增长
          <span class="add">+15%</span> /
          <span> {{ store.online }}人在线</span>
        </div>
      </div>
    </div>
    <div class="div-item-charts">
      <v-chart class="chart" :option="option" autoresize />
    </div>
  </div>
</template>
<style lang="less">
.chart-right {
  width: 53%;

  .chart {
    margin-top: 10px;
    height: 400px;
  }

  .online-count {
    font-size: 13px;
    color: rgb(120, 130, 157);
  }
}
</style>
<script setup>
import VChart, { THEME_KEY } from "vue-echarts";

import { use } from "echarts/core";
import { LineChart } from "echarts/charts";
import { GridComponent, TooltipComponent } from "echarts/components";
import { CanvasRenderer } from "echarts/renderers";
import { useAdminStore } from "@/store";
provide(THEME_KEY, "light");
const store = useAdminStore();

use([GridComponent, TooltipComponent, LineChart, CanvasRenderer]);

const option = ref({
  xAxis: {
    type: "category",
    data: [
      "1月",
      "2月",
      "3月",
      "4月",
      "5月",
      "6月",
      "7月",
      "8月",
      "9月",
      "10月",
      "11月",
      "12月",
    ],
    axisLabel: {
      // fontWeight: 'bold' // 加粗 X 轴标签
    },
  },
  yAxis: {
    type: "value",
    axisLabel: {
      fontWeight: "bold", // 加粗 Y 轴标签
    },
  },
  tooltip: {
    trigger: "axis",
    axisPointer: {
      type: "line",
    },
    formatter: function (params) {
      const date = params[0].axisValue;
      const value = params[0].data;
      return `${date}<br/><div style="font-size: 13px"><span style="display: inline-block; width: 12px; height: 12px; background: #348fff; border-radius: 50%; vertical-align: middle; margin-right: 6px;"></span>浏览 <span style="font-weight: 800">${value}</span></div>`;
    },
  },
  series: [
    {
      data: [120, 200, 150, 80, 70, 110, 130, 150, 170, 190, 210, 230],
      type: "line",
      smooth: true,
      color: "#2788ff",
      symbol: "none", // 去掉点
      lineStyle: {
        width: 2.2, // 增加线的宽度
      },
    },
  ],
  grid: {
    left: "2.2%",
    right: "3%",
    top: "5%",
    containLabel: true,
  },
});
</script>
