<template>
    <div v-loading="loading">
        <SectionPanel title="MCP 服务器" subtitle="Claude Code 等 MCP 客户端挂上这一个地址，就能用到下面全部工具">
            <template #icon>
                <el-icon>
                    <Cpu />
                </el-icon>
            </template>
            <template #extra>
                <el-button size="small" :icon="Refresh" @click="load">刷新</el-button>
            </template>

            <el-descriptions :column="1" border size="small">
                <el-descriptions-item label="端点">
                    <div class="flex items-center gap-2">
                        <code>{{ mcpUrl }}</code>
                        <el-button link type="primary" :icon="CopyDocument" @click="copy(mcpUrl, '端点')" />
                    </div>
                </el-descriptions-item>
                <el-descriptions-item label="服务器">
                    <code>{{ catalog?.serverName || '-' }}</code>
                    <span class="ml-2 text-gray-400">v{{ catalog?.version || '-' }}</span>
                </el-descriptions-item>
                <el-descriptions-item label="鉴权">
                    <code>Authorization: Bearer gw_live_...</code>
                    <span class="ml-2 text-gray-400">用「接入密钥」里签发的网关 Key</span>
                </el-descriptions-item>
            </el-descriptions>

            <div class="mt-4">
                <div class="mb-2 text-[13px] text-[#475467]">
                    服务器说明
                    <span class="ml-2 text-xs text-[#98a2b3]">initialize 时交给模型，决定它愿不愿意主动用这些工具</span>
                </div>
                <p class="m-0 rounded-[10px] bg-[#fafbfc] px-3.5 py-3 text-xs leading-[1.9] text-[#476582]">
                    {{ catalog?.instructions || '—' }}
                </p>
            </div>
        </SectionPanel>

        <SectionPanel title="工具目录" subtitle="按能力分组；一把 Key 能看到哪些工具，取决于它勾了哪些能力范围">
            <template #icon>
                <el-icon>
                    <Tools />
                </el-icon>
            </template>

            <el-empty v-if="!groups.length" description="尚未挂载任何 MCP 工具" :image-size="60" />

            <div v-for="group in groups" :key="group.value" class="scope-group">
                <div class="mb-2 flex flex-wrap items-center gap-2">
                    <span class="text-sm font-semibold text-[#1f2937]">{{ group.label }}</span>
                    <code>{{ group.value }}</code>
                    <el-tag v-if="group.baseline" size="small" type="success" effect="plain">所有 Key 可用</el-tag>
                    <el-tag v-else size="small" :type="group.granted ? 'primary' : 'info'" effect="plain">
                        {{ group.granted }} 把 Key 已授权
                    </el-tag>
                    <span class="ml-auto text-xs text-[#98a2b3]">{{ group.tools.length }} 个工具</span>
                </div>
                <p class="mb-3 mt-0 text-xs leading-relaxed text-[#98a2b3]">{{ group.description }}</p>

                <el-collapse>
                    <el-collapse-item v-for="tool in group.tools" :key="tool.name" :name="tool.name">
                        <template #title>
                            <div class="flex min-w-0 flex-wrap items-center gap-2">
                                <code>{{ tool.name }}</code>
                                <span class="text-[13px] text-[#475467]">{{ tool.title }}</span>
                            </div>
                        </template>

                        <p class="m-0 whitespace-pre-line text-xs leading-[1.9] text-[#667085]">
                            {{ tool.description }}
                        </p>

                        <el-table v-if="tool.params.length" :data="tool.params" size="small" class="mt-3">
                            <el-table-column label="参数" min-width="140">
                                <template #default="scope">
                                    <code>{{ scope.row.name }}</code>
                                    <span v-if="scope.row.required" class="ml-1 text-[#f56c6c]">*</span>
                                </template>
                            </el-table-column>
                            <el-table-column prop="type" label="类型" width="100" />
                            <el-table-column prop="description" label="说明" min-width="320" />
                        </el-table>
                    </el-collapse-item>
                </el-collapse>
            </div>
        </SectionPanel>

        <SectionPanel title="接入 Claude Code" subtitle="先挂上服务器，再按你要用的能力看对应那一段">
            <template #icon>
                <el-icon>
                    <MagicStick />
                </el-icon>
            </template>

            <el-alert v-if="insecureTransport" type="warning" :closable="false" show-icon class="mb-4">
                <template #title>
                    当前端点是 http，Key 会以明文经过网络。建议只在本机或内网这么连，公网请尽快换成 https。
                </template>
            </el-alert>

            <div class="step">挂上 MCP Server（把 Key 换成「接入密钥」里签发的那把）</div>
            <CodeBlock :code="mcpAddCommand" label="命令" />

            <div class="step">或写进项目的 <code>.mcp.json</code>，Key 从环境变量取、不落进仓库</div>
            <CodeBlock :code="mcpJson" label="配置" />

            <p class="mt-3 mb-0 text-xs leading-relaxed text-[#98a2b3]">
                装完在 Claude Code 里执行 <code>/mcp</code> 应该能看到 <code>{{ serverName }}</code>。
                客户端列出的工具就是上面目录里这把 Key 有权限的那些，没勾的能力连工具名都看不到。<br />
                别名早期叫 <code>dh-search</code>，那时这台服务器确实只有搜索。已经按旧名装过的，
                先 <code>claude mcp remove dh-search --scope user</code> 再按上面重装；
                工具名会随之从 <code>mcp__dh-search__*</code> 变成 <code>mcp__{{ serverName }}__*</code>。
            </p>

            <el-divider />

            <div class="capability">用作联网搜索<span class="capability-scope">所有 Key 自带</span></div>
            <div class="step">
                禁掉内置搜索，写进 <code>~/.claude/settings.json</code>
                <span class="step-sub">不禁的话两个搜索工具并存，模型多半仍会用内置那个</span>
            </div>
            <CodeBlock :code="denyWebSearchJson" label="配置" />
            <p class="mt-3 mb-0 text-xs leading-relaxed text-[#98a2b3]">
                内置 <code>WebSearch</code> 只给标题和链接，要读正文还得再 <code>WebFetch</code> 一次；
                <code>web_search</code> 直接带摘要，需要全文时把 <code>include_raw_content</code> 设为 true 即可，省掉那一步。<br />
                域名过滤同时认 <code>allowed_domains</code> / <code>blocked_domains</code>（内置搜索的叫法）
                和 <code>include_domains</code> / <code>exclude_domains</code>。<br />
                可选的 provider 会按这把 Key 的供应商限制自动裁剪；调用与统一接口共用限速、配额与缓存，
                流水里的 endpoint 记为 <code>mcp/search</code>。
            </p>

            <el-divider />

            <div class="capability">用作博客写入<span class="capability-scope">需要 content:read / content:write</span></div>
            <p class="mt-0 mb-3 text-xs leading-relaxed text-[#98a2b3]">
                在「接入密钥」里新建 Key 时勾上对应能力范围，或给已有 Key 补上；能力是逐把配置的，
                不勾就还是一把纯搜索 Key。
            </p>
            <ul class="m-0 list-disc pl-5 text-xs leading-[2] text-[#667085]">
                <li>
                    <b>署名</b>：Key 的「署名」字段决定文章作者名，留空则用 Key 名，前台文章会带 Agent 徽标。
                </li>
                <li>
                    <b>改动范围</b>：一把 Key 只能免授权修改自己写的文章；改别人的（含站长手写的）需要你在
                    「博客管理」里对那一篇逐篇签发临时授权。
                </li>
                <li>
                    <b>加密文章</b>：不返回正文和摘要，也不允许修改。
                </li>
                <li>
                    <b>限速</b>：内容工具和搜索共用 Key 的「限速（次/分）」额度，不是各算各的。
                </li>
            </ul>
        </SectionPanel>
    </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue';
import { Cpu, CopyDocument, MagicStick, Refresh, Tools } from '@element-plus/icons-vue';
import { notify } from '@/utils/notification';
import SectionPanel from './SectionPanel.vue';
import CodeBlock from './CodeBlock.vue';
import {
    gatewayBaseUrl,
    getGatewayApiKeys,
    getGatewayMcpCatalog,
    type GatewayApiKey,
    type GatewayMcpCatalog,
    type GatewayMcpTool
} from '@/api/gateway';

const mcpUrl = `${gatewayBaseUrl()}/mcp`;
// 站点还没上 https 时如实提示，而不是假装 Key 的传输是安全的
const insecureTransport = mcpUrl.startsWith('http://');

const catalog = ref<GatewayMcpCatalog | null>(null);
const apiKeys = ref<GatewayApiKey[]>([]);
const loading = ref(false);

// 客户端挂载时的别名跟着服务器自报的名字走。写死一个名字迟早会和实际能力对不上：
// 这台服务器早就不只有搜索了，一把只开内容能力的 Key 挂成 dh-search 纯属误导。
const serverName = computed(() => catalog.value?.serverName || 'dh-blog');

const mcpAddCommand = computed(
    () => `claude mcp add --transport http ${serverName.value} ${mcpUrl} \\\n` +
        `  --header "Authorization: Bearer <你的网关 Key>" --scope user`
);

// WebSearch 的权限规则不带参数，deny 里写裸工具名就是全部禁用
const denyWebSearchJson = JSON.stringify({
    permissions: { deny: ['WebSearch'] }
}, null, 2);

// ${DH_GATEWAY_KEY} 是要原样写进配置的占位符，用单引号字符串避免被模板插值吃掉
const mcpJson = computed(() => JSON.stringify({
    mcpServers: {
        [serverName.value]: {
            type: 'http',
            url: mcpUrl,
            headers: { Authorization: 'Bearer ${DH_GATEWAY_KEY}' }
        }
    }
}, null, 2));

/**
 * 按 scope 把工具分组。分组和顺序都取后端的 scope 目录，工具挂在哪一组也由后端说了算，
 * 所以后端新注册一个工具、甚至新增一类能力，这个页面都不用改。
 */
const groups = computed(() => {
    const tools = catalog.value?.tools ?? [];
    const byScope = new Map<string, GatewayMcpTool[]>();
    for (const tool of tools) {
        const bucket = byScope.get(tool.scope);
        if (bucket) bucket.push(tool);
        else byScope.set(tool.scope, [tool]);
    }
    return (catalog.value?.scopes ?? [])
        .map((scope) => ({
            ...scope,
            tools: byScope.get(scope.value) ?? [],
            granted: grantedCount(scope.value)
        }))
        .filter((group) => group.tools.length > 0);
});

/**
 * 数一把 Key 显式勾了这个能力。基线能力不走这里（它对每把 Key 都成立，
 * 而空 scopes 恰恰表示「仅搜索」，按字面统计会数成 0）。
 */
function grantedCount(scope: string) {
    return apiKeys.value.filter((key) =>
        (key.scopes || '').split(',').map((item) => item.trim()).includes(scope)
    ).length;
}

async function copy(text: string, label: string) {
    try {
        await navigator.clipboard.writeText(text);
        notify.success(`${label}已复制`);
    } catch {
        notify.warning('复制失败，请手动选中复制');
    }
}

async function load() {
    loading.value = true;
    try {
        [catalog.value, apiKeys.value] = await Promise.all([getGatewayMcpCatalog(), getGatewayApiKeys()]);
    } finally {
        loading.value = false;
    }
}

onMounted(load);
</script>

<style scoped>
.scope-group+.scope-group {
    margin-top: 24px;
    padding-top: 20px;
    border-top: 1px dashed #eaecf0;
}

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

.capability {
    margin-bottom: 10px;
    font-size: 14px;
    font-weight: 600;
    color: #1f2937;
}

.capability-scope {
    margin-left: 8px;
    font-size: 12px;
    font-weight: 400;
    color: #98a2b3;
}

:deep(.el-collapse-item__header) {
    height: auto;
    padding: 10px 0;
    line-height: 1.6;
}
</style>
