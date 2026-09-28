<template>
    <SectionPanel title="调度方式" subtitle="仅在请求未指定 provider（即 auto）时生效">
        <template #icon>
            <el-icon>
                <Switch />
            </el-icon>
        </template>

        <p class="mb-[16px] px-[12px] py-[10px] rounded-[8px] bg-[#f7f9fc] text-[12px] leading-[1.7] text-[#667085]">
            能力、健康、密钥与配额过滤始终优先——调度方式只决定通过过滤后的先后顺序，
            所以换策略不会让一个用不了的供应商被选中。
        </p>

        <div v-loading="loading" class="grid grid-cols-1 lg:grid-cols-3 gap-4">
            <button v-for="option in strategies" :key="option.value" type="button"
                class="flex flex-col gap-[8px] p-[16px] text-left border rounded-[12px] transition-all duration-200 ease-[ease]"
                :class="optionClass(option)" :disabled="!option.implemented" @click="onPick(option.value)">
                <span class="flex items-center gap-[8px]">
                    <span class="text-[14px] font-semibold text-[#1f2937]">{{ option.label }}</span>
                    <el-tag v-if="!option.implemented" size="small" type="info" effect="plain">开发中</el-tag>
                    <el-icon v-else-if="strategy === option.value" :size="17" color="#3f8cff" class="ml-auto">
                        <CircleCheckFilled />
                    </el-icon>
                </span>
                <span class="text-[12px] leading-[1.7] text-[#98a2b3]">{{ option.description }}</span>
            </button>
        </div>
    </SectionPanel>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue';
import { CircleCheckFilled, Switch } from '@element-plus/icons-vue';
import { notify } from '@/utils/notification';
import SectionPanel from './SectionPanel.vue';
import {
    getGatewaySettings,
    updateGatewaySettings,
    type RoutingStrategy,
    type StrategyOption
} from '@/api/gateway';

const strategy = ref<RoutingStrategy>('balanced');
const strategies = ref<StrategyOption[]>([]);
const loading = ref(false);

async function load() {
    loading.value = true;
    try {
        const settings = await getGatewaySettings();
        strategy.value = settings.routingStrategy;
        strategies.value = settings.strategies;
    } finally {
        loading.value = false;
    }
}

// 未接入的方式在界面上就点不动，后端也会拒绝——否则等于显示的和实际执行的不是一回事
async function onPick(value: RoutingStrategy) {
    if (value === strategy.value) return;
    const option = strategies.value.find((item) => item.value === value);
    if (!option?.implemented) return;

    const previous = strategy.value;
    strategy.value = value;
    try {
        await updateGatewaySettings({ routingStrategy: value });
    } catch (error) {
        strategy.value = previous;
        throw error;
    }
    notify.success(`调度方式已切换为「${option.label}」`);
}

/**
 * 每个状态给一份完整的样式串，而不是拆成多个条件类。
 * 原因：Tailwind 无法靠 class 顺序覆盖同属性（border-color / background-color 的声明顺序由
 * 生成结果的排序决定），同属性写两处会随机有一处失效。锁定态优先判断，与原 CSS 中
 * `.option--locked` 排在 `.option--active` 之后的实际效果一致。
 */
function optionClass(option: StrategyOption) {
    if (!option.implemented) {
        return 'cursor-not-allowed border-[#e6e9ee] bg-[#fafafa] opacity-70';
    }
    if (option.value === strategy.value) {
        return 'cursor-pointer border-[#3f8cff] bg-[#f5f9ff] shadow-[0_0_0_3px_rgba(63,140,255,0.08)]';
    }
    return 'cursor-pointer border-[#e6e9ee] bg-white hover:border-[#b9d4ff]';
}

onMounted(load);
</script>
