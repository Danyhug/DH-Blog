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
                <code>/mcp</code> 的工具目录与接入步骤见「MCP 能力」标签页。
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
                    <el-checkbox-group v-model="createScopes" class="w-full">
                        <div v-for="opt in selectableScopes" :key="opt.value" class="mb-1 last:mb-0">
                            <el-checkbox :value="opt.value">
                                {{ opt.label }} <code>{{ opt.value }}</code>
                            </el-checkbox>
                            <div class="text-xs text-gray-400 leading-relaxed pl-6 -mt-1">{{ opt.description }}</div>
                        </div>
                    </el-checkbox-group>
                    <div v-if="baselineScope" class="text-xs text-gray-400 mt-1 leading-relaxed">
                        {{ baselineScope.label }}对任何 Key 始终可用，无需勾选；上面都不勾就是一把纯搜索 Key。
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
            <CodeBlock :code="createdMcpCommand" label="命令" />
            <p class="mt-3 mb-0 text-xs text-gray-400">
                其余接入步骤（禁用内置 <code>WebSearch</code>、这把 Key 能看到哪些工具）见「MCP 能力」标签页。
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
import { CopyDocument, Key, Link, Plus, Refresh } from '@element-plus/icons-vue';
import { notify } from '@/utils/notification';
import SectionPanel from './SectionPanel.vue';
import CodeBlock from './CodeBlock.vue';
import { formatTime } from './format';
import {
    createGatewayApiKey,
    deleteGatewayApiKey,
    gatewayBaseUrl,
    getGatewayApiKeys,
    getGatewayMcpCatalog,
    revealGatewayApiKey,
    updateGatewayApiKey,
    type CreateGatewayApiKeyPayload,
    type GatewayApiKey,
    type GatewayMcpScope,
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

const apiKeys = ref<GatewayApiKey[]>([]);
const loading = ref(false);
const revealing = ref(0);

const createDialogVisible = ref(false);
const creating = ref(false);
const createAllowed = ref<string[]>([]);
// 能力清单取自后端的 scope 目录，后端加一类能力这里自动多一个勾选项。
// 基线能力（搜索）每把 Key 自带，做成勾选项会让人以为可以取消，所以只在说明里提一句；
// 存量 Key 的 scopes 里若仍显式存着它，scopeLabel 照样渲染得出中文名。
const scopeCatalog = ref<GatewayMcpScope[]>([]);
// 挂载别名跟着服务器自报的名字走，和「MCP 能力」页给的命令保持一致
const serverName = ref('dh-blog');
const selectableScopes = computed(() => scopeCatalog.value.filter((item) => !item.baseline));
const baselineScope = computed(() => scopeCatalog.value.find((item) => item.baseline));
const createScopes = ref<string[]>([]);
const scopeLabel = (value: string) =>
    scopeCatalog.value.find((item) => item.value === value)?.label || value;
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

const createdMcpCommand = computed(
    () => `claude mcp add --transport http ${serverName.value} ${mcpUrl} \\\n` +
        `  --header "Authorization: Bearer ${createdSecret.value}" --scope user`
);

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

async function loadScopeCatalog() {
    const catalog = await getGatewayMcpCatalog();
    scopeCatalog.value = catalog.scopes;
    serverName.value = catalog.serverName;
}

onMounted(load);
// 目录只决定表单里渲染哪些勾选项，单独拉：它挂了不该连带把 Key 列表也空着
onMounted(loadScopeCatalog);
</script>

<style scoped>
.step {
    margin-bottom: 8px;
    font-size: 13px;
    color: #475467;
}

.masked {
    padding: 1px 6px;
    border-radius: 4px;
    background-color: #f4f6f8;
    font-size: 12px;
    color: #667085;
}
</style>
