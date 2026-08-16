<template>
    <div>
        <SectionPanel title="接入方式" subtitle="网关自己的 Key 只应用于服务端或 agent 侧，写进浏览器代码即等同公开">
            <template #icon>
                <el-icon>
                    <Link />
                </el-icon>
            </template>

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

            <p class="mt-3 mb-0 text-xs text-gray-400 leading-relaxed">
                原生透传保持上游的请求与响应格式，现有 SDK 改 base_url 即可；透传不做跨供应商回退，上游报什么就返回什么。
            </p>
        </SectionPanel>

        <SectionPanel title="接入 Claude Code" subtitle="装好后用它替掉 Claude Code 内置的 WebSearch">
            <template #icon>
                <el-icon>
                    <MagicStick />
                </el-icon>
            </template>

            <el-alert v-if="insecureTransport" type="warning" :closable="false" show-icon class="mb-4">
                <template #title>
                    当前基础地址是 http，Key 会以明文经过网络。建议只在本机或内网这么连，公网请尽快换成 https。
                </template>
            </el-alert>

            <div class="step">1. 挂上 MCP Server（把 Key 换成下面签发的那把）</div>
            <div class="code-block">
                <pre>{{ mcpAddCommand }}</pre>
                <el-button class="copy-btn" link type="primary" :icon="CopyDocument"
                    @click="copy(mcpAddCommand, '命令')" />
            </div>

            <div class="step">或写进项目的 <code>.mcp.json</code>，Key 从环境变量取、不落进仓库</div>
            <div class="code-block">
                <pre>{{ mcpJson }}</pre>
                <el-button class="copy-btn" link type="primary" :icon="CopyDocument" @click="copy(mcpJson, '配置')" />
            </div>

            <div class="step">
                2. 禁掉内置搜索，写进 <code>~/.claude/settings.json</code>
                <span class="step-sub">不禁的话两个搜索工具并存，模型多半仍会用内置那个</span>
            </div>
            <div class="code-block">
                <pre>{{ denyWebSearchJson }}</pre>
                <el-button class="copy-btn" link type="primary" :icon="CopyDocument"
                    @click="copy(denyWebSearchJson, '配置')" />
            </div>

            <p class="mt-4 mb-0 text-xs text-gray-400 leading-relaxed">
                3. 在 Claude Code 里执行 <code>/mcp</code> 应该能看到 <code>dh-search</code>，
                工具名为 <code>web_search</code>，可选的 provider 会按这把 Key 的供应商限制自动裁剪。<br />
                内置 <code>WebSearch</code> 只给标题和链接，要读正文还得再 <code>WebFetch</code> 一次；
                这个工具直接带摘要，需要全文时把 <code>include_raw_content</code> 设为 true 即可，省掉那一步。<br />
                域名过滤同时认 <code>allowed_domains</code> / <code>blocked_domains</code>（内置搜索的叫法）
                和 <code>include_domains</code> / <code>exclude_domains</code>。<br />
                MCP 调用与统一接口共用限速、配额与缓存，流水里的 endpoint 记为 <code>mcp/search</code>。
            </p>
        </SectionPanel>

        <SectionPanel title="网关 API Key" subtitle="签发给 agent 的凭据，可随时吊销；明文可重复复制" flush>
            <template #icon>
                <el-icon>
                    <Key />
                </el-icon>
            </template>
            <template #extra>
                <div class="flex gap-2">
                    <el-button size="small" :icon="Refresh" @click="load">刷新</el-button>
                    <el-button size="small" type="primary" :icon="Plus" @click="openCreateDialog">新建 Key</el-button>
                </div>
            </template>

            <el-table v-loading="loading" :data="apiKeys" size="default" empty-text="尚未签发 Key">
                <el-table-column prop="name" label="名称" min-width="130" />
                <el-table-column label="Key" min-width="210">
                    <template #default="scope">
                        <div class="flex items-center gap-1">
                            <code class="masked">{{ scope.row.keyPrefix }}…</code>
                            <el-button link type="primary" size="small" :icon="CopyDocument"
                                :loading="revealing === scope.row.id" @click="onCopyKey(scope.row)" />
                        </div>
                    </template>
                </el-table-column>
                <el-table-column label="状态" min-width="80">
                    <template #default="scope">
                        <el-switch :model-value="scope.row.enabled"
                            @change="(value: any) => onToggle(scope.row, Boolean(value))" />
                    </template>
                </el-table-column>
                <el-table-column label="供应商限制" min-width="120">
                    <template #default="scope">{{ scope.row.allowedProviders || '全部' }}</template>
                </el-table-column>
                <el-table-column label="能力" min-width="160">
                    <template #default="scope">
                        <template v-if="scope.row.scopes">
                            <el-tag v-for="s in scope.row.scopes.split(',')" :key="s" size="small"
                                effect="plain" class="mr-1">
                                {{ scopeLabel(s) }}
                            </el-tag>
                        </template>
                        <span v-else class="text-gray-400">仅搜索</span>
                    </template>
                </el-table-column>
                <el-table-column label="署名" min-width="130">
                    <template #default="scope">
                        <span :class="scope.row.authorName ? '' : 'text-gray-400'">
                            {{ scope.row.authorName || '—' }}
                        </span>
                    </template>
                </el-table-column>
                <el-table-column label="限速" min-width="100">
                    <template #default="scope">
                        {{ scope.row.rateLimitPerMin ? `${scope.row.rateLimitPerMin} 次/分` : '不限' }}
                    </template>
                </el-table-column>
                <el-table-column label="本月用量" min-width="110">
                    <template #default="scope">
                        {{ scope.row.monthlyUsed }} / {{ scope.row.monthlyQuota || '不限' }}
                    </template>
                </el-table-column>
                <el-table-column label="最后使用" min-width="160">
                    <template #default="scope">{{ formatTime(scope.row.lastUsedAt) }}</template>
                </el-table-column>
                <el-table-column label="操作" width="80">
                    <template #default="scope">
                        <el-button size="small" type="danger" link @click="onDelete(scope.row)">吊销</el-button>
                    </template>
                </el-table-column>
            </el-table>
        </SectionPanel>

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
                <el-form-item label="能力范围">
                    <el-select v-model="createScopes" multiple placeholder="留空 = 纯搜索 Key" class="w-full">
                        <el-option v-for="opt in scopeOptions" :key="opt.value" :label="opt.label" :value="opt.value" />
                    </el-select>
                    <div class="text-xs text-gray-400 mt-1">
                        联网搜索对任何 Key 始终可用，无需勾选；下方只配置内容能力（留空即为纯搜索 Key）。
                        content:write 才能创建/修改文章与传图。
                    </div>
                </el-form-item>
                <el-form-item label="署名">
                    <el-input v-model="createForm.authorName" placeholder="写文章时显示的作者名，留空用 Key 名" clearable />
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

        <!-- 明文展示：列表里点复制也走这个弹窗 -->
        <el-dialog v-model="secretDialogVisible" :title="`Key · ${secretName}`" width="620px">
            <el-input :model-value="createdSecret" readonly>
                <template #append>
                    <el-button :icon="CopyDocument" @click="copy(createdSecret, 'Key')">复制</el-button>
                </template>
            </el-input>

            <div class="step mt-4">接入 Claude Code，直接复制执行</div>
            <div class="code-block">
                <pre>{{ createdMcpCommand }}</pre>
                <el-button class="copy-btn" link type="primary" :icon="CopyDocument"
                    @click="copy(createdMcpCommand, '命令')" />
            </div>
            <p class="mt-3 mb-0 text-xs text-gray-400">
                装完记得按上面第 2 步禁掉内置 <code>WebSearch</code>，否则两个搜索工具并存。
            </p>

            <template #footer>
                <el-button type="primary" @click="secretDialogVisible = false">关闭</el-button>
            </template>
        </el-dialog>
    </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue';
import { ElMessageBox } from 'element-plus';
import { CopyDocument, Key, Link, MagicStick, Plus, Refresh } from '@element-plus/icons-vue';
import { notify } from '@/utils/notification';
import SectionPanel from './SectionPanel.vue';
import { formatTime } from './format';
import {
    createGatewayApiKey,
    deleteGatewayApiKey,
    gatewayBaseUrl,
    getGatewayApiKeys,
    revealGatewayApiKey,
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
    { method: 'POST', path: '/firecrawl/search', desc: 'Firecrawl 原生透传' },
    { method: 'POST', path: '/mcp', desc: 'MCP Server，供 Claude Code 等 MCP 客户端挂载' }
];

const mcpUrl = `${baseUrl}/mcp`;
// 站点还没上 https 时如实提示，而不是假装 Key 的传输是安全的
const insecureTransport = mcpUrl.startsWith('http://');

function mcpCommandFor(key: string) {
    return `claude mcp add --transport http dh-search ${mcpUrl} \\\n  --header "Authorization: Bearer ${key}" --scope user`;
}

const mcpAddCommand = mcpCommandFor('<你的网关 Key>');

// WebSearch 的权限规则不带参数，deny 里写裸工具名就是全部禁用
const denyWebSearchJson = JSON.stringify({
    permissions: { deny: ['WebSearch'] }
}, null, 2);

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
const revealing = ref(0);

const createDialogVisible = ref(false);
const creating = ref(false);
const createAllowed = ref<string[]>([]);
// 搜索恒为基线能力，前端不再把它做成可勾选项；这里只配置内容权限。
// 若一条存量 Key 的 scopes 里仍有 search（后端 NormalizeScopes 仍接受它），
// scopeLabel 的映射把这种旧值渲染成可读文案。
const scopeOptions = [
    { value: 'content:read', label: '读取文章' },
    { value: 'content:write', label: '写入文章' }
];
const scopeLabels: Record<string, string> = {
    search: '联网搜索',
    'content:read': '读取文章',
    'content:write': '写入文章'
};
const createScopes = ref<string[]>([]);
const scopeLabel = (s: string) => scopeLabels[s] || s;
const createForm = ref<CreateGatewayApiKeyPayload>({
    name: '',
    rateLimitPerMin: 60,
    monthlyQuota: 0,
    expireDays: 0,
    note: '',
    authorName: ''
});
const secretDialogVisible = ref(false);
const createdSecret = ref('');
const secretName = ref('');

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
    createForm.value = { name: '', rateLimitPerMin: 60, monthlyQuota: 0, expireDays: 0, note: '', authorName: '' };
    createAllowed.value = [];
    createScopes.value = [];
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
            allowedProviders: createAllowed.value.join(','),
            scopes: createScopes.value.join(',')
        });
        createDialogVisible.value = false;
        showSecret(created.name, created.apiKey);
        await load();
    } finally {
        creating.value = false;
    }
}

function showSecret(name: string, secret: string) {
    secretName.value = name;
    createdSecret.value = secret;
    secretDialogVisible.value = true;
}

// 明文存在库里就是为了这一步：Key 掉了不用重签发、也不用挨个改 agent 的配置
async function onCopyKey(key: GatewayApiKey) {
    revealing.value = key.id;
    try {
        const revealed = await revealGatewayApiKey(key.id);
        await copy(revealed.apiKey, 'Key');
        showSecret(revealed.name, revealed.apiKey);
    } finally {
        revealing.value = 0;
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
.step {
    margin-bottom: 8px;
    font-size: 13px;
    color: #475467;
}

.step:not(:first-of-type) {
    margin-top: 16px;
}

.step-sub {
    margin-left: 8px;
    font-size: 12px;
    color: #98a2b3;
}

.code-block {
    position: relative;
    padding: 12px 40px 12px 14px;
    border: 1px solid #e9edf2;
    border-radius: 10px;
    background-color: #fafbfc;
}

.code-block pre {
    margin: 0;
    font-size: 12px;
    line-height: 1.75;
    color: #476582;
    white-space: pre-wrap;
    word-break: break-all;
}

.copy-btn {
    position: absolute;
    top: 8px;
    right: 10px;
}

.masked {
    padding: 1px 6px;
    border-radius: 4px;
    background-color: #f4f6f8;
    font-size: 12px;
    color: #667085;
}
</style>
