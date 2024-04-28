<template>
  <div>
    <el-row>
      <el-col>
        <el-card class="box">
          <img src="https://shadow.elemecdn.com/app/element/hamburger.9cf7b091-55e9-11e9-a976-7f4d0b07eef6.png"
            class="image" />

          <div>
            <p class="title">文章标题3</p>
            <div class="schedule">
              <el-progress :percentage="50"></el-progress>
              <p>已阅读时长：1分24秒</p>
            </div>
          </div>

          <div class="links">
            <a>
              <Icon iconName="icon-31erweima" iconSize="2"></Icon>
            </a>
            <a>
              <Icon iconName="icon-home" iconSize="2"></Icon>
            </a>
            <a>
              <Icon iconName="icon-forward" iconSize="2"></Icon>
            </a>
            <a>
              <Icon iconName="icon-share" iconSize="2"></Icon>
            </a>
            <a>
              <Icon iconName="icon-setting" iconSize="2"></Icon>
            </a>
          </div>

          <div class="tags">
            <el-divider>
              <Icon iconName="icon-layers" iconSize="1"></Icon>
            </el-divider>
            <a href="" class="tag" v-for="item in 9" :style="{ backgroundColor: getRandomColor() }">原神</a>
          </div>
        </el-card>
      </el-col>
    </el-row>
  </div>
</template>

<script>
export default {
  data() {
    return {};
  },
  methods: {
    getRandomColor() {
      const tagColors = [
        "#FF6347", // 亮红色  
        "#2ECC71", // 亮绿色  
        "#3498DB", // 亮蓝色  
        "#9B59B6", // 紫色  
        "#F1C40F", // 黄色  
        "#1ABC9C", // 青色  
        "#F39C12", // 橙色  
        "#E67E22", // 金色  
        "#16A085", // 海蓝色  
        "#D2527F"  // 粉色  
      ]
      return tagColors[Math.floor(Math.random() * tagColors.length)];
    },
  },
};
</script>

<style lang="less" scoped>
.box {
  padding: 0 20px;
  height: 100vh;
  background-color: #fff;
}

.schedule {
  p {
    font-size: 14px;
    color: #606266;
  }
}

.title {
  font-size: 1.25rem;
  margin: 0.75rem 0;
}

.box {
  width: 100%;
}

.links {
  margin-top: 30px;
  padding: 8px 0;
  display: grid;
  grid-template-columns: repeat(5, 1fr);
  justify-items: center;
  width: 100%;
  color: #909399;
  font-size: 14px;
}

.tags {
  width: 100%;
  display: flex;
  flex-wrap: wrap;
  justify-content: space-between;
  color: #909399;
  font-size: 14px;

  a {
    width: 21%;
    margin-bottom: 10px;
    margin-top: 0;
    color: #fff;
    border-radius: 5px;
    padding: 3px 6px;
  }
}
</style>
