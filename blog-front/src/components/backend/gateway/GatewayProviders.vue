<template>
    <el-card shadow="hover">
        <template #header>
            <div class="flex items-center justify-between">
                <span class="flex items-center">
                    <el-icon class="mr-2">
                        <Connection />
                    </el-icon> 搜索供应商
                </span>
                <el-button size="small" :icon="Refresh" @click="emit('refresh')">刷新</el-button>
            </div>
        </template>

        <el-empty v-if="!providers.length" description="暂无供应商" :image-size="60" />
        <el-collapse v-else v-model="expanded">
            <el-collapse-item v-for="provider in providers" :key="provider.name" :name="provider.name">
                <template #title>
                    <div class="flex items-center gap-3 w-full pr-4">
                        <ProviderLogo :name="provider.name" :src="provider.logoUrl" :size="24" />

                        <span class="font-medium">{{ provider.displayName || provider.name }}</span>
                        <el-tag :type="provider.enabled ? 'success' : 'info'" size="small">
                            {{ provider.enabled ? '已启用' : '未启用' }}
                        </el-tag>
                        <el-tag v-if="!provider.apiKeyPresent" type="danger" size="small" effect="plain">未配置秘钥</el-tag>
                        <el-tag v-if="provider.health !== 'closed'" type="warning" size="small">
                            {{ provider.health === 'open' ? '已熔断' : '半开探测中' }}
                        </el-tag>
                        <el-tag v-if="isDirty(provider)" type="warning" size="small" effect="dark">未保存</el-tag>
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

                <el-form v-if="drafts[provider.name]" label-position="top" size="default">
                    <div class="mb-3 text-xs text-gray-500">
                        <el-icon class="align-middle mr-1">
                            <InfoFilled />
                        </el-icon>{{ provider.billing }}
                    </div>

                    <el-divider content-position="left">凭据</el-divider>
                    <el-row :gutter="16">
                        <el-col :span="6">
                            <el-form-item label="启用">
                                <el-switch v-model="drafts[provider.name].enabled" />
                            </el-form-item>
                        </el-col>
                        <el-col :span="18">
                            <el-form-item
                                :label="`API 秘钥（当前：${provider.apiKeyPresent ? provider.apiKeyMasked : '未配置'}）`">
                                <el-input v-model="drafts[provider.name].apiKey" show-password :prefix-icon="Key"
                                    placeholder="留空表示不修改已保存的秘钥" />
                            </el-form-item>
                        </el-col>
                    </el-row>
                    <el-form-item label="接口地址（留空使用官方地址）">
                        <el-input v-model="drafts[provider.name].baseUrl" placeholder="https://api.tavily.com"
                            clearable />
                    </el-form-item>

                    <el-divider content-position="left">调度与限额</el-divider>
                    <el-row :gutter="16">
                        <el-col :span="6">
                            <el-form-item label="优先级（越小越优先）">
                                <el-input-number v-model="drafts[provider.name].priority" :min="0" :max="999"
                                    class="w-full" />
                            </el-form-item>
                        </el-col>
                        <el-col :span="6">
                            <el-form-item label="权重">
                                <el-input-number v-model="drafts[provider.name].weight" :min="0" :max="100"
                                    class="w-full" />
                            </el-form-item>
                        </el-col>
                        <el-col :span="6">
                            <el-form-item label="出站限速（次/秒，0 表示不限）">
                                <el-input-number v-model="drafts[provider.name].rps" :min="0" :max="100" :step="0.5"
                                    class="w-full" />
                            </el-form-item>
                        </el-col>
                        <el-col :span="6">
                            <el-form-item label="月配额（0 表示不限）">
                                <el-input-number v-model="drafts[provider.name].monthlyQuota" :min="0" :max="1000000"
                                    class="w-full" />
                            </el-form-item>
                        </el-col>
                    </el-row>
                    <el-form-item label="附加参数（JSON）">
                        <el-input v-model="drafts[provider.name].extra" placeholder='{"search_depth":"basic"}' />
                    </el-form-item>

                    <div class="text-right">
                        <el-button v-if="isDirty(provider)" @click="resetDraft(provider)">撤销改动</el-button>
                        <el-button :loading="testing === provider.name" @click="onTest(provider)">连通性测试</el-button>
                        <el-button type="primary" :loading="saving === provider.name" @click="onSave(provider)">
                            保存
                        </el-button>
                    </div>
                </el-form>
            </el-collapse-item>
        </el-collapse>
    </el-card>
</template>

<script setup lang="ts">
import { reactive, ref, watch } from 'vue';
import { Connection, InfoFilled, Key, Refresh } from '@element-plus/icons-vue';
import { notify } from '@/utils/notification';
import ProviderLogo from './ProviderLogo.vue';
import { formatCost } from './format';
import { testGatewayProvider, updateGatewayProvider, type GatewayProvider } from '@/api/gateway';

const props = defineProps<{ providers: GatewayProvider[] }>();
const emit = defineEmits<{ refresh: [] }>();

/** 表单草稿。列表里的秘钥是脱敏值，直接绑定会把脱敏串当成新秘钥存回去 */
interface ProviderDraft {
    enabled: boolean;
    apiKey: string;
    baseUrl: string;
    priority: number;
    weight: number;
    rps: number;
    monthlyQuota: number;
    extra: string;
}

const drafts = reactive<Record<string, ProviderDraft>>({});
const expanded = ref<string[]>([]);
const saving = ref('');
const testing = ref('');

function draftOf(provider: GatewayProvider): ProviderDraft {
    return {
        enabled: provider.enabled,
        apiKey: '',
        baseUrl: provider.baseUrl,
        priority: provider.priority,
        weight: provider.weight,
        rps: provider.rps,
        monthlyQuota: provider.monthlyQuota,
        extra: provider.extra
    };
}

function isDirty(provider: GatewayProvider) {
    const draft = drafts[provider.name];
    if (!draft) return false;
    return draft.apiKey.trim() !== ''
        || draft.enabled !== provider.enabled
        || draft.baseUrl !== provider.baseUrl
        || draft.priority !== provider.priority
        || draft.weight !== provider.weight
        || draft.rps !== provider.rps
        || draft.monthlyQuota !== provider.monthlyQuota
        || draft.extra !== provider.extra;
}

function resetDraft(provider: GatewayProvider) {
    drafts[provider.name] = draftOf(provider);
}

// 列表刷新常常由"保存另一家"触发，改动过的面板不跟着重置，否则会顺手清掉刚粘贴进去的秘钥
watch(() => props.providers, (list) => {
    list.forEach((provider) => {
        if (drafts[provider.name] && isDirty(provider)) return;
        drafts[provider.name] = draftOf(provider);
    });
    if (!expanded.value.length && list.length) {
        expanded.value = [list[0].name];
    }
}, { immediate: true });

async function onSave(provider: GatewayProvider) {
    const draft = drafts[provider.name];
    if (draft.extra.trim() && !isJson(draft.extra)) {
        notify.warning('附加参数必须是合法 JSON');
        return;
    }

    saving.value = provider.name;
    try {
        await updateGatewayProvider(provider.name, {
            enabled: draft.enabled,
            // 留空表示不修改，后端也会忽略空串
            apiKey: draft.apiKey,
            baseUrl: draft.baseUrl,
            priority: draft.priority,
            weight: draft.weight,
            rps: draft.rps,
            monthlyQuota: draft.monthlyQuota,
            extra: draft.extra
        });
        draft.apiKey = '';
        notify.success(`${provider.displayName || provider.name} 配置已保存`);
        emit('refresh');
    } finally {
        saving.value = '';
    }
}

async function onTest(provider: GatewayProvider) {
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

// 后端也会校验，这里先拦一道是为了让错误落在输入框旁边而不是一条接口报错
function isJson(value: string) {
    try {
        JSON.parse(value);
        return true;
    } catch {
        return false;
    }
}
</script>
