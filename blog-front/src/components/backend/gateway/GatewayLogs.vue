<template>
    <el-card shadow="hover">
        <template #header>
            <div class="flex items-center justify-between">
                <span class="flex items-center">
                    <el-icon class="mr-2">
                        <Document />
                    </el-icon> 请求流水
                </span>
                <div class="flex gap-2">
                    <el-select v-model="provider" placeholder="全部供应商" clearable size="small" class="w-36"
                        @change="load(1)">
                        <el-option v-for="item in providers" :key="item.name"
                            :label="item.displayName || item.name" :value="item.name" />
                    </el-select>
                    <el-select v-model="status" placeholder="全部状态" clearable size="small" class="w-44"
                        @change="load(1)">
                        <el-option v-for="option in statusOptions" :key="option" :label="option" :value="option" />
                    </el-select>
                    <el-button size="small" :icon="Refresh" @click="load(page)">刷新</el-button>
                </div>
            </div>
        </template>

        <el-table v-loading="loading" :data="logs" size="small" empty-text="暂无记录">
            <el-table-column label="时间" min-width="160">
                <template #default="scope">{{ formatTime(scope.row.createdAt) }}</template>
            </el-table-column>
            <el-table-column label="供应商" min-width="110">
                <template #default="scope">
                    <div v-if="scope.row.provider" class="flex items-center gap-2">
                        <ProviderLogo :name="scope.row.provider" :src="metaOf(scope.row.provider)?.logoUrl"
                            :size="16" />
                        <span>{{ scope.row.provider }}</span>
                    </div>
                    <span v-else>-</span>
                </template>
            </el-table-column>
            <el-table-column prop="endpoint" label="接口" min-width="130" show-overflow-tooltip />
            <el-table-column prop="query" label="查询" min-width="180" show-overflow-tooltip />
            <el-table-column label="状态" min-width="150">
                <template #default="scope">
                    <el-tag :type="statusType(scope.row.status)" size="small">{{ scope.row.status }}</el-tag>
                </template>
            </el-table-column>
            <el-table-column prop="resultCount" label="结果" min-width="70" />
            <el-table-column label="缓存" min-width="70">
                <template #default="scope">{{ scope.row.cached ? '是' : '否' }}</template>
            </el-table-column>
            <el-table-column label="回退自" min-width="90">
                <template #default="scope">{{ scope.row.fallbackFrom || '-' }}</template>
            </el-table-column>
            <el-table-column label="花费" min-width="90">
                <template #default="scope">
                    {{ scope.row.costMicroUsd ? formatCost(scope.row.costMicroUsd) : '-' }}
                </template>
            </el-table-column>
            <el-table-column label="耗时" min-width="90">
                <template #default="scope">{{ scope.row.latencyMs }} ms</template>
            </el-table-column>
            <el-table-column prop="error" label="错误" min-width="180" show-overflow-tooltip />
        </el-table>

        <div class="mt-4 flex justify-end">
            <el-pagination layout="total, prev, pager, next" :total="total" :page-size="pageSize" :current-page="page"
                @current-change="load" />
        </div>
    </el-card>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue';
import { Document, Refresh } from '@element-plus/icons-vue';
import ProviderLogo from './ProviderLogo.vue';
import { formatCost, formatTime } from './format';
import { getGatewayLogs, type GatewayProvider, type GatewayRequestLog } from '@/api/gateway';

const props = defineProps<{ providers: GatewayProvider[] }>();

const logs = ref<GatewayRequestLog[]>([]);
const total = ref(0);
const page = ref(1);
const pageSize = 20;
const provider = ref('');
const status = ref('');
const loading = ref(false);

// 与后端 log_status 常量一一对应，改动时两边要一起改
const statusOptions = [
    'ok', 'provider_error', 'rate_limited', 'quota_exceeded',
    'invalid_request', 'provider_not_allowed', 'provider_not_found', 'no_provider_available'
];

function metaOf(name: string) {
    return props.providers.find((item) => item.name === name);
}

// 调用方写错参数和上游真的挂了不是一回事，颜色上分开，免得扫一眼全是红的
function statusType(value: string) {
    if (value === 'ok') return 'success';
    if (value === 'provider_error' || value === 'no_provider_available') return 'danger';
    return 'warning';
}

async function load(target = 1) {
    page.value = target;
    loading.value = true;
    try {
        const result = await getGatewayLogs({
            page: target,
            pageSize,
            provider: provider.value || undefined,
            status: status.value || undefined
        });
        logs.value = result.list ?? [];
        total.value = result.total ?? 0;
    } finally {
        loading.value = false;
    }
}

onMounted(() => load(1));
</script>
