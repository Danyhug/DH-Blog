<template>
    <div>
        <SectionPanel title="搜索供应商" subtitle="每家可以配多把密钥，网关轮换使用；被上游拒掉的密钥会自动停止调度">
            <template #icon>
                <el-icon>
                    <Connection />
                </el-icon>
            </template>
            <template #extra>
                <el-button size="small" :icon="Odometer" :loading="syncing" @click="onSyncUsage">同步上游用量</el-button>
                <el-button size="small" :icon="Refresh" @click="emit('refresh')">刷新</el-button>
            </template>

            <el-empty v-if="!providers.length" description="暂无供应商" :image-size="60" />
            <div v-else class="grid grid-cols-1 lg:grid-cols-2 2xl:grid-cols-3 gap-4">
                <article v-for="provider in providers" :key="provider.name" class="card"
                    :class="{ 'card--off': !provider.enabled }">
                    <div class="flex items-center gap-3">
                        <ProviderLogo :name="provider.name" :size="34" />
                        <div class="min-w-0 flex-1">
                            <div class="flex items-center gap-2">
                                <span class="font-medium text-gray-800 truncate">
                                    {{ provider.displayName || provider.name }}
                                </span>
                                <el-tag v-if="provider.health !== 'closed'" size="small" type="warning" effect="plain">
                                    {{ provider.health === 'open' ? '已熔断' : '探测中' }}
                                </el-tag>
                            </div>
                            <div class="mt-0.5 text-xs" :class="keyHintClass(provider)">{{ keyHint(provider) }}</div>
                        </div>
                        <el-switch :model-value="provider.enabled" :loading="toggling === provider.name"
                            @change="(value: any) => onToggle(provider, Boolean(value))" />
                    </div>

                    <div class="mt-4">
                        <div class="flex items-center justify-between text-xs text-gray-400 mb-1.5">
                            <span>本月用量<span class="text-gray-300">（本地计数）</span></span>
                            <span class="tabular-nums">{{ budget(provider).text }}</span>
                        </div>
                        <el-progress :percentage="budget(provider).percent" :stroke-width="6" :show-text="false"
                            :status="budget(provider).percent >= 90 ? 'exception' : undefined" />

                        <!-- 上游口径单独一行：本地只数经过网关的请求，两个数字对不上是正常的 -->
                        <div class="mt-2 text-xs text-gray-400 flex items-center justify-between gap-2">
                            <template v-if="upstream(provider)">
                                <span class="truncate">
                                    上游口径<span v-if="upstream(provider)!.tightest" class="text-gray-300">（余量最紧的一把）</span>
                                </span>
                                <span class="tabular-nums shrink-0" :class="upstream(provider)!.warning ? 'text-orange-500' : ''">
                                    {{ upstream(provider)!.used }} / {{ upstream(provider)!.limit || '不限' }}
                                    {{ usageUnitLabel(upstream(provider)!.unit) }}
                                </span>
                            </template>
                            <span v-else-if="!provider.supportsUsageSync" class="text-gray-300">
                                该供应商不提供用量接口，只能按本地计数
                            </span>
                            <span v-else class="text-gray-300">上游用量尚未同步</span>
                        </div>
                    </div>

                    <div class="mt-4 flex items-center gap-3 text-xs">
                        <a :href="provider.consoleUrl" target="_blank" rel="noopener noreferrer" class="link">
                            去 {{ provider.displayName || provider.name }} 控制台申请密钥
                        </a>
                        <a :href="provider.docsUrl" target="_blank" rel="noopener noreferrer" class="link">文档</a>
                        <el-button class="ml-auto" size="small" type="primary" plain :icon="Setting"
                            @click="openDrawer(provider)">
                            配置
                        </el-button>
                    </div>
                </article>
            </div>
        </SectionPanel>

        <el-drawer v-model="drawerVisible" :title="`配置 ${active?.displayName || active?.name || ''}`" size="600px">
            <div v-if="active" class="drawer">
                <p class="hint">
                    <el-icon class="align-middle mr-1">
                        <InfoFilled />
                    </el-icon>{{ active.billing }}
                    <br />
                    <template v-if="active.supportsUsageSync">
                        每 60 分钟从上游同步一次真实用量；上游报额度用尽时这把密钥会自动停止调度。
                    </template>
                    <template v-else>
                        该供应商不提供用量接口，只能按网关本地计数——在别处用过同一把密钥这里看不到。
                    </template>
                </p>

                <h4 class="group-title">
                    密钥
                    <span class="group-sub">多把密钥按顺序轮换；上游报鉴权失败或配额耗尽时自动停止调度</span>
                </h4>

                <div v-if="!active.keys.length" class="empty-keys">还没有密钥，先在下面添加一把</div>
                <div v-for="item in active.keys" :key="item.id" class="key-row">
                    <span class="dot" :class="item.inRotation ? 'dot--on' : 'dot--off'" />
                    <div class="min-w-0 flex-1">
                        <div class="flex items-center gap-2">
                            <span class="text-sm text-gray-700 truncate">{{ item.label || '未命名' }}</span>
                            <code class="masked">{{ item.masked }}</code>
                            <el-tag v-if="item.status !== 'active'" size="small" type="danger" effect="plain">
                                {{ statusLabel(item.status) }}
                            </el-tag>
                            <el-tag v-else-if="!item.enabled" size="small" type="info" effect="plain">已停用</el-tag>
                        </div>
                        <div class="mt-0.5 text-xs text-gray-400 truncate">
                            <template v-if="item.lastError">停用原因：{{ item.lastError }}</template>
                            <template v-else-if="item.lastUsedAt">最后使用 {{ formatTime(item.lastUsedAt) }}</template>
                            <template v-else>尚未使用过</template>
                        </div>
                        <!-- 上游报的用量：说清单位、周期和是否与其它密钥共享，避免和本地计数混为一谈 -->
                        <div v-if="item.upstreamSyncedAt" class="mt-1 text-xs text-gray-400 truncate">
                            上游：{{ item.upstreamUsed }} / {{ item.upstreamLimit || '不限' }}
                            {{ usageUnitLabel(item.upstreamUnit) }}
                            <span class="text-gray-300">
                                · {{ usageScopeLabel(item.upstreamScope) }}
                                <template v-if="item.upstreamWindow">· {{ item.upstreamWindow }}</template>
                                · {{ formatSince(item.upstreamSyncedAt) }}
                            </span>
                        </div>
                        <div v-if="item.upstreamError" class="mt-1 text-xs text-orange-500 truncate">
                            用量同步失败：{{ item.upstreamError }}
                        </div>
                    </div>
                    <div class="flex items-center gap-1 shrink-0">
                        <el-button size="small" link :loading="testing === `key-${item.id}`"
                            @click="onTestStored(item)">测试</el-button>
                        <el-button v-if="item.status !== 'active'" size="small" link type="success"
                            @click="onPatchKey(item, { revive: true, enabled: true })">恢复</el-button>
                        <el-button size="small" link @click="onPatchKey(item, { enabled: !item.enabled })">
                            {{ item.enabled ? '停用' : '启用' }}
                        </el-button>
                        <el-button size="small" link type="danger" @click="onDeleteKey(item)">删除</el-button>
                    </div>
                </div>

                <div class="add-key">
                    <div class="flex gap-2">
                        <el-input v-model="newKey.label" placeholder="标签，例如「主账号」" class="w-40" clearable />
                        <el-input v-model="newKey.apiKey" placeholder="粘贴上游密钥" show-password clearable />
                    </div>
                    <div class="mt-2 flex justify-end gap-2">
                        <el-button size="small" :loading="testing === 'draft'" :disabled="!newKey.apiKey.trim()"
                            @click="onTestDraft">
                            先测一下
                        </el-button>
                        <el-button size="small" type="primary" :loading="adding" :disabled="!newKey.apiKey.trim()"
                            @click="onAddKey">
                            添加
                        </el-button>
                    </div>
                </div>

                <h4 class="group-title">
                    参数
                    <span class="group-sub">改动需要点保存才生效</span>
                </h4>
                <el-form label-position="top" size="default">
                    <el-form-item label="接口地址（留空使用官方地址）">
                        <el-input v-model="form.baseUrl" placeholder="https://api.tavily.com" clearable />
                    </el-form-item>
                    <el-row :gutter="14">
                        <el-col :span="12">
                            <el-form-item label="优先级（越小越优先）">
                                <el-input-number v-model="form.priority" :min="0" :max="999" class="w-full" />
                            </el-form-item>
                        </el-col>
                        <el-col :span="12">
                            <el-form-item label="权重">
                                <el-input-number v-model="form.weight" :min="0" :max="100" class="w-full" />
                            </el-form-item>
                        </el-col>
                    </el-row>
                    <el-row :gutter="14">
                        <el-col :span="12">
                            <el-form-item label="出站限速（次/秒，0 不限）">
                                <el-input-number v-model="form.rps" :min="0" :max="100" :step="0.5" class="w-full" />
                            </el-form-item>
                        </el-col>
                        <el-col :span="12">
                            <el-form-item label="月配额（次数，0 不限）">
                                <el-input-number v-model="form.monthlyQuota" :min="0" :max="1000000" class="w-full" />
                            </el-form-item>
                        </el-col>
                    </el-row>
                    <el-form-item>
                        <template #label>
                            月费用上限（美元，0 不限）
                            <span class="label-hint">按金额计费的供应商用这个，次数上限说不清预算</span>
                        </template>
                        <el-input-number v-model="form.monthlyCostLimitUsd" :min="0" :max="10000" :step="1"
                            :precision="2" class="w-full" />
                    </el-form-item>
                    <el-form-item label="附加参数（JSON）">
                        <el-input v-model="form.extra" placeholder='{"search_depth":"basic"}' />
                    </el-form-item>
                </el-form>

                <h4 class="group-title">
                    本月用量校准
                    <span class="group-sub">
                        本地计数只看得到网关自己发出的请求，和官网账单会有出入；按官网数字改这里即可，是覆盖不是累加
                    </span>
                </h4>
                <p v-if="active.upstreamCostMicroUsd !== null" class="hint">
                    <el-icon class="align-middle mr-1">
                        <InfoFilled />
                    </el-icon>
                    当前选路用的是上游报的花费 {{ formatUsd(active.upstreamCostMicroUsd) }}，本地数字仅供参考。
                </p>
                <el-form label-position="top">
                    <el-row :gutter="14">
                        <el-col :span="12">
                            <el-form-item label="本月调用次数">
                                <el-input-number v-model="usageForm.count" :min="0" :max="10000000" class="w-full" />
                            </el-form-item>
                        </el-col>
                        <el-col :span="12">
                            <el-form-item label="本月花费（美元）">
                                <el-input-number v-model="usageForm.costUsd" :min="0" :max="100000" :step="0.01"
                                    :precision="2" class="w-full" />
                            </el-form-item>
                        </el-col>
                    </el-row>
                    <el-button size="small" type="primary" plain :loading="savingUsage" :disabled="!usageDirty"
                        @click="onSaveUsage">
                        保存用量
                    </el-button>
                </el-form>

                <template v-if="supportsUsageCredential">
                    <h4 class="group-title">
                        上游用量接口
                        <span class="group-sub">
                            Exa 的花费要用团队 service key 查，和搜索密钥不是一把；填齐后每小时同步一次，选路改用上游数字
                        </span>
                    </h4>
                    <el-form label-position="top">
                        <el-form-item>
                            <template #label>
                                Service Key
                                <span class="label-hint">
                                    留空表示不修改；已保存的值只显示掩码
                                    <template v-if="active.usageServiceKeyMasked">（当前 {{ active.usageServiceKeyMasked }}）</template>
                                </span>
                            </template>
                            <el-input v-model="usageCredential.serviceKey" placeholder="在 Exa 控制台创建 team service key"
                                clearable />
                        </el-form-item>
                        <el-form-item label="搜索密钥的 UUID">
                            <el-input v-model="usageCredential.keyId" placeholder="550e8400-e29b-41d4-a716-446655440000"
                                clearable />
                        </el-form-item>
                        <div class="flex gap-2">
                            <el-button size="small" type="primary" plain :loading="savingCredential"
                                @click="onSaveUsageCredential">
                                保存
                            </el-button>
                            <el-button v-if="active.usageServiceKeyMasked" size="small" plain
                                :loading="savingCredential" @click="onClearUsageCredential">
                                清除 Service Key
                            </el-button>
                        </div>
                    </el-form>
                </template>
            </div>

            <template #footer>
                <span v-if="dirty" class="mr-auto text-xs text-orange-500">参数有未保存的改动</span>
                <el-button @click="drawerVisible = false">关闭</el-button>
                <el-button type="primary" :loading="saving" :disabled="!dirty" @click="onSaveParams">保存参数</el-button>
            </template>
        </el-drawer>
    </div>
</template>

<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue';
import { ElMessageBox } from 'element-plus';
import { Connection, InfoFilled, Odometer, Refresh, Setting } from '@element-plus/icons-vue';
import { notify } from '@/utils/notification';
import ProviderLogo from './ProviderLogo.vue';
import SectionPanel from './SectionPanel.vue';
import { budgetView as budget, formatSince, formatTime, usageScopeLabel, usageUnitLabel } from './format';
import {
    createGatewayProviderKey,
    deleteGatewayProviderKey,
    syncGatewayUsage,
    testGatewayProvider,
    updateGatewayProvider,
    updateGatewayProviderKey,
    updateGatewayProviderUsage,
    type GatewayProvider,
    type GatewayProviderKey
} from '@/api/gateway';

const props = defineProps<{ providers: GatewayProvider[] }>();
const emit = defineEmits<{ refresh: [] }>();

const drawerVisible = ref(false);
const activeName = ref('');
const toggling = ref('');
const saving = ref(false);
const adding = ref(false);
const testing = ref('');
const syncing = ref(false);

const newKey = reactive({ label: '', apiKey: '' });
// 费用上限在表单里用美元，存取时与后端的微美元互转：让人填 10000000 是不合理的
const form = reactive({
    baseUrl: '', priority: 100, weight: 1, rps: 1,
    monthlyQuota: 0, monthlyCostLimitUsd: 0, extra: ''
});
// 用量校准与上游凭据各自独立保存，避免和参数表单的 dirty 判断纠缠在一起
const usageForm = reactive({ count: 0, costUsd: 0 });
const usageCredential = reactive({ serviceKey: '', keyId: '' });
const savingUsage = ref(false);
const savingCredential = ref(false);

const MICRO_PER_USD = 1_000_000;

// 抽屉始终跟着列表里的最新数据走：加完密钥父组件刷新后，这里立刻能看到
const active = computed(() => props.providers.find((item) => item.name === activeName.value));

const dirty = computed(() => {
    const provider = active.value;
    if (!provider) return false;
    return form.baseUrl !== provider.baseUrl
        || form.priority !== provider.priority
        || form.weight !== provider.weight
        || form.rps !== provider.rps
        || form.monthlyQuota !== provider.monthlyQuota
        || costLimitMicro() !== provider.monthlyCostLimitMicroUsd
        || form.extra !== provider.extra;
});

// 只有 Exa 提供按密钥查花费的团队管理接口，其它家要么没有、要么已经走 UsageReporter
const supportsUsageCredential = computed(() => active.value?.name === 'exa');

const usageDirty = computed(() => {
    const provider = active.value;
    if (!provider) return false;
    return usageForm.count !== provider.monthlyUsed
        || usageCostMicro() !== provider.monthlyCostMicroUsd;
});

function usageCostMicro() {
    return Math.round((usageForm.costUsd || 0) * MICRO_PER_USD);
}

function formatUsd(micro: number) {
    return `$${(micro / MICRO_PER_USD).toFixed(2)}`;
}

async function onSaveUsage() {
    const provider = active.value;
    if (!provider) return;
    savingUsage.value = true;
    try {
        await updateGatewayProviderUsage(provider.name, {
            count: usageForm.count,
            costMicroUsd: usageCostMicro()
        });
        notify.success('用量已按填写的数字更新');
        emit('refresh');
    } finally {
        savingUsage.value = false;
    }
}

async function onSaveUsageCredential() {
    const provider = active.value;
    if (!provider) return;
    savingCredential.value = true;
    try {
        // service key 留空表示不改，避免一保存就把已存的密钥清掉
        await updateGatewayProvider(provider.name, {
            ...(usageCredential.serviceKey.trim() ? { usageServiceKey: usageCredential.serviceKey.trim() } : {}),
            usageKeyId: usageCredential.keyId.trim()
        });
        usageCredential.serviceKey = '';
        notify.success('上游用量凭据已保存');
        emit('refresh');
    } finally {
        savingCredential.value = false;
    }
}

async function onClearUsageCredential() {
    const provider = active.value;
    if (!provider) return;
    savingCredential.value = true;
    try {
        await updateGatewayProvider(provider.name, { usageServiceKey: '' });
        usageCredential.serviceKey = '';
        notify.success('已清除 Service Key，用量回到本地统计');
        emit('refresh');
    } finally {
        savingCredential.value = false;
    }
}

function keyHint(provider: GatewayProvider) {
    if (!provider.keys.length) return '未配置密钥';
    if (provider.activeKeys === provider.keys.length) return `${provider.activeKeys} 把密钥在轮换`;
    return `${provider.activeKeys}/${provider.keys.length} 把密钥在轮换`;
}

function keyHintClass(provider: GatewayProvider) {
    if (!provider.keys.length) return 'text-red-400';
    return provider.activeKeys === provider.keys.length ? 'text-gray-400' : 'text-orange-500';
}

function statusLabel(status: string) {
    if (status === 'auth_failed') return '密钥被拒';
    if (status === 'quota_exceeded') return '配额用尽';
    return status;
}

/**
 * upstream 从多把密钥里挑出余量最紧的一把展示。
 * 这里刻意不做求和：account 口径的额度是同账户共享的，几把密钥加起来会把同一份额度算好几遍。
 */
function upstream(provider: GatewayProvider) {
    const synced = provider.keys.filter((key) => key.upstreamSyncedAt);
    if (!synced.length) return null;
    const ratio = (key: GatewayProviderKey) => (key.upstreamLimit ? key.upstreamUsed / key.upstreamLimit : -1);
    const tightest = synced.reduce((worst, key) => (ratio(key) > ratio(worst) ? key : worst));
    return {
        used: tightest.upstreamUsed,
        limit: tightest.upstreamLimit,
        unit: tightest.upstreamUnit,
        tightest: synced.length > 1,
        warning: ratio(tightest) >= 0.9
    };
}

async function onSyncUsage() {
    syncing.value = true;
    try {
        const result = await syncGatewayUsage();
        const parts = [`已更新 ${result.synced} 把密钥`];
        if (result.skipped) parts.push(`跳过 ${result.skipped} 把`);
        if (result.failed) parts.push(`失败 ${result.failed} 把`);
        if (result.parked.length) parts.push(`停用 ${result.parked.join('、')}`);
        if (result.revived.length) parts.push(`恢复 ${result.revived.join('、')}`);
        // 失败不抛错：一家上游读不到用量，不该让另外两家的结果也看不见
        if (result.failed) notify.warning(parts.join('，'));
        else notify.success(parts.join('，'));
        emit('refresh');
    } finally {
        syncing.value = false;
    }
}

function openDrawer(provider: GatewayProvider) {
    activeName.value = provider.name;
    resetForm(provider);
    newKey.label = '';
    newKey.apiKey = '';
    drawerVisible.value = true;
}

function resetForm(provider: GatewayProvider) {
    form.baseUrl = provider.baseUrl;
    form.priority = provider.priority;
    form.weight = provider.weight;
    form.rps = provider.rps;
    form.monthlyQuota = provider.monthlyQuota;
    form.monthlyCostLimitUsd = provider.monthlyCostLimitMicroUsd / MICRO_PER_USD;
    usageForm.count = provider.monthlyUsed;
    usageForm.costUsd = provider.monthlyCostMicroUsd / MICRO_PER_USD;
    usageCredential.serviceKey = '';
    usageCredential.keyId = provider.usageKeyId || '';
    form.extra = provider.extra;
}

// 走整数取整，避免 0.1 这类小数在美元与微美元之间来回转出 999999 这种值
function costLimitMicro() {
    return Math.round((form.monthlyCostLimitUsd || 0) * MICRO_PER_USD);
}

// 列表刷新后，如果参数没被改过就跟着服务端的新值走，改过就保留操作者的输入
watch(() => props.providers, (list) => {
    const provider = list.find((item) => item.name === activeName.value);
    if (provider && !dirty.value) resetForm(provider);
});

async function onToggle(provider: GatewayProvider, enabled: boolean) {
    toggling.value = provider.name;
    try {
        await updateGatewayProvider(provider.name, { enabled });
        notify.success(enabled ? `${provider.displayName || provider.name} 已启用` : `${provider.displayName || provider.name} 已停用`);
        emit('refresh');
    } finally {
        toggling.value = '';
    }
}

async function onSaveParams() {
    const provider = active.value;
    if (!provider) return;
    if (form.extra.trim() && !isJson(form.extra)) {
        notify.warning('附加参数必须是合法 JSON');
        return;
    }
    saving.value = true;
    try {
        await updateGatewayProvider(provider.name, {
            baseUrl: form.baseUrl,
            priority: form.priority,
            weight: form.weight,
            rps: form.rps,
            monthlyQuota: form.monthlyQuota,
            monthlyCostLimitMicroUsd: costLimitMicro(),
            extra: form.extra
        });
        notify.success('参数已保存');
        emit('refresh');
    } finally {
        saving.value = false;
    }
}

// 草稿测试带上表单里的地址与附加参数，这样改了 baseUrl 也能在保存前验证
function probeBase() {
    return { baseUrl: form.baseUrl, extra: form.extra };
}

function reportProbe(result: { ok: boolean; latencyMs: number; resultCount?: number; error?: string }) {
    if (result.ok) {
        notify.success(`连通正常，耗时 ${result.latencyMs}ms，返回 ${result.resultCount ?? 0} 条结果`);
    } else {
        notify.error(`连通失败：${result.error}`);
    }
}

async function onTestDraft() {
    const provider = active.value;
    if (!provider) return;
    testing.value = 'draft';
    try {
        reportProbe(await testGatewayProvider(provider.name, { ...probeBase(), apiKey: newKey.apiKey.trim() }));
    } finally {
        testing.value = '';
    }
}

async function onTestStored(key: GatewayProviderKey) {
    const provider = active.value;
    if (!provider) return;
    testing.value = `key-${key.id}`;
    try {
        reportProbe(await testGatewayProvider(provider.name, { ...probeBase(), keyId: key.id }));
    } finally {
        testing.value = '';
    }
}

async function onAddKey() {
    const provider = active.value;
    if (!provider) return;
    adding.value = true;
    try {
        await createGatewayProviderKey(provider.name, {
            label: newKey.label.trim(),
            apiKey: newKey.apiKey.trim()
        });
        newKey.label = '';
        newKey.apiKey = '';
        notify.success('密钥已添加，立即参与轮换');
        emit('refresh');
    } finally {
        adding.value = false;
    }
}

async function onPatchKey(key: GatewayProviderKey, patch: { enabled?: boolean; revive?: boolean }) {
    const provider = active.value;
    if (!provider) return;
    await updateGatewayProviderKey(provider.name, key.id, patch);
    notify.success('已更新');
    emit('refresh');
}

async function onDeleteKey(key: GatewayProviderKey) {
    const provider = active.value;
    if (!provider) return;
    try {
        await ElMessageBox.confirm(`确定删除密钥「${key.label || key.masked}」吗？`, '提示', { type: 'warning' });
    } catch {
        return;
    }
    await deleteGatewayProviderKey(provider.name, key.id);
    notify.success('已删除');
    emit('refresh');
}

// 后端也会校验，这里先拦一道是为了让错误落在输入框旁边而不是变成一条接口报错
function isJson(value: string) {
    try {
        JSON.parse(value);
        return true;
    } catch {
        return false;
    }
}
</script>

<style scoped>
.card {
    padding: 16px 18px;
    border: 1px solid #edf0f3;
    border-radius: 12px;
    background-color: #fff;
    transition: border-color 0.2s, box-shadow 0.2s;
}

.card:hover {
    border-color: #d6e4ff;
    box-shadow: 0 4px 16px rgba(63, 140, 255, 0.08);
}

.card--off {
    background-color: #fbfbfc;
}

.card--off :deep(.el-progress-bar__inner) {
    background-color: #dcdfe6;
}

.link {
    color: #3f8cff;
    white-space: nowrap;
}

.link:hover {
    text-decoration: underline;
}

.drawer {
    padding-bottom: 8px;
}

.hint {
    margin: 0 0 18px;
    padding: 10px 12px;
    border-radius: 8px;
    background-color: #f7f9fc;
    font-size: 12px;
    line-height: 1.6;
    color: #667085;
}

.group-title {
    margin: 0 0 12px;
    font-size: 14px;
    font-weight: 600;
    color: #1f2937;
}

.group-title:not(:first-of-type) {
    margin-top: 26px;
}

.group-sub {
    margin-left: 8px;
    font-size: 12px;
    font-weight: 400;
    color: #98a2b3;
}

.label-hint {
    margin-left: 6px;
    font-size: 12px;
    font-weight: 400;
    color: #a8b0bd;
}

.empty-keys {
    padding: 14px;
    border: 1px dashed #e4e7ed;
    border-radius: 10px;
    text-align: center;
    font-size: 13px;
    color: #98a2b3;
}

.key-row {
    display: flex;
    align-items: center;
    gap: 10px;
    padding: 10px 12px;
    border: 1px solid #f0f2f5;
    border-radius: 10px;
}

.key-row+.key-row {
    margin-top: 8px;
}

.dot {
    width: 8px;
    height: 8px;
    border-radius: 50%;
    flex-shrink: 0;
}

.dot--on {
    background-color: #16a34a;
    box-shadow: 0 0 0 3px rgba(22, 163, 74, 0.14);
}

.dot--off {
    background-color: #d0d5dd;
}

.masked {
    padding: 1px 6px;
    border-radius: 4px;
    background-color: #f4f6f8;
    font-size: 12px;
    color: #667085;
}

.add-key {
    margin-top: 12px;
    padding: 12px;
    border: 1px dashed #dcdfe6;
    border-radius: 10px;
    background-color: #fcfcfd;
}
</style>
