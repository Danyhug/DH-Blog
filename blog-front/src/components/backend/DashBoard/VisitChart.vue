<template>
  <div class="w-[53%]">
    <!-- item-top / item-title / item-sub 三个类名是父组件 DashBoardView 用 :deep(>div) 兜底的锚点，不能删 -->
    <div class="item-top mb-[3px]">
      <div class="info-section">
        <div class="item-title text-[#303133]">访问量统计</div>
        <div class="item-sub">
          {{ chartMode === 'daily' ? '今日' : '本月' }}增长
          <span class="add font-medium">{{ growth > 0 ? '+' + growth.toFixed(0) : growth.toFixed(0) }}%</span> /
          <span class="online-count"> {{ store.online }}人在线</span>
        </div>
      </div>
      <div class="chart-controls flex items-center gap-[8px]">
        <el-radio-group v-model="chartMode" size="small" class="rounded-[4px] overflow-hidden shadow-[0_1px_2px_rgba(0,0,0,0.05)]" @change="handleChartModeChange">
          <el-radio-button label="daily">按天</el-radio-button>
          <el-radio-button label="monthly">按月</el-radio-button>
        </el-radio-group>
        
        <el-select 
          v-if="chartMode === 'monthly'" 
          v-model="selectedYear" 
          placeholder="选择年份" 
          @change="fetchMonthlyStats" 
          size="small"
          class="chart-select">
          <el-option
            v-for="year in availableYears"
            :key="year"
            :label="year + '年'"
            :value="year"
          />
        </el-select>
        
        <el-select 
          v-if="chartMode === 'daily'" 
          v-model="selectedDays" 
          placeholder="选择天数" 
          @change="fetchDailyStats" 
          size="small"
          class="chart-select">
          <el-option :label="'最近7天'" :value="7" />
          <el-option :label="'最近15天'" :value="15" />
          <el-option :label="'最近30天'" :value="30" />
        </el-select>
      </div>
    </div>
    <div class="div-item-charts" v-loading="loading">
      <v-chart class="mt-[10px] h-[400px]" :option="option" autoresize />
    </div>
  </div>
</template>
<style scoped lang="less">
/*
  只剩两类元素自己的样式，都是工具类替代不了的：
  - .chart-select 的定宽：Element Plus 有 .el-select { width: var(--el-select-width) }（无层级规则），
    会压过 @layer utilities 里的 w-[100px]；改 --el-select-width 也一样，同一元素上无层级声明优先。
  - 按钮组的内边距/高度在库内部元素上，工具类加不上去；且 .el-radio-button--small 的选择器
    比特异性高于裸 :deep()，需要 .chart-controls 作为锚点。

  原文件是非 scoped 块，样式全局泄漏（§4.7），迁移后改为 scoped。
*/
.chart-select {
  width: 100px;
}

.chart-controls :deep(.el-radio-button__inner) {
  padding: 6px 12px;
  height: 28px;
  line-height: 1;
}
</style>
<script setup>
import { ref, onMounted, provide, computed, watch } from 'vue';
import VChart, { THEME_KEY } from "vue-echarts";
import { use } from "echarts/core";
import { LineChart } from "echarts/charts";
import { GridComponent, TooltipComponent } from "echarts/components";
import { CanvasRenderer } from "echarts/renderers";
import { useAdminStore } from "@/store";
import { getMonthlyVisitStats, getDailyVisitStats } from "@/api/admin";

provide(THEME_KEY, "light");
const store = useAdminStore();
const loading = ref(false);

// 图表模式
const chartMode = ref('daily'); // 默认为按天显示

// 图表数据
const monthlyData = ref([0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0]);
const dailyData = ref([]);
const dailyLabels = ref([]);
const monthNames = ["1月", "2月", "3月", "4月", "5月", "6月", "7月", "8月", "9月", "10月", "11月", "12月"];

// 年份和天数选择
const currentYear = new Date().getFullYear();
const selectedYear = ref(currentYear);
const availableYears = ref([currentYear, currentYear - 1, currentYear - 2]);
const selectedDays = ref(30); // 默认显示30天的数据

// 增长率计算
const growth = computed(() => {
  if (chartMode.value === 'monthly') {
    // 月度增长率计算
    const currentMonth = new Date().getMonth();
    const lastMonth = currentMonth === 0 ? 11 : currentMonth - 1;
    const currentMonthData = monthlyData.value[currentMonth];
    const lastMonthData = monthlyData.value[lastMonth];
    
    if (lastMonthData === 0) return 0;
    return ((currentMonthData - lastMonthData) / lastMonthData) * 100;
  } else {
    // 日度增长率计算
    if (dailyData.value.length < 2) return 0;
    const today = dailyData.value[dailyData.value.length - 1];
    const yesterday = dailyData.value[dailyData.value.length - 2];
    
    if (yesterday === 0) return 0;
    return ((today - yesterday) / yesterday) * 100;
  }
});

// 处理图表模式变化
const handleChartModeChange = (mode) => {
  if (mode === 'monthly') {
    fetchMonthlyStats();
  } else {
    fetchDailyStats();
  }
};

// 获取月度访问统计数据
const fetchMonthlyStats = async (year = selectedYear.value) => {
  loading.value = true;
  try {
    const data = await getMonthlyVisitStats(year);
    // 重置数据
    monthlyData.value = [0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0];
    
    // 填充数据
    data.forEach(item => {
      if (item.month >= 1 && item.month <= 12) {
        monthlyData.value[item.month - 1] = item.visit_count;
      }
    });
  } catch (error) {
    console.error('获取月度访问统计数据失败:', error);
  } finally {
    loading.value = false;
  }
};

// 获取每日访问统计数据
const fetchDailyStats = async (days = selectedDays.value) => {
  loading.value = true;
  try {
    const data = await getDailyVisitStats(days);
    
    // 重置并填充数据
    dailyData.value = [];
    dailyLabels.value = [];
    
    data.forEach(item => {
      dailyLabels.value.push(formatDate(item.date));
      dailyData.value.push(item.visit_count);
    });
  } catch (error) {
    console.error('获取每日访问统计数据失败:', error);
  } finally {
    loading.value = false;
  }
};

// 格式化日期，例如将 "2024-07-09" 转换为 "7-9"，今天显示为"今日"
const formatDate = (dateStr) => {
  const today = new Date();
  const todayStr = today.getFullYear() + '-' + 
                  String(today.getMonth() + 1).padStart(2, '0') + '-' + 
                  String(today.getDate()).padStart(2, '0');
  
  if (dateStr === todayStr) {
    return '今日';
  }
  
  const parts = dateStr.split('-');
  if (parts.length === 3) {
    // 去掉前导零
    const month = parseInt(parts[1], 10);
    const day = parseInt(parts[2], 10);
    return `${month}-${day}`;
  }
  return dateStr;
};

use([GridComponent, TooltipComponent, LineChart, CanvasRenderer]);

// 图表配置
const option = computed(() => {
  if (chartMode.value === 'monthly') {
    return {
      xAxis: {
        type: "category",
        data: monthNames,
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
          data: monthlyData.value,
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
    };
  } else {
    return {
      xAxis: {
        type: "category",
        data: dailyLabels.value,
        axisLabel: {
          // fontWeight: 'bold' // 加粗 X 轴标签
          rotate: dailyLabels.value.length > 15 ? 45 : 0, // 如果标签太多，旋转45度
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
          if (date === '今日') {
            return `今日<br/><div style="font-size: 13px"><span style="display: inline-block; width: 12px; height: 12px; background: #348fff; border-radius: 50%; vertical-align: middle; margin-right: 6px;"></span>浏览 <span style="font-weight: 800">${value}</span></div>`;
          }
          return `${date}日<br/><div style="font-size: 13px"><span style="display: inline-block; width: 12px; height: 12px; background: #348fff; border-radius: 50%; vertical-align: middle; margin-right: 6px;"></span>浏览 <span style="font-weight: 800">${value}</span></div>`;
        },
      },
      series: [
        {
          data: dailyData.value,
          type: "line",
          smooth: true,
          color: "#2788ff",
          symbol: function(dataIndex, params) {
            // 为最后一个点（今日）添加标记
            return dataIndex === dailyLabels.value.length - 1 ? 'circle' : 'none';
          },
          symbolSize: function(dataIndex, params) {
            // 为最后一个点（今日）设置大小
            return dataIndex === dailyLabels.value.length - 1 ? 8 : 0;
          },
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
    };
  }
});

onMounted(() => {
  fetchDailyStats(); // 默认加载每日数据
});
</script>
