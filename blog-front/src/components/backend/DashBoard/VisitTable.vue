<template>
  <div class="chart-left">
    <div class="item-top">
      <div>
        <div class="item-title">访问记录</div>
        <div class="item-sub">
          今日访问次数
          <span> {{ totalVisits }} </span>
          <!-- / -->
          <!-- <span class="sub"> -37% </span> -->
        </div>
      </div>

      <el-radio-group v-model="visitRadio">
        <el-radio-button label="今日" value="day" />
        <el-radio-button label="此周" value="week" />
        <el-radio-button label="本月" value="month" />
        <el-radio-button label="累计" value="total" />
      </el-radio-group>
    </div>

    <div class="item-table" v-loading="loading">
      <el-table :data="ipData" @scroll.native="loadMore" style="overflow-y: auto; max-height: 315px;"
        @row-click="handleRowClick" ref="vechart" :row-class-name="tableRowClassName">

        <el-table-column type="index" label="No" min-width="8%"></el-table-column>
        <el-table-column label="城市" min-width="21%" show-overflow-tooltip>
          <template #default="scope">
            <div>
              <Icon class="tele-icon" :iconName="getTelecom(scope.row.city)" iconSize="1.8"></Icon>
              <span>{{ getCity(scope.row.city) }}</span>
            </div>
          </template>
        </el-table-column>
        <el-table-column prop="ipAddress" label="IP" min-width="23%"></el-table-column>
        <el-table-column prop="accessCount" label="访问" min-width="12%"></el-table-column>
        <el-table-column prop="bannedCount" label="Ban" min-width="9%"></el-table-column>
      </el-table>
    </div>


    <!-- 对话框 -->
    <el-dialog :title="banText" v-model="dialogVisible" width="30%">
      <span>确定要{{ banText }}IP地址 {{ selectedIp?.ipAddress }} 吗？</span>
      <template #footer>
        <span class="dialog-footer">
          <el-button @click="dialogVisible = false">取消</el-button>
          <el-button type="primary" @click="banIp">{{ banText }}</el-button>
        </span>
      </template>
    </el-dialog>
  </div>
</template>

<style lang="less">
.chart-left {
  width: 45%;
  --el-border-radius-base: 6px;

  .item-table {
    margin-top: 15px;
    padding: 0 8px;
  }
}

.tele-icon {
  margin-right: 5px;
}

.ban-row {
  background-color: rgba(201, 61, 64, .1) !important;
}
</style>

<script setup lang="ts">
import { ref, reactive, watch, onMounted, computed } from 'vue';
import { getOverviewLog, postBanIp } from "@/api/admin";
import { IpStat } from "@/types/IpStat";
import { plusDate } from "@/utils/tool";
import { debounce } from '@/utils/tool';

const today = new Date();
const loading = ref(false)
const vechart = ref<HTMLElement>();
const selectedIp = ref<IpStat>()
const dialogVisible = ref(false);
const banText = ref("封禁");
const formatDate = (date: Date) => `${date.getFullYear()}-${date.getMonth() + 1}-${date.getDate()}`;

// 访问记录
const visitRadio = ref("day");

// 计算总访问量
const totalVisits = computed(() => {
  return ipData.value.reduce((sum, ip) => sum + ip.accessCount, 0);
});

// 分页参数
const page = reactive({
  page: 1,
  pageSize: 30,
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
  loading.value = true;
  const data = await getOverviewLog(page.page, page.pageSize, page.startTime, page.endTime);
  ipData.value = ipData.value.concat(data.list).sort((a, b) => b.accessCount - a.accessCount);
  setTimeout(() => loading.value = false, 300)
};

// 获取运营商
const getTelecom = (city: string) => {
  const arr = city.split("/");
  if (arr.length == 3) { city = arr[1] + '/' + arr[2] }
  if (city.includes("/")) {
    const arr = city.split("/");
    if (arr[0].includes("联通")) {
      return "icon-cuc"
    } else if (arr[0].includes("电信")) {
      return "icon-dianxinxuke"
    } else if (arr[0].includes("移动")) {
      return "icon-mobile"
    } else {
      return "icon-waixingren"
    }
  }
  return "icon-waixingren";
}

const getCity = (city: string) => {
  const arr = city.split("/");

  if (arr.length == 2) {
    return arr[1];
  } else if (arr.length == 3) {
    return arr[2];
  } else if (city.includes(" ")) {
    return city.split(' ')[1];
  }
  return city;
}

const loadMore = debounce((event: Event) => {
  const target = event.target as HTMLElement;
  if (target && !loading.value) {
    const { scrollTop, clientHeight, scrollHeight } = target;
    if (scrollTop + clientHeight >= scrollHeight - 60) {
      // 滚动到底部时加载更多数据
      page.page++;
      getVisit().then(() => target.scrollTop = scrollTop + 2)
    }
  }
}, 160);

// 监听选项变化
watch(visitRadio, (newVal) => {
  page.page = 1;
  updateTimeRange(newVal);
  ipData.value = []
  getVisit();
});

// 组件挂载时获取数据
onMounted(() => getVisit());

function handleRowClick(row: any) {
  selectedIp.value = row;
  if (row.banStatus == 1) {
    banText.value = "解封"
  } else {
    banText.value = "封禁"
  }

  dialogVisible.value = true;
}

function banIp() {
  if (selectedIp.value == undefined) return

  postBanIp(selectedIp.value?.ipAddress, selectedIp.value?.banStatus).then(r => {
    dialogVisible.value = false;
    ipData.value = []
    getVisit();
    ElMessage.success(banText.value + "成功")
  })
}

const tableRowClassName = ({ row }: { row: IpStat }) => {
  console.log(row.banStatus == 1 ? 'ban-row' : '')
  return row.banStatus == 1 ? 'ban-row' : ''
}
</script>
