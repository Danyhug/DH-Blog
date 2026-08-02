<template>
    <SectionPanel title="调度方式" subtitle="仅在请求未指定 provider（即 auto）时生效">
        <template #icon>
            <el-icon>
                <Switch />
            </el-icon>
        </template>

        <p class="note">
            能力、健康、密钥与配额过滤始终优先——调度方式只决定通过过滤后的先后顺序，
            所以换策略不会让一个用不了的供应商被选中。
        </p>

        <div v-loading="loading" class="grid grid-cols-1 lg:grid-cols-3 gap-4">
            <button v-for="option in strategies" :key="option.value" type="button" class="option"
                :class="{
                    'option--active': strategy === option.value,
                    'option--locked': !option.implemented
                }" :disabled="!option.implemented" @click="onPick(option.value)">
                <span class="option-head">
                    <span class="option-label">{{ option.label }}</span>
                    <el-tag v-if="!option.implemented" size="small" type="info" effect="plain">开发中</el-tag>
                    <el-icon v-else-if="strategy === option.value" class="option-check">
                        <CircleCheckFilled />
                    </el-icon>
                </span>
                <span class="option-desc">{{ option.description }}</span>
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

onMounted(load);
</script>

<style scoped>
.note {
    margin: 0 0 16px;
    padding: 10px 12px;
    border-radius: 8px;
    background-color: #f7f9fc;
    font-size: 12px;
    line-height: 1.7;
    color: #667085;
}

.option {
    display: flex;
    flex-direction: column;
    gap: 8px;
    padding: 16px;
    text-align: left;
    border: 1px solid #e6e9ee;
    border-radius: 12px;
    background-color: #fff;
    cursor: pointer;
    transition: all 0.2s;
}

.option:hover:not(:disabled) {
    border-color: #b9d4ff;
}

.option--active {
    border-color: #3f8cff;
    background-color: #f5f9ff;
    box-shadow: 0 0 0 3px rgba(63, 140, 255, 0.08);
}

.option--locked {
    cursor: not-allowed;
    background-color: #fafafa;
    opacity: 0.7;
}

.option-head {
    display: flex;
    align-items: center;
    gap: 8px;
}

.option-label {
    font-size: 14px;
    font-weight: 600;
    color: #1f2937;
}

.option-check {
    margin-left: auto;
    color: #3f8cff;
    font-size: 17px;
}

.option-desc {
    font-size: 12px;
    line-height: 1.7;
    color: #98a2b3;
}
</style>
