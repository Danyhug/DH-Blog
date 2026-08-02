<template>
    <section class="panel">
        <header class="panel-head">
            <span v-if="$slots.icon" class="panel-icon">
                <slot name="icon" />
            </span>
            <div class="min-w-0 flex-1">
                <h3 class="panel-title">{{ title }}</h3>
                <p v-if="subtitle" class="panel-sub">{{ subtitle }}</p>
            </div>
            <div v-if="$slots.extra" class="shrink-0">
                <slot name="extra" />
            </div>
        </header>
        <div class="panel-body" :class="{ 'panel-body--flush': flush }">
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
.panel {
    background-color: #fff;
    border: 1px solid #edf0f3;
    border-radius: 14px;
    box-shadow: 0 1px 2px rgba(16, 24, 40, 0.04);
    overflow: hidden;
}

.panel+.panel {
    margin-top: 18px;
}

.panel-head {
    display: flex;
    align-items: center;
    gap: 12px;
    padding: 16px 20px;
    border-bottom: 1px solid #f2f4f7;
}

.panel-icon {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    width: 34px;
    height: 34px;
    border-radius: 10px;
    background-color: #eef4ff;
    color: #3f8cff;
    font-size: 17px;
}

.panel-title {
    margin: 0;
    font-size: 15px;
    font-weight: 600;
    color: #1f2937;
}

.panel-sub {
    margin: 3px 0 0;
    font-size: 12px;
    line-height: 1.5;
    color: #98a2b3;
}

.panel-body {
    padding: 18px 20px 20px;
}

.panel-body--flush {
    padding: 0;
}
</style>
