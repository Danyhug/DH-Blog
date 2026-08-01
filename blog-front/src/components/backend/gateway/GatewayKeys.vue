<template>
    <div>
        <el-card shadow="hover" class="mb-5">
            <template #header>
                <span class="flex items-center">
                    <el-icon class="mr-2">
                        <Link />
                    </el-icon> 接入方式
                </span>
            </template>

            <el-alert type="warning" :closable="false" class="mb-4" show-icon>
                <template #title>
                    网关 Key 只应用于服务端或 agent 侧。本站 CORS 允许任意来源，Key 一旦写进浏览器代码即等同公开。
                </template>
            </el-alert>

            <el-descriptions :column="1" border size="small" class="mb-4">
                <el-descriptions-item label="基础地址">
                    <div class="flex items-center gap-2">
                        <code>{{ baseUrl }}</code>
                        <el-button link type="primary" :icon="CopyDocument" @click="copy(baseUrl, '基础地址')" />
                    </div>
                </el-descriptions-item>
                <el-descriptions-item label="鉴权">
                    <code>Authorization: Bearer gw_live_...</code>
                    <span class="text-gray-400 ml-2">也接受 <code>X-API-Key</code> 头</span>
                </el-descriptions-item>
            </el-descriptions>

            <el-table :data="endpoints" size="small">
                <el-table-column label="方法" width="80">
                    <template #default="scope">
                        <el-tag :type="scope.row.method === 'GET' ? 'success' : 'primary'" size="small" effect="plain">
                            {{ scope.row.method }}
                        </el-tag>
                    </template>
                </el-table-column>
                <el-table-column label="路径" min-width="220">
                    <template #default="scope"><code>{{ scope.row.path }}</code></template>
                </el-table-column>
                <el-table-column prop="desc" label="说明" min-width="320" />
            </el-table>

            <div class="mt-3 text-xs text-gray-400">
                原生透传保持上游的请求与响应格式，现有 SDK 改 base_url 即可；透传不做跨供应商回退，上游报什么就返回什么。
            </div>
        </el-card>

        <el-card shadow="hover" class="mb-5">
            <template #header>
                <span class="flex items-center">
                    <el-icon class="mr-2">
                        <MagicStick />
                    </el-icon> 接入 Claude Code
                </span>
            </template>

            <el-alert v-if="insecureTransport" type="warning" :closable="false" show-icon class="mb-4">
                <template #title>
                    当前基础地址是 http，Key 会以明文经过网络。建议只在本机或内网这么连，公网请尽快换成 https。
                </template>
            </el-alert>

            <div class="text-sm text-gray-600 mb-2">1. 终端执行（把 Key 换成下面签发的那把）：</div>
            <div class="code-block">
                <pre>{{ mcpAddCommand }}</pre>
                <el-button class="!absolute top-2 right-2" link type="primary" :icon="CopyDocument"
                    @click="copy(mcpAddCommand, '命令')" />
            </div>

            <div class="text-sm text-gray-600 mt-4 mb-2">
                或写进项目的 <code>.mcp.json</code>，Key 从环境变量取、不落进仓库：
            </div>
            <div class="code-block">
                <pre>{{ mcpJson }}</pre>
                <el-button class="!absolute top-2 right-2" link type="primary" :icon="CopyDocument"
                    @click="copy(mcpJson, '配置')" />
            </div>

            <div class="mt-4 text-xs text-gray-400 leading-relaxed">
                2. 装好后在 Claude Code 里执行 <code>/mcp</code> 应该能看到 <code>dh-search</code>，工具名 <code>web_search</code>，
                可选的 provider 会按这把 Key 的供应商限制自动裁剪。<br />
                3. 想让搜索一律走网关，在 <code>~/.claude/settings.json</code> 里加
                <code>"permissions": { "deny": ["WebSearch"] }</code> 禁掉内置搜索。<br />
                MCP 调用与统一接口共用限速、配额与缓存，流水里的 endpoint 记为 <code>mcp/search</code>，可在请求日志里单独筛。
            </div>
        </el-card>

        <el-card shadow="hover">
            <template #header>
                <div class="flex items-center justify-between">
                    <span class="flex items-center">
                        <el-icon class="mr-2">
                            <Key />
                        </el-icon> 网关 API Key
                    </span>
                    <div class="flex gap-2">
                        <el-button size="small" :icon="Refresh" @click="load">刷新</el-button>
                        <el-button type="primary" size="small" @click="openCreateDialog">新建 Key</el-button>
                    </div>
                </div>
            </template>

            <el-table v-loading="loading" :data="apiKeys" size="default" empty-text="尚未签发 Key">
                <el-table-column prop="name" label="名称" min-width="140" />
                <el-table-column prop="keyPrefix" label="前缀" min-width="180" />
                <el-table-column label="状态" min-width="90">
                    <template #default="scope">
                        <el-switch :model-value="scope.row.enabled"
                            @change="(value: any) => onToggle(scope.row, Boolean(value))" />
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
                <el-table-column label="过期时间" min-width="170">
                    <template #default="scope">{{ scope.row.expireAt ? formatTime(scope.row.expireAt) : '永不过期' }}</template>
                </el-table-column>
                <el-table-column label="最后使用" min-width="170">
                    <template #default="scope">{{ formatTime(scope.row.lastUsedAt) }}</template>
                </el-table-column>
                <el-table-column label="操作" width="100">
                    <template #default="scope">
                        <el-button size="small" type="danger" link @click="onDelete(scope.row)">吊销</el-button>
                    </template>
                </el-table-column>
            </el-table>
        </el-card>

        <!-- 新建 Key -->
        <el-dialog v-model="createDialogVisible" title="新建网关 API Key" width="480px">
            <el-form :model="createForm" label-width="110px" size="default">
                <el-form-item label="名称" required>
                    <el-input v-model="createForm.name" placeholder="例如：我的写作 agent" clearable />
                </el-form-item>
                <el-form-item label="允许的供应商">
                    <el-select v-model="createAllowed" multiple placeholder="留空表示全部" class="w-full">
                        <el-option v-for="provider in providers" :key="provider.name"
                            :label="provider.displayName || provider.name" :value="provider.name" />
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
                <el-button type="primary" :loading="creating" @click="onCreate">创建</el-button>
            </template>
        </el-dialog>

        <!-- 明文只展示一次 -->
        <el-dialog v-model="secretDialogVisible" title="请立即保存这个 Key" width="620px" :close-on-click-modal="false">
            <el-alert type="warning" :closable="false" show-icon class="mb-4" title="明文只显示这一次，关闭后无法再次查看。" />
            <el-input :model-value="createdSecret" readonly />

            <div class="mt-4 mb-2 text-sm text-gray-600">接入 Claude Code，直接复制执行：</div>
            <div class="code-block">
                <pre>{{ createdMcpCommand }}</pre>
                <el-button class="!absolute top-2 right-2" link type="primary" :icon="CopyDocument"
                    @click="copy(createdMcpCommand, '命令')" />
            </div>

            <template #footer>
                <el-button type="primary" @click="copySecret">复制 Key 并关闭</el-button>
            </template>
        </el-dialog>
    </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue';
import { ElMessageBox } from 'element-plus';
import { CopyDocument, Key, Link, MagicStick, Refresh } from '@element-plus/icons-vue';
import { notify } from '@/utils/notification';
import { formatTime } from './format';
import {
    createGatewayApiKey,
    deleteGatewayApiKey,
    gatewayBaseUrl,
    getGatewayApiKeys,
    updateGatewayApiKey,
    type CreateGatewayApiKeyPayload,
    type GatewayApiKey,
    type GatewayProvider
} from '@/api/gateway';

defineProps<{ providers: GatewayProvider[] }>();

const baseUrl = gatewayBaseUrl();
const endpoints = [
    { method: 'POST', path: '/search', desc: '统一搜索，网关按调度方式选路，响应为统一格式' },
    { method: 'GET', path: '/search?q=...', desc: '同上，方便用浏览器或 curl 直接试' },
    { method: 'GET', path: '/providers', desc: '列出当前可用的供应商及其能力' },
    { method: 'POST', path: '/tavily/search', desc: 'Tavily 原生透传' },
    { method: 'GET', path: '/brave/web/search', desc: 'Brave 原生透传' },
    { method: 'POST', path: '/exa/search', desc: 'Exa 原生透传' },
    { method: 'POST', path: '/mcp', desc: 'MCP Server，供 Claude Code 等 MCP 客户端挂载，见下方接入说明' }
];

const mcpUrl = `${baseUrl}/mcp`;
// 用户暂时没上 https 时如实提示，而不是假装 Key 传输是安全的
const insecureTransport = mcpUrl.startsWith('http://');

function mcpCommandFor(key: string) {
    return `claude mcp add --transport http dh-search ${mcpUrl} \\\n  --header "Authorization: Bearer ${key}" --scope user`;
}

const mcpAddCommand = mcpCommandFor('<你的网关 Key>');

// 单引号字符串，${DH_GATEWAY_KEY} 是要原样写进配置的占位符，不能被模板插值吃掉
const mcpJson = JSON.stringify({
    mcpServers: {
        'dh-search': {
            type: 'http',
            url: mcpUrl,
            headers: { Authorization: 'Bearer ${DH_GATEWAY_KEY}' }
        }
    }
}, null, 2);

const apiKeys = ref<GatewayApiKey[]>([]);
const loading = ref(false);

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

// 刚签发的 Key 直接拼进命令里，省得再手动粘一次
const createdMcpCommand = computed(() => mcpCommandFor(createdSecret.value));

async function load() {
    loading.value = true;
    try {
        apiKeys.value = await getGatewayApiKeys();
    } finally {
        loading.value = false;
    }
}

function openCreateDialog() {
    createForm.value = { name: '', rateLimitPerMin: 60, monthlyQuota: 0, expireDays: 0, note: '' };
    createAllowed.value = [];
    createDialogVisible.value = true;
}

async function onCreate() {
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
        await load();
    } finally {
        creating.value = false;
    }
}

async function copy(text: string, label: string) {
    try {
        await navigator.clipboard.writeText(text);
        notify.success(`${label}已复制`);
    } catch {
        notify.warning('复制失败，请手动选中复制');
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

async function onToggle(key: GatewayApiKey, enabled: boolean) {
    await updateGatewayApiKey(key.id, { enabled });
    notify.success(enabled ? 'Key 已启用' : 'Key 已停用');
    await load();
}

async function onDelete(key: GatewayApiKey) {
    try {
        await ElMessageBox.confirm(`确定要吊销「${key.name}」吗？使用该 Key 的 agent 将立刻失效。`, '提示', { type: 'warning' });
    } catch {
        return;
    }
    await deleteGatewayApiKey(key.id);
    notify.success('已吊销');
    await load();
}

onMounted(load);
</script>

<style scoped>
.code-block {
    position: relative;
    padding: 12px 40px 12px 14px;
    border: 1px solid #e4e7ed;
    border-radius: 8px;
    background-color: #fafafa;
}

.code-block pre {
    margin: 0;
    font-size: 12px;
    line-height: 1.7;
    color: #476582;
    white-space: pre-wrap;
    word-break: break-all;
}
</style>
