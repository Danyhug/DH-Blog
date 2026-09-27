<template>
    <section class="panel bg-white border border-[#edf0f3] rounded-[14px] shadow-[0_1px_2px_rgba(16,24,40,0.04)] overflow-hidden">
        <header class="flex items-center gap-[12px] px-[20px] py-[16px] border-b border-[#f2f4f7]">
            <span v-if="$slots.icon" class="inline-flex items-center justify-center w-[34px] h-[34px] rounded-[10px] bg-[#eef4ff] text-[#3f8cff] text-[17px]">
                <slot name="icon" />
            </span>
            <div class="min-w-0 flex-1">
                <h3 class="panel-title m-0 text-[15px] text-[#1f2937]">{{ title }}</h3>
                <p v-if="subtitle" class="mt-[3px] mb-0 text-[12px] leading-[1.5] text-[#98a2b3]">{{ subtitle }}</p>
            </div>
            <div v-if="$slots.extra" class="shrink-0">
                <slot name="extra" />
            </div>
        </header>
        <!-- padding 两套写法整体二选一：Tailwind 里 p-0 的声明位置在 px-*/pt-*/pb-* 之前，靠 class 顺序压不住 -->
        <div :class="flush ? 'p-0' : 'px-[20px] pt-[18px] pb-[20px]'">
            <slot />
        </div>
    </section>
</template>

<script setup lang="ts">
// 网关各标签页共用的分区外壳。抽出来是为了让标题、留白、圆角只有一处定义，
// 不然每加一块内容就多一套 el-card 的默认样式，页面很快就花了。
withDefaults(defineProps<{ title: string; subtitle?: string; flush?: boolean }>(), {
    subtitle: '',
    flush: false
});
</script>

<style scoped>
/*
  组件类：相邻面板之间的间距只能靠兄弟选择器表达，工具类没有等价写法，
  因此模板上保留 .panel 这个类名作为选择器锚点。
*/
.panel+.panel {
    margin-top: 18px;
}

/*
  全局 style.less 里的 `h1, h2, h3 { font-weight: 400 }` 是无层级规则，
  会压过 @layer utilities 里的 font-semibold，所以标题字重只能写在这里。
*/
.panel-title {
    font-weight: 600;
}
</style>
