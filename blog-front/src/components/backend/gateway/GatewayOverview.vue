<template>
    <div v-loading="loading">
        <div class="grid grid-cols-2 xl:grid-cols-4 gap-4 mb-[18px]">
            <div v-for="tile in tiles" :key="tile.label" class="tile">
                <span class="tile-icon" :style="{ backgroundColor: tile.tint, color: tile.color }">
                    <el-icon>
                        <component :is="tile.icon" />
                    </el-icon>
                </span>
                <div class="min-w-0">
                    <div class="tile-value">{{ tile.value }}</div>
                    <div class="tile-label">{{ tile.label }}</div>
                </div>
            </div>
        </div>

        <SectionPanel title="各供应商用量" :subtitle="`统计范围：${rangeLabel}`" flush>
            <template #icon>
                <el-icon>
                    <DataAnalysis />
                </el-icon>
            </template>
            <template #extra>
                <div class="flex items-center gap-2">
                    <el-radio-group v-model="days" size="small" @change="load">
                        <el-radio-button :value="1">今日</el-radio-button>
                        <el-radio-button :value="7">7 天</el-radio-button>
                        <el-radio-button :value="30">30 天</el-radio-button>
                    </el-radio-group>
                    <el-button size="small" :icon="Refresh" circle @click="load" />
                </div>
            </template>

            <el-table :data="stats?.providers ?? []" size="default" empty-text="该时间范围内没有调用记录">
                <el-table-column label="供应商" min-width="150">
                    <template #default="scope">
                        <div class="flex items-center gap-2">
                            <ProviderLogo :name="scope.row.provider" :src="metaOf(scope.row.provider)?.logoUrl"
                                :size="20" />
                            <span class="font-medium text-gray-700">
                                {{ metaOf(scope.row.provider)?.displayName || scope.row.provider }}
                            </span>
                        </div>
                    </template>
                </el-table-column>
                <el-table-column prop="total" label="请求" min-width="80" align="right" />
                <el-table-column label="成功率" min-width="100" align="right">
                    <template #default="scope">{{ rate(scope.row.succeeded, scope.row.total) }}</template>
                </el-table-column>
                <el-table-column label="缓存命中" min-width="100" align="right">
                    <template #default="scope">{{ rate(scope.row.cached, scope.row.total) }}</template>
                </el-table-column>
                <el-table-column prop="credits" label="额度" min-width="80" align="right" />
                <el-table-column label="花费" min-width="100" align="right">
                    <template #default="scope">
                        {{ scope.row.costMicroUsd ? formatCost(scope.row.costMicroUsd) : '—' }}
                    </template>
                </el-table-column>
                <el-table-column label="平均耗时" min-width="110" align="right">
                    <template #default="scope">{{ Math.round(scope.row.avgLatency) }} ms</template>
                </el-table-column>
            </el-table>
        </SectionPanel>

        <SectionPanel title="本月配额" subtitle="按自然月统计，与上面的时间范围无关">
            <template #icon>
                <el-icon>
                    <PieChart />
                </el-icon>
            </template>

            <el-empty v-if="!stats?.quotas?.length" description="暂无配额数据" :image-size="60" />
            <div v-else class="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-x-7 gap-y-5">
                <div v-for="quota in stats?.quotas ?? []" :key="quota.provider">
                    <div class="mb-2 flex items-center gap-2">
                        <ProviderLogo :name="quota.provider" :src="metaOf(quota.provider)?.logoUrl" :size="18" />
                        <span class="text-sm text-gray-700">
                            {{ metaOf(quota.provider)?.displayName || quota.provider }}
                        </span>
                        <span class="ml-auto text-xs text-gray-400 tabular-nums">
                            {{ quota.monthlyUsed }} / {{ quota.monthlyQuota || '不限' }}
                            <template v-if="quota.monthlyCostMicroUsd">
                                · {{ formatCost(quota.monthlyCostMicroUsd) }}
                            </template>
                        </span>
                    </div>
                    <el-progress :percentage="quotaPercentage(quota.monthlyUsed, quota.monthlyQuota)"
                        :stroke-width="8" :show-text="false"
                        :status="quotaPercentage(quota.monthlyUsed, quota.monthlyQuota) >= 90 ? 'exception' : undefined" />
                </div>
            </div>
        </SectionPanel>
    </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue';
import { Coin, DataAnalysis, Histogram, PieChart, Refresh, Timer } from '@element-plus/icons-vue';
import ProviderLogo from './ProviderLogo.vue';
import SectionPanel from './SectionPanel.vue';
import { formatCost, quotaPercentage } from './format';
import { getGatewayStats, type GatewayProvider, type GatewayStats } from '@/api/gateway';

const props = defineProps<{ providers: GatewayProvider[] }>();

const stats = ref<GatewayStats | null>(null);
const days = ref(1);
const loading = ref(false);

const rangeLabel = computed(() => (days.value === 1 ? '今日' : `近 ${days.value} 天`));

// 统计接口只回名字，展示名与 logo 得从供应商列表里取
function metaOf(name: string) {
    return props.providers.find((item) => item.name === name);
}

function rate(part: number, total: number) {
    if (!total) return '—';
    return `${((part / total) * 100).toFixed(1)}%`;
}

const tiles = computed(() => [
    {
        label: '请求数', value: String(stats.value?.total ?? 0),
        icon: Histogram, tint: '#eef4ff', color: '#3f8cff'
    },
    {
        label: '成功率', value: rate(stats.value?.succeeded ?? 0, stats.value?.total ?? 0),
        icon: Timer, tint: '#e9f9f0', color: '#16a34a'
    },
    {
        label: '缓存命中率', value: rate(stats.value?.cached ?? 0, stats.value?.total ?? 0),
        icon: DataAnalysis, tint: '#fff4e6', color: '#f59e0b'
    },
    {
        label: `消耗额度 · ${formatCost(stats.value?.costMicroUsd ?? 0)}`,
        value: String(stats.value?.credits ?? 0),
        icon: Coin, tint: '#f4f0ff', color: '#7c5cff'
    }
]);

async function load() {
    loading.value = true;
    try {
        stats.value = await getGatewayStats(days.value);
    } finally {
        loading.value = false;
    }
}

onMounted(load);
</script>

<style scoped>
.tile {
    display: flex;
    align-items: center;
    gap: 14px;
    padding: 16px 18px;
    background-color: #fff;
    border: 1px solid #edf0f3;
    border-radius: 14px;
    box-shadow: 0 1px 2px rgba(16, 24, 40, 0.04);
}

.tile-icon {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    width: 40px;
    height: 40px;
    border-radius: 12px;
    font-size: 19px;
    flex-shrink: 0;
}

.tile-value {
    font-size: 24px;
    font-weight: 600;
    line-height: 1.2;
    color: #1f2937;
    font-variant-numeric: tabular-nums;
}

.tile-label {
    margin-top: 2px;
    font-size: 12px;
    color: #98a2b3;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
}
</style>
