<template>
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
        <el-table-column type="index" label="No" min-width="8%"></el-table-column>
        <el-table-column label="城市" min-width="21%" show-overflow-tooltip>
          <template #default="scope">
            <div>
              <Icon :iconName="getTelecom(scope.row.city)" iconSize="1.8"></Icon>

              <span>{{ getCity(scope.row.city) }}</span>
            </div>
          </template>
        </el-table-column>
        <el-table-column prop="ipAddress" label="IP" min-width="23%"></el-table-column>
        <el-table-column prop="accessCount" label="访问" min-width="12%"></el-table-column>
        <el-table-column prop="bannedCount" label="Ban" min-width="9%"></el-table-column>
      </el-table>
    </div>
  </div>
</template>

<style lang="less">
.chart-left {
  width: 44%;
  --el-border-radius-base: 6px;

  .item-table {
    margin-top: 15px;
    padding: 0 8px;
  }
}
</style>

<script setup lang="ts">
import { ref, reactive, watch, onMounted } from 'vue';
import { getOverviewLog } from "@/api/admin";
import { IpStat } from "@/types/IpStat";
import { plusDate } from "@/utils/tool";

const today = new Date();
const formatDate = (date: Date) => `${date.getFullYear()}-${date.getMonth() + 1}-${date.getDate()}`;

// 访问记录
const visitRadio = ref("day");

// 分页参数
const page = reactive({
  page: 1,
  pageSize: 100,
  startTime: formatDate(today),
  endTime: formatDate(plusDate(today, 1))
});

// 更新时间范围
const updateTimeRange = (type: string) => {
  switch (type) {
    case "day":
      page.startTime = formatDate(today);
      page.endTime = formatDate(plusDate(today, 1));
      break;
    case "week":
      page.startTime = formatDate(plusDate(today, -today.getDay()));
      page.endTime = formatDate(plusDate(today, 1));
      break;
    case "month":
      page.startTime = formatDate(plusDate(today, -today.getDate() + 1));
      page.endTime = formatDate(plusDate(today, 1));
      break;
    case "total":
      page.startTime = "2020-10-30";
      page.endTime = "2099-11-14";
      break;
  }
};

// 获取访问记录
const ipData = ref<IpStat[]>([]);
const getVisit = async () => {
  const data = await getOverviewLog(page.page, page.pageSize, page.startTime, page.endTime);
  ipData.value = data.list.sort((a, b) => b.accessCount - a.accessCount);
};

// 获取运营商
const getTelecom = (city: string) => {
  if (city.includes("/")) {
    const arr = city.split("/");
    if (arr[0].includes("联通")) {
      return "icon-cuc"
    } else if (arr[0].includes("电信")) {
      return "icon-dianxinxuke"
    } else if (arr[0].includes("移动")) {
      return "icon-mobile"
    } else {
      return "icon-weizhi"
    }
  }
  return "icon-weizhi";
}

const getCity = (city: string) => {
  if (city.includes("/")) {
    const arr = city.split("/");
    return arr[1];
  }
  return city.replace(" ", "");
}

// 监听选项变化
watch(visitRadio, (newVal) => {
  updateTimeRange(newVal);
  getVisit();
});

// 组件挂载时获取数据
onMounted(() => getVisit());
</script>
