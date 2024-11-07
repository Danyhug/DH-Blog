<template>
  <div class="dash-container">
    <TotalItem />

    <!-- 中间图表，左侧显示 访客数、用户数、总浏览数、总评论数，右侧显示具体图表数据 -->
    <div class="chart-item">
      <div class="chart-left">
        <div class="item-top">
          <div>
            <div class="item-title">访问记录</div>
            <div class="item-sub">
              今日增长
              <span> 128 </span>
              /
              <span class="sub"> -37% </span>
            </div>
          </div>

          <el-radio-group v-model="visitRadio">
            <el-radio-button label="今日" value="day" />
            <el-radio-button label="此周" value="week" />
            <el-radio-button label="本月" value="month" />
            <el-radio-button label="累计" value="total" />
          </el-radio-group>
        </div>

        <div class="item-table">
          <el-table :data="ipData" max-height="315px">
            <el-table-column type="index" label="序号" min-width="9%"></el-table-column>
            <el-table-column prop="city" label="城市" min-width="20%"></el-table-column>
            <el-table-column prop="ip" label="IP" min-width="22%"></el-table-column>
            <el-table-column prop="count" label="访问数" min-width="14%"></el-table-column>
            <el-table-column prop="ban" label="Ban" min-width="10%"></el-table-column>
          </el-table>
        </div>
      </div>

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

  .chart-left {
    width: 44%;
    --el-border-radius-base: 6px;

    .item-table {
      margin-top: 15px;
      padding: 0 8px;
    }
  }

  .chart-right {
    width: 54%;

    .chart {
      margin-top: 10px;
      height: 400px;
    }

    .online-count {
      font-size: 13px;
      color: rgb(120, 130, 157);
    }
  }
}
</style>
<script setup>
import TotalItem from "@/components/backend/DashBoard/TotalItem.vue";
const visitRadio = ref('day')

import VChart, { THEME_KEY } from 'vue-echarts';

import { use } from 'echarts/core'
import { LineChart } from 'echarts/charts'
import { GridComponent, TooltipComponent } from 'echarts/components'
import { CanvasRenderer } from 'echarts/renderers'
import { useAdminStore } from "@/store";
provide(THEME_KEY, 'light');

use([GridComponent, TooltipComponent, LineChart, CanvasRenderer])

const ipData = [
  { city: "河北张家口", ip: "127.0.0.1", count: 8848, ban: 0 },
  { city: "北京昌平", ip: "192.168.1.1", count: 5672, ban: 1 },
  { city: "上海浦东区", ip: "10.0.0.1", count: 3456, ban: 0 },
  { city: "广东东莞", ip: "172.16.0.1", count: 7890, ban: 1 },
  { city: "江苏镇江", ip: "192.168.0.2", count: 4321, ban: 0 },
  { city: "浙江杭州", ip: "10.0.0.2", count: 6543, ban: 1 },
  { city: "山东曹县", ip: "172.16.0.2", count: 9876, ban: 0 },
  { city: "河南洛阳", ip: "192.168.1.2", count: 2345, ban: 1 },
  { city: "湖北武汉", ip: "10.0.0.3", count: 5678, ban: 0 },
  { city: "湖南长沙", ip: "172.16.0.3", count: 1234, ban: 1 },
];

const option = ref({
  xAxis: {
    type: 'category',
    data: [
      '1月',
      '2月',
      '3月',
      '4月',
      '5月',
      '6月',
      '7月',
      '8月',
      '9月',
      '10月',
      '11月',
      '12月'
    ],
    axisLabel: {
      // fontWeight: 'bold' // 加粗 X 轴标签
    }
  },
  yAxis: {
    type: 'value',
    axisLabel: {
      fontWeight: 'bold' // 加粗 Y 轴标签
    }
  },
  tooltip: {
    trigger: 'axis',
    axisPointer: {
      type: 'line'
    },
    formatter: function (params) {
      const date = params[0].axisValue;
      const value = params[0].data;
      return `${date}<br/><div style="font-size: 13px"><span style="display: inline-block; width: 12px; height: 12px; background: #348fff; border-radius: 50%; vertical-align: middle; margin-right: 6px;"></span>浏览 <span style="font-weight: 800">${value}</span></div>`;
    }
  },
  series: [
    {
      data: [120, 200, 150, 80, 70, 110, 130, 150, 170, 190, 210, 230],
      type: 'line',
      smooth: true,
      color: '#2788ff',
      symbol: 'none', // 去掉点
      lineStyle: {
        width: 2.2 // 增加线的宽度
      }
    }
  ],
  grid: {
    left: '2.2%',
    right: '3%',
    top: '5%',
    containLabel: true
  }
})

const store = useAdminStore()
</script>
