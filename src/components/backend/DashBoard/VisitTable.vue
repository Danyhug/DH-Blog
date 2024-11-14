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
        <el-table-column type="index" label="序号" min-width="9%"></el-table-column>
        <el-table-column prop="city" label="城市" min-width="20%"></el-table-column>
        <el-table-column prop="ip" label="IP" min-width="22%"></el-table-column>
        <el-table-column prop="count" label="访问数" min-width="14%"></el-table-column>
        <el-table-column prop="ban" label="Ban" min-width="10%"></el-table-column>
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

<script setup>
const visitRadio = ref('day')

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
</script>