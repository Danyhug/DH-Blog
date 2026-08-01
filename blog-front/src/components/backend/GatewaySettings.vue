<template>
    <div class="gateway-settings">
        <!-- 用量看板 -->
        <el-card shadow="hover" class="mb-5">
            <template #header>
                <div class="flex items-center justify-between">
                    <span class="flex items-center">
                        <el-icon class="mr-2">
                            <DataAnalysis />
                        </el-icon> 网关用量
                    </span>
                    <el-radio-group v-model="statsDays" size="small" @change="loadStats">
                        <el-radio-button :value="1">今日</el-radio-button>
                        <el-radio-button :value="7">近 7 天</el-radio-button>
                        <el-radio-button :value="30">近 30 天</el-radio-button>
                    </el-radio-group>
                </div>
            </template>

            <el-row :gutter="16" class="mb-4">
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

            <el-table :data="stats?.providers ?? []" size="small" empty-text="暂无调用记录">
                <el-table-column prop="provider" label="供应商" min-width="110" />
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

            <el-divider content-position="left">本月配额</el-divider>
            <div class="flex flex-wrap gap-6">
                <div v-for="quota in stats?.quotas ?? []" :key="quota.provider" class="min-w-[220px]">
                    <div class="mb-1 text-sm text-gray-600">
                        {{ quota.provider }}
                        <span class="ml-1 text-gray-400">
                            {{ quota.monthlyUsed }} / {{ quota.monthlyQuota || '不限' }}
                            <template v-if="quota.monthlyCostMicroUsd">
                                · {{ formatCost(quota.monthlyCostMicroUsd) }}
                            </template>
                        </span>
                    </div>
                    <el-progress :percentage="quotaPercentage(quota)"
                        :status="quotaPercentage(quota) >= 90 ? 'exception' : undefined" />
                </div>
            </div>
        </el-card>

        <!-- 调度方式 -->
        <el-card shadow="hover" class="mb-5">
            <template #header>
                <div class="flex items-center">
                    <span class="flex items-center">
                        <el-icon class="mr-2">
                            <Switch />
                        </el-icon> 调度方式
                    </span>
                </div>
            </template>

            <div class="text-xs text-gray-500 mb-4">
                仅在请求未指定 <code>provider</code>（即 <code>auto</code>）时生效。
                能力、健康与配额过滤始终优先——调度方式只决定通过过滤后的先后顺序。
            </div>

            <el-radio-group v-model="routingStrategy" class="w-full flex flex-col gap-3"
                @change="onStrategyChange">
                <el-radio v-for="option in strategies" :key="option.value" :value="option.value"
                    class="!mr-0 !h-auto !items-start w-full border border-gray-200 rounded-lg px-4 py-3 transition-colors"
                    :class="routingStrategy === option.value ? 'border-blue-400 bg-blue-50/50' : 'hover:border-gray-300'">
                    <div class="flex flex-col gap-1 whitespace-normal">
                        <span class="flex items-center gap-2 font-medium">
                            {{ option.label }}
                            <el-tag v-if="!option.implemented" type="warning" size="small" effect="plain">
                                开发中
                            </el-tag>
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

        <!-- 供应商配置 -->
        <el-card shadow="hover" class="mb-5">
            <template #header>
                <div class="flex items-center justify-between">
                    <span class="flex items-center">
                        <el-icon class="mr-2">
                            <Connection />
                        </el-icon> 搜索供应商
                    </span>
                    <el-button size="small" :icon="Refresh" @click="loadProviders">刷新</el-button>
                </div>
            </template>

            <el-alert type="info" :closable="false" class="mb-4" show-icon>
                <template #title>
                    网关 Key 只应用于服务端或 agent 侧。本站 CORS 允许任意来源，Key 一旦写进浏览器代码即等同公开。
                </template>
            </el-alert>

            <el-descriptions :column="1" border size="small" class="mb-4">
                <el-descriptions-item label="统一接口">
                    <code>POST /api/gateway/v1/search</code> · <code>GET /api/gateway/v1/search?q=</code>
                    <span class="text-gray-400 ml-2">网关自行选路，响应为统一格式</span>
                </el-descriptions-item>
                <el-descriptions-item label="原生透传">
                    <code>POST /api/gateway/v1/tavily/search</code> ·
                    <code>GET /api/gateway/v1/brave/web/search</code> ·
                    <code>POST /api/gateway/v1/exa/search</code>
                    <span class="text-gray-400 ml-2">请求与响应保持上游原样，现有 SDK 改 base_url 即可，不做跨供应商回退</span>
                </el-descriptions-item>
            </el-descriptions>

            <el-collapse v-model="expandedProviders">
                <el-collapse-item v-for="provider in providers" :key="provider.name" :name="provider.name">
                    <template #title>
                        <div class="flex items-center gap-3 w-full pr-4">
                            <img v-if="provider.logoUrl && !brokenLogos.has(provider.name)" :src="provider.logoUrl"
                                :alt="provider.displayName" class="w-6 h-6 rounded object-contain shrink-0"
                                loading="lazy" @error="brokenLogos.add(provider.name)" />
                            <span v-else
                                class="w-6 h-6 rounded shrink-0 flex items-center justify-center text-xs font-semibold text-white"
                                :style="{ backgroundColor: fallbackColor(provider.name) }">
                                {{ provider.name.charAt(0).toUpperCase() }}
                            </span>

                            <span class="font-medium">{{ provider.displayName || provider.name }}</span>
                            <el-tag :type="provider.enabled ? 'success' : 'info'" size="small">
                                {{ provider.enabled ? '已启用' : '未启用' }}
                            </el-tag>
                            <el-tag v-if="provider.health !== 'closed'" type="warning" size="small">
                                {{ provider.health === 'open' ? '已熔断' : '半开探测中' }}
                            </el-tag>
                            <span class="text-xs text-gray-400">
                                本月 {{ provider.monthlyUsed }} / {{ provider.monthlyQuota || '不限' }}
                                <template v-if="provider.monthlyCostMicroUsd">
                                    · {{ formatCost(provider.monthlyCostMicroUsd) }}
                                </template>
                            </span>

                            <span class="ml-auto flex items-center gap-3 text-xs" @click.stop>
                                <a :href="provider.homeUrl" target="_blank" rel="noopener noreferrer"
                                    class="text-blue-500 hover:underline">官网</a>
                                <a :href="provider.docsUrl" target="_blank" rel="noopener noreferrer"
                                    class="text-blue-500 hover:underline">文档</a>
                                <a :href="provider.consoleUrl" target="_blank" rel="noopener noreferrer"
                                    class="text-blue-500 hover:underline">取密钥</a>
                            </span>
                        </div>
                    </template>

                    <el-form label-position="top" size="default">
                        <div class="mb-3 text-xs text-gray-500">
                            <el-icon class="align-middle mr-1">
                                <InfoFilled />
                            </el-icon>{{ provider.billing }}
                        </div>
                        <el-row :gutter="16">
                            <el-col :span="6">
                                <el-form-item label="启用">
                                    <el-switch v-model="provider.enabled" />
                                </el-form-item>
                            </el-col>
                            <el-col :span="18">
                                <el-form-item :label="`API 秘钥（当前：${provider.apiKeyPresent ? provider.apiKeyMasked : '未配置'}）`">
                                    <el-input v-model="drafts[provider.name].apiKey" show-password :prefix-icon="Key"
                                        placeholder="留空表示不修改已保存的秘钥" />
                                </el-form-item>
                            </el-col>
                        </el-row>

                        <el-row :gutter="16">
                            <el-col :span="12">
                                <el-form-item label="接口地址（留空使用官方地址）">
                                    <el-input v-model="provider.baseUrl" placeholder="https://api.tavily.com" clearable />
                                </el-form-item>
                            </el-col>
                            <el-col :span="6">
                                <el-form-item label="优先级（越小越优先）">
                                    <el-input-number v-model="provider.priority" :min="0" :max="999" class="w-full" />
                                </el-form-item>
                            </el-col>
                            <el-col :span="6">
                                <el-form-item label="权重">
                                    <el-input-number v-model="provider.weight" :min="0" :max="100" class="w-full" />
                                </el-form-item>
                            </el-col>
                        </el-row>

                        <el-row :gutter="16">
                            <el-col :span="8">
                                <el-form-item label="出站限速（次/秒，0 表示不限）">
                                    <el-input-number v-model="provider.rps" :min="0" :max="100" :step="0.5"
                                        class="w-full" />
                                </el-form-item>
                            </el-col>
                            <el-col :span="8">
                                <el-form-item label="月配额（0 表示不限）">
                                    <el-input-number v-model="provider.monthlyQuota" :min="0" :max="1000000"
                                        class="w-full" />
                                </el-form-item>
                            </el-col>
                            <el-col :span="8">
                                <el-form-item label="附加参数（JSON）">
                                    <el-input v-model="provider.extra" placeholder='{"search_depth":"basic"}' />
                                </el-form-item>
                            </el-col>
                        </el-row>

                        <div class="text-right">
                            <el-button :loading="testing === provider.name" @click="onTestProvider(provider)">
                                连通性测试
                            </el-button>
                            <el-button type="primary" :loading="saving === provider.name"
                                @click="onSaveProvider(provider)">
                                保存
                            </el-button>
                        </div>
                    </el-form>
                </el-collapse-item>
            </el-collapse>
        </el-card>

        <!-- API Key 管理 -->
        <el-card shadow="hover" class="mb-5">
            <template #header>
                <div class="flex items-center justify-between">
                    <span class="flex items-center">
                        <el-icon class="mr-2">
                            <Key />
                        </el-icon> 网关 API Key
                    </span>
                    <el-button type="primary" size="small" @click="openCreateDialog">新建 Key</el-button>
                </div>
            </template>

            <el-table :data="apiKeys" size="default" empty-text="尚未签发 Key">
                <el-table-column prop="name" label="名称" min-width="140" />
                <el-table-column prop="keyPrefix" label="前缀" min-width="180" />
                <el-table-column label="状态" min-width="90">
                    <template #default="scope">
                        <el-switch :model-value="scope.row.enabled"
                            @change="(value: any) => onToggleKey(scope.row, Boolean(value))" />
                    </template>
                </el-table-column>
                <el-table-column label="供应商限制" min-width="130">
                    <template #default="scope">{{ scope.row.allowedProviders || '全部' }}</template>
                </el-table-column>
                <el-table-column label="限速" min-width="110">
                    <template #default="scope">
                        {{ scope.row.rateLimitPerMin ? `${scope.row.rateLimitPerMin} 次/分` : '不限' }}
                    </template>
                </el-table-column>
                <el-table-column label="本月用量" min-width="120">
                    <template #default="scope">
                        {{ scope.row.monthlyUsed }} / {{ scope.row.monthlyQuota || '不限' }}
                    </template>
                </el-table-column>
                <el-table-column label="最后使用" min-width="170">
                    <template #default="scope">{{ formatTime(scope.row.lastUsedAt) }}</template>
                </el-table-column>
                <el-table-column label="操作" width="100">
                    <template #default="scope">
                        <el-button size="small" type="danger" link @click="onDeleteKey(scope.row)">吊销</el-button>
                    </template>
                </el-table-column>
            </el-table>
        </el-card>

        <!-- 请求流水 -->
        <el-card shadow="hover">
            <template #header>
                <div class="flex items-center justify-between">
                    <span class="flex items-center">
                        <el-icon class="mr-2">
                            <Document />
                        </el-icon> 请求流水
                    </span>
                    <div class="flex gap-2">
                        <el-select v-model="logProvider" placeholder="全部供应商" clearable size="small" class="w-36"
                            @change="loadLogs(1)">
                            <el-option v-for="provider in providers" :key="provider.name" :label="provider.name"
                                :value="provider.name" />
                        </el-select>
                        <el-select v-model="logStatus" placeholder="全部状态" clearable size="small" class="w-40"
                            @change="loadLogs(1)">
                            <el-option v-for="status in statusOptions" :key="status" :label="status" :value="status" />
                        </el-select>
                    </div>
                </div>
            </template>

            <el-table :data="logs" size="small" empty-text="暂无记录">
                <el-table-column label="时间" min-width="160">
                    <template #default="scope">{{ formatTime(scope.row.createdAt) }}</template>
                </el-table-column>
                <el-table-column prop="provider" label="供应商" min-width="90" />
                <el-table-column prop="endpoint" label="接口" min-width="130" show-overflow-tooltip />
                <el-table-column prop="query" label="查询" min-width="180" show-overflow-tooltip />
                <el-table-column label="状态" min-width="130">
                    <template #default="scope">
                        <el-tag :type="scope.row.status === 'ok' ? 'success' : 'danger'" size="small">
                            {{ scope.row.status }}
                        </el-tag>
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
                <el-pagination layout="total, prev, pager, next" :total="logTotal" :page-size="logPageSize"
                    :current-page="logPage" @current-change="loadLogs" />
            </div>
        </el-card>

        <!-- 新建 Key -->
        <el-dialog v-model="createDialogVisible" title="新建网关 API Key" width="480px">
            <el-form :model="createForm" label-width="110px" size="default">
                <el-form-item label="名称" required>
                    <el-input v-model="createForm.name" placeholder="例如：我的写作 agent" clearable />
                </el-form-item>
                <el-form-item label="允许的供应商">
                    <el-select v-model="createAllowed" multiple placeholder="留空表示全部" class="w-full">
                        <el-option v-for="provider in providers" :key="provider.name" :label="provider.name"
                            :value="provider.name" />
                    </el-select>
                </el-form-item>
                <el-form-item label="限速（次/分）">
                    <el-input-number v-model="createForm.rateLimitPerMin" :min="0" :max="10000" class="w-full" />
                </el-form-item>
                <el-form-item label="月配额">
                    <el-input-number v-model="createForm.monthlyQuota" :min="0" :max="1000000" class="w-full" />
                </el-form-item>
                <el-form-item label="有效天数">
                    <el-input-number v-model="createForm.expireDays" :min="0" :max="3650" class="w-full" />
                    <div class="text-xs text-gray-400 mt-1">0 表示永不过期</div>
                </el-form-item>
                <el-form-item label="备注">
                    <el-input v-model="createForm.note" clearable />
                </el-form-item>
            </el-form>
            <template #footer>
                <el-button @click="createDialogVisible = false">取消</el-button>
                <el-button type="primary" :loading="creating" @click="onCreateKey">创建</el-button>
            </template>
        </el-dialog>

        <!-- 明文只展示一次 -->
        <el-dialog v-model="secretDialogVisible" title="请立即保存这个 Key" width="520px" :close-on-click-modal="false">
            <el-alert type="warning" :closable="false" show-icon class="mb-4"
                title="明文只显示这一次，关闭后无法再次查看。" />
            <el-input :model-value="createdSecret" readonly />
            <template #footer>
                <el-button type="primary" @click="copySecret">复制并关闭</el-button>
            </template>
        </el-dialog>
    </div>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue';
import { ElMessageBox } from 'element-plus';
import { Connection, DataAnalysis, Document, InfoFilled, Key, Refresh, Switch } from '@element-plus/icons-vue';
import { notify } from '@/utils/notification';
import {
    createGatewayApiKey,
    getGatewaySettings,
    updateGatewaySettings,
    type RoutingStrategy,
    type StrategyOption,
    deleteGatewayApiKey,
    getGatewayApiKeys,
    getGatewayLogs,
    getGatewayProviders,
    getGatewayStats,
    testGatewayProvider,
    updateGatewayApiKey,
    updateGatewayProvider,
    type CreateGatewayApiKeyPayload,
    type GatewayApiKey,
    type GatewayProvider,
    type GatewayQuota,
    type GatewayRequestLog,
    type GatewayStats
} from '@/api/gateway';

const routingStrategy = ref<RoutingStrategy>('balanced');
const strategies = ref<StrategyOption[]>([]);

const providers = ref<GatewayProvider[]>([]);
const expandedProviders = ref<string[]>([]);
// 供应商 logo 取自各家官网，加载失败时降级为首字母色块，避免外链挂掉后出现破图
const brokenLogos = reactive(new Set<string>());
// 密钥草稿与列表分开存放：列表里的是脱敏值，直接绑定会把脱敏串当成新密钥存回去
const drafts = reactive<Record<string, { apiKey: string }>>({});
const saving = ref('');
const testing = ref('');

const apiKeys = ref<GatewayApiKey[]>([]);
const createDialogVisible = ref(false);
const creating = ref(false);
const createAllowed = ref<string[]>([]);
const createForm = ref<CreateGatewayApiKeyPayload>({
    name: '',
    rateLimitPerMin: 60,
    monthlyQuota: 0,
    expireDays: 0,
    note: ''
});
const secretDialogVisible = ref(false);
const createdSecret = ref('');

const stats = ref<GatewayStats | null>(null);
const statsDays = ref(1);

const logs = ref<GatewayRequestLog[]>([]);
const logTotal = ref(0);
const logPage = ref(1);
const logPageSize = 20;
const logProvider = ref('');
const logStatus = ref('');
const statusOptions = [
    'ok', 'provider_error', 'rate_limited', 'quota_exceeded',
    'invalid_request', 'provider_not_allowed', 'provider_not_found', 'no_provider_available'
];

const activeStrategy = computed(() => strategies.value.find((item) => item.value === routingStrategy.value));
const activeStrategyLabel = computed(() => activeStrategy.value?.label ?? routingStrategy.value);
// 选了尚未接入的方式时必须显式提示，否则界面等于谎称路由已改变
const activeStrategyPending = computed(() => activeStrategy.value?.implemented === false);

const successRate = computed(() => {
    if (!stats.value?.total) return 0;
    return (stats.value.succeeded / stats.value.total) * 100;
});

const cacheHitRate = computed(() => {
    if (!stats.value?.total) return 0;
    return (stats.value.cached / stats.value.total) * 100;
});

// fallbackColor 由名称派生出稳定的色相，保证同一供应商每次渲染颜色一致
function fallbackColor(name: string) {
    let hash = 0;
    for (let i = 0; i < name.length; i++) hash = (hash * 31 + name.charCodeAt(i)) % 360;
    return `hsl(${hash}, 55%, 48%)`;
}

// formatCost 把微美元转成可读金额；小额也保留有效数字，否则一律显示为 $0.00
function formatCost(microUsd: number) {
    if (!microUsd) return '$0';
    const usd = microUsd / 1_000_000;
    if (usd < 0.01) return `$${usd.toFixed(4)}`;
    return `$${usd.toFixed(2)}`;
}

function quotaPercentage(quota: GatewayQuota) {
    if (!quota.monthlyQuota) return 0;
    return Math.min(100, Math.round((quota.monthlyUsed / quota.monthlyQuota) * 100));
}

function formatTime(value: string | null) {
    if (!value) return '-';
    const parsed = new Date(value);
    return Number.isNaN(parsed.getTime()) ? value : parsed.toLocaleString('zh-CN');
}

async function loadSettings() {
    const settings = await getGatewaySettings();
    routingStrategy.value = settings.routingStrategy;
    strategies.value = settings.strategies;
}

async function onStrategyChange(value: any) {
    const next = value as RoutingStrategy;
    try {
        await updateGatewaySettings({ routingStrategy: next });
    } catch (error) {
        await loadSettings(); // 保存失败时回到服务端的真实取值
        throw error;
    }
    const option = strategies.value.find((item) => item.value === next);
    if (option && !option.implemented) {
        notify.warning(`已保存，但「${option.label}」尚未接入，实际仍按负载均衡调度`);
        return;
    }
    notify.success(`调度方式已切换为「${option?.label ?? next}」`);
}

async function loadProviders() {
    providers.value = await getGatewayProviders();
    providers.value.forEach((provider) => {
        if (!drafts[provider.name]) drafts[provider.name] = { apiKey: '' };
    });
    if (!expandedProviders.value.length && providers.value.length) {
        expandedProviders.value = [providers.value[0].name];
    }
}

async function onSaveProvider(provider: GatewayProvider) {
    saving.value = provider.name;
    try {
        await updateGatewayProvider(provider.name, {
            enabled: provider.enabled,
            // 留空表示不修改，后端也会忽略空串
            apiKey: drafts[provider.name].apiKey,
            baseUrl: provider.baseUrl,
            priority: provider.priority,
            weight: provider.weight,
            rps: provider.rps,
            monthlyQuota: provider.monthlyQuota,
            extra: provider.extra
        });
        drafts[provider.name].apiKey = '';
        notify.success(`${provider.displayName || provider.name} 配置已保存`);
        await loadProviders();
    } finally {
        saving.value = '';
    }
}

async function onTestProvider(provider: GatewayProvider) {
    testing.value = provider.name;
    try {
        const result = await testGatewayProvider(provider.name);
        if (result.ok) {
            notify.success(`连通正常，耗时 ${result.latencyMs}ms，返回 ${result.resultCount} 条结果`);
        } else {
            notify.error(`连通失败：${result.error}`);
        }
    } finally {
        testing.value = '';
    }
}

async function loadApiKeys() {
    apiKeys.value = await getGatewayApiKeys();
}

function openCreateDialog() {
    createForm.value = { name: '', rateLimitPerMin: 60, monthlyQuota: 0, expireDays: 0, note: '' };
    createAllowed.value = [];
    createDialogVisible.value = true;
}

async function onCreateKey() {
    if (!createForm.value.name.trim()) {
        notify.warning('请填写名称');
        return;
    }
    creating.value = true;
    try {
        const created = await createGatewayApiKey({
            ...createForm.value,
            allowedProviders: createAllowed.value.join(',')
        });
        createDialogVisible.value = false;
        createdSecret.value = created.apiKey;
        secretDialogVisible.value = true;
        await loadApiKeys();
    } finally {
        creating.value = false;
    }
}

async function copySecret() {
    try {
        await navigator.clipboard.writeText(createdSecret.value);
        notify.success('已复制到剪贴板');
    } catch {
        notify.warning('复制失败，请手动选中复制');
        return;
    }
    secretDialogVisible.value = false;
}

async function onToggleKey(key: GatewayApiKey, enabled: boolean) {
    await updateGatewayApiKey(key.id, { enabled });
    notify.success(enabled ? 'Key 已启用' : 'Key 已停用');
    await loadApiKeys();
}

async function onDeleteKey(key: GatewayApiKey) {
    try {
        await ElMessageBox.confirm(`确定要吊销「${key.name}」吗？使用该 Key 的 agent 将立刻失效。`, '提示', { type: 'warning' });
    } catch {
        return;
    }
    await deleteGatewayApiKey(key.id);
    notify.success('已吊销');
    await loadApiKeys();
}

async function loadStats() {
    stats.value = await getGatewayStats(statsDays.value);
}

async function loadLogs(page = 1) {
    logPage.value = page;
    const result = await getGatewayLogs({
        page,
        pageSize: logPageSize,
        provider: logProvider.value || undefined,
        status: logStatus.value || undefined
    });
    logs.value = result.list ?? [];
    logTotal.value = result.total ?? 0;
}

onMounted(async () => {
    await Promise.all([loadSettings(), loadProviders()]);
    await Promise.all([loadApiKeys(), loadStats(), loadLogs(1)]);
});
</script>
