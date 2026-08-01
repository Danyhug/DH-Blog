<template>
    <div v-loading="loading">
        <el-card shadow="hover" class="mb-5">
            <template #header>
                <div class="flex items-center justify-between">
                    <span class="flex items-center">
                        <el-icon class="mr-2">
                            <DataAnalysis />
                        </el-icon> 调用概览
                    </span>
                    <div class="flex items-center gap-2">
                        <el-radio-group v-model="days" size="small" @change="load">
                            <el-radio-button :value="1">今日</el-radio-button>
                            <el-radio-button :value="7">近 7 天</el-radio-button>
                            <el-radio-button :value="30">近 30 天</el-radio-button>
                        </el-radio-group>
                        <el-button size="small" :icon="Refresh" @click="load">刷新</el-button>
                    </div>
                </div>
            </template>

            <el-row :gutter="16" class="mb-5">
                <el-col :span="6">
                    <el-statistic title="请求数" :value="stats?.total ?? 0" />
                </el-col>
                <el-col :span="6">
                    <el-statistic title="成功率" :value="successRate" suffix="%" :precision="1" />
                </el-col>
                <el-col :span="6">
                    <el-statistic title="缓存命中率" :value="cacheHitRate" suffix="%" :precision="1" />
                </el-col>
                <el-col :span="6">
                    <el-statistic title="消耗额度" :value="stats?.credits ?? 0" />
                    <div class="text-xs text-gray-400 mt-1">
                        按额计费花费 {{ formatCost(stats?.costMicroUsd ?? 0) }}
                    </div>
                </el-col>
            </el-row>

            <el-table :data="stats?.providers ?? []" size="small" empty-text="所选时间范围内没有调用记录">
                <el-table-column label="供应商" min-width="140">
                    <template #default="scope">
                        <div class="flex items-center gap-2">
                            <ProviderLogo :name="scope.row.provider" :src="metaOf(scope.row.provider)?.logoUrl"
                                :size="18" />
                            <span>{{ metaOf(scope.row.provider)?.displayName || scope.row.provider }}</span>
                        </div>
                    </template>
                </el-table-column>
                <el-table-column prop="total" label="请求数" min-width="90" />
                <el-table-column prop="succeeded" label="成功" min-width="90" />
                <el-table-column prop="cached" label="缓存命中" min-width="100" />
                <el-table-column prop="credits" label="额度" min-width="90" />
                <el-table-column label="花费" min-width="100">
                    <template #default="scope">
                        {{ scope.row.costMicroUsd ? formatCost(scope.row.costMicroUsd) : '-' }}
                    </template>
                </el-table-column>
                <el-table-column label="平均耗时" min-width="110">
                    <template #default="scope">{{ Math.round(scope.row.avgLatency) }} ms</template>
                </el-table-column>
            </el-table>
        </el-card>

        <el-card shadow="hover">
            <template #header>
                <span class="flex items-center">
                    <el-icon class="mr-2">
                        <PieChart />
                    </el-icon> 本月配额
                    <span class="ml-2 text-xs text-gray-400">按自然月统计，与上面的时间范围无关</span>
                </span>
            </template>

            <el-empty v-if="!stats?.quotas?.length" description="暂无配额数据" :image-size="60" />
            <div v-else class="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-x-8 gap-y-5">
                <div v-for="quota in stats?.quotas ?? []" :key="quota.provider">
                    <div class="mb-1 flex items-center gap-2 text-sm">
                        <ProviderLogo :name="quota.provider" :src="metaOf(quota.provider)?.logoUrl" :size="18" />
                        <span class="text-gray-700">
                            {{ metaOf(quota.provider)?.displayName || quota.provider }}
                        </span>
                        <span class="ml-auto text-xs text-gray-400">
                            {{ quota.monthlyUsed }} / {{ quota.monthlyQuota || '不限' }}
                            <template v-if="quota.monthlyCostMicroUsd">
                                · {{ formatCost(quota.monthlyCostMicroUsd) }}
                            </template>
                        </span>
                    </div>
                    <el-progress :percentage="quotaPercentage(quota.monthlyUsed, quota.monthlyQuota)"
                        :status="quotaPercentage(quota.monthlyUsed, quota.monthlyQuota) >= 90 ? 'exception' : undefined" />
                </div>
            </div>
        </el-card>
    </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue';
import { DataAnalysis, PieChart, Refresh } from '@element-plus/icons-vue';
import ProviderLogo from './ProviderLogo.vue';
import { formatCost, quotaPercentage } from './format';
import { getGatewayStats, type GatewayProvider, type GatewayStats } from '@/api/gateway';

const props = defineProps<{ providers: GatewayProvider[] }>();

const stats = ref<GatewayStats | null>(null);
const days = ref(1);
const loading = ref(false);

// 统计接口只回名字，展示名与 logo 得从供应商列表里取
function metaOf(name: string) {
    return props.providers.find((item) => item.name === name);
}

const successRate = computed(() => {
    if (!stats.value?.total) return 0;
    return (stats.value.succeeded / stats.value.total) * 100;
});

const cacheHitRate = computed(() => {
    if (!stats.value?.total) return 0;
    return (stats.value.cached / stats.value.total) * 100;
});

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
