<template>
    <div class="relative rounded-[10px] border border-[#e9edf2] bg-[#fafbfc] py-3 pl-3.5 pr-10">
        <pre class="m-0 whitespace-pre-wrap break-all text-xs leading-[1.75] text-[#476582]">{{ code }}</pre>
        <el-button class="!absolute right-2.5 top-2" link type="primary" :icon="CopyDocument"
            @click="onCopy" />
    </div>
</template>

<script setup lang="ts">
import { CopyDocument } from '@element-plus/icons-vue';
import { notify } from '@/utils/notification';

// 网关页里要照抄的命令和配置有十来处，样式和复制逻辑集中在这一个组件里，
// 免得每加一段接入说明就重写一遍 pre + 按钮。
const props = withDefaults(defineProps<{ code: string; label?: string }>(), { label: '内容' });

async function onCopy() {
    try {
        await navigator.clipboard.writeText(props.code);
        notify.success(`${props.label}已复制`);
    } catch {
        notify.warning('复制失败，请手动选中复制');
    }
}
</script>
