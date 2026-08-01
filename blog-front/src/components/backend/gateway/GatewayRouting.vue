<template>
    <el-card shadow="hover" v-loading="loading">
        <template #header>
            <span class="flex items-center">
                <el-icon class="mr-2">
                    <Switch />
                </el-icon> 调度方式
            </span>
        </template>

        <el-alert type="info" :closable="false" show-icon class="mb-4">
            <template #title>
                仅在请求未指定 <code>provider</code>（即 <code>auto</code>）时生效。
                能力、健康与配额过滤始终优先——调度方式只决定通过过滤后的先后顺序。
            </template>
        </el-alert>

        <el-radio-group v-model="strategy" class="w-full flex flex-col gap-3" @change="onChange">
            <el-radio v-for="option in strategies" :key="option.value" :value="option.value"
                class="!mr-0 !h-auto !items-start w-full border border-gray-200 rounded-lg px-4 py-3 transition-colors"
                :class="strategy === option.value ? 'border-blue-400 bg-blue-50/50' : 'hover:border-gray-300'">
                <div class="flex flex-col gap-1 whitespace-normal">
                    <span class="flex items-center gap-2 font-medium">
                        {{ option.label }}
                        <el-tag v-if="!option.implemented" type="warning" size="small" effect="plain">开发中</el-tag>
                    </span>
                    <span class="text-xs text-gray-500 leading-relaxed">{{ option.description }}</span>
                </div>
            </el-radio>
        </el-radio-group>

        <el-alert v-if="activeStrategyPending" type="warning" :closable="false" show-icon class="mt-4">
            <template #title>
                「{{ activeStrategyLabel }}」尚未接入，当前请求实际仍按「负载均衡」调度。
            </template>
        </el-alert>
    </el-card>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue';
import { Switch } from '@element-plus/icons-vue';
import { notify } from '@/utils/notification';
import {
    getGatewaySettings,
    updateGatewaySettings,
    type RoutingStrategy,
    type StrategyOption
} from '@/api/gateway';

const strategy = ref<RoutingStrategy>('balanced');
const strategies = ref<StrategyOption[]>([]);
const loading = ref(false);

const activeStrategy = computed(() => strategies.value.find((item) => item.value === strategy.value));
const activeStrategyLabel = computed(() => activeStrategy.value?.label ?? strategy.value);
// 选了尚未接入的方式时必须显式提示，否则界面等于谎称路由已改变
const activeStrategyPending = computed(() => activeStrategy.value?.implemented === false);

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

async function onChange(value: any) {
    const next = value as RoutingStrategy;
    try {
        await updateGatewaySettings({ routingStrategy: next });
    } catch (error) {
        await load(); // 保存失败时回到服务端的真实取值
        throw error;
    }
    const option = strategies.value.find((item) => item.value === next);
    if (option && !option.implemented) {
        notify.warning(`已保存，但「${option.label}」尚未接入，实际仍按负载均衡调度`);
        return;
    }
    notify.success(`调度方式已切换为「${option?.label ?? next}」`);
}

onMounted(load);
</script>
