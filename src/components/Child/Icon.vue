<template>
  <svg class="icon" aria-hidden="true" :style="`width:${iconSize}em; height:${iconSize}em`">
    <use :xlink:href="'#' + iconName"></use>
  </svg>
</template>
<script>
export default {
  name: 'Icon',
  props: {
    // 字体名
    iconName: {
      type: String,
      require: true
    },
    // 图标宽度 em单位
    iconSize: {
      type: String,
      require: false,
      default: '2'
    }
  }
}
</script>