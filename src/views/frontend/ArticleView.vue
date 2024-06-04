<template>
  <!-- 文章浏览页 -->
  <div>
    <div class="blog-container">
      <MdPreview :modelValue="content" previewTheme="github" />
    </div>
    <div class="info">
      <span>
        更新于 2024-4-28 21:31:22
      </span>
      <span>
        阅读次数 22 次
      </span>
    </div>
  </div>
</template>

<script>
export default {
  name: 'HomeView',
  data() {
    return {
      // 文章信息
      id: 1,
      title: '',
      content: `盼望着，盼望着，东风来了，春天的脚步近了。一切都像刚睡醒的样子，欣欣然张开了眼。山朗润起来了，水涨起来了，太阳的脸红起来了。

小草偷偷地从土里钻出来，嫩嫩的，绿绿的。园子里，田野里，瞧去，一大片一大片满是的。坐着，趟着，打两个滚，踢几脚球，赛几趟跑，捉几回迷藏。风轻悄悄的，草软绵绵的。

桃树、杏树、梨树，你不让我，我不让你，都开满了花赶趟儿。红的像火，粉的像霞，白的像雪。花里带着甜味儿；闭了眼，树上仿佛已经满是桃儿、杏儿、梨儿。花下成千成百的蜜蜂嗡嗡地闹着，大小的蝴蝶飞来飞去。

野花遍地是：杂样儿，有名字的，没名字的，散在草丛里像眼睛，像星星，还眨呀眨的。“吹面不寒杨柳风”，不错的，像母亲的手抚摸着你。风里带来些新翻的泥土的气息，混着青草味儿，还有各种花的香都在微微润湿的空气里酝酿。

鸟儿将窠巢安在繁花嫩叶当中，高兴起来了，呼朋引伴地卖弄清脆的喉咙，唱出宛转的曲子，与轻风流水应和着。牛背上牧童的短笛，这时候也成天嘹亮地响。

雨是最寻常的，一下就是三两天。可别恼。看，像牛毛，像花针，像细丝，密密地斜织着，人家屋顶上全笼着一层薄烟。树叶儿却绿得发亮，小草儿也青得逼你的眼。傍晚时候，上灯了，一点点黄晕的光烘托出一片安静而和平的夜。在乡下，小路上，石桥边，有撑起伞慢慢走着的人；还有地里工作的农民，披着蓑戴着笠。他们的房屋稀稀疏疏的在雨里静默着。

天上风筝渐渐多了，地上孩子也多了。城里乡下，家家户户，老老小小他们也赶趟儿似的，一个个都出来了。舒活舒活筋骨抖擞抖擞精神各做各的一份事去。“一年之计在于春”；刚起头儿有的是工夫有的是希望。

春天像刚落地的娃娃从头到脚都是新的它生长着。春天像小姑娘花枝招展的笑着走着。春天像健壮的青年有铁一般的胳膊和腰脚他领着我们上前去。`,
      created: 0,
      update: 0,
      viewnum: 0
    }
  },
  mounted() {

  },
  methods: {

  }
}
</script>
<style lang="less" scoped>
.blog-container {
  padding: 1.25rem 1.5rem;
  padding-top: 0;
}

.left {
  display: flex;
  flex-direction: column;
  align-items: center;
  width: 30%;

  top: 0;
  margin: .6rem 0;
}

.info {
  padding: 10px 0;
  font-size: 12px;
  color: #606266;
  text-align: right;
  border-top: 1px solid var(--grey-4);

  span {
    margin: 0 10px;
  }
}
</style>
