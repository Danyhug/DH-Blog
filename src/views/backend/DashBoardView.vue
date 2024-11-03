<template>
  <div class="dash-container">
    <!-- 总览 -->
    <ul class="total-item">
      <li>
        <div class="item-left">
          <span class="title">总文章数</span>
          <span class="num">100</span>
          <span class="trend">+30%</span>
        </div>
        <div class="item-right">
          <Icon iconName="icon-history" :iconSize="iconSize.itemTotal" fill="#fff"></Icon>
        </div>
      </li>
      <li>
        <div class="item-left">
          <span class="title">总分类数</span>
          <span class="num">100</span>
          <span class="trend">+30%</span>
        </div>
        <div class="item-right">
          <Icon iconName="icon-layout" :iconSize="iconSize.itemTotal" fill="#fff"></Icon>
        </div>
      </li>
      <li>
        <div class="item-left">
          <span class="title">总标签数</span>
          <span class="num">100</span>
          <span class="trend">+30%</span>
        </div>
        <div class="item-right">
          <Icon iconName="icon-training" :iconSize="iconSize.itemTotal" fill="#fff"></Icon>
        </div>
      </li>
      <li>
        <div class="item-left">
          <span class="title">总评论数</span>
          <span class="num">100</span>
          <span class="trend">+30%</span>
        </div>
        <div class="item-right">
          <Icon iconName="icon-smile" :iconSize="iconSize.itemTotal" fill="#fff"></Icon>
        </div>
      </li>
    </ul>

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

.total-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 16px;
  background-color: #fff;
  margin-bottom: 10px;

  li {
    width: 23.6%;
    height: 90px;
    background: #fff;
    border-radius: 14px;
    padding: 20px 18px 0;
    list-style: none;
    box-shadow: 0 4px 20px rgba(0, 0, 0, 0.08);
    display: flex;
    justify-content: space-between;

    .item-left {
      width: 70%;

      .title {
        font-size: 13px;
        color: #78829d;
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
      border-radius: 12px;
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

.chart-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 16px;
  margin-bottom: 10px;
  height: 450px;

  >div {
    height: 100%;
    border-radius: 16px;
    padding-top: 20px;
    padding-bottom: 30px;
    box-shadow: 0 4px 20px rgba(0, 0, 0, 0.08);
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
const iconSize = reactive({
  itemTotal: 1.2
})
</script>