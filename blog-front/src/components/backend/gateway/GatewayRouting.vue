<template>
    <div class="flex flex-col gap-[18px]">
        <SectionPanel title="调度方式" subtitle="仅在请求未指定 provider（即 auto）时生效">
            <template #icon>
                <el-icon>
                    <Switch />
                </el-icon>
            </template>
            <template #extra>
                <el-tag size="small" type="info" effect="plain">当前：{{ activeStrategyLabel }}</el-tag>
            </template>

            <div v-loading="loading" class="grid grid-cols-1 lg:grid-cols-3 gap-[12px]">
                <button v-for="option in strategies" :key="option.value" type="button"
                    :class="strategyCardClass(option)" :disabled="!option.implemented" @click="onPick(option.value)">
                    <span class="inline-flex items-center justify-center w-[34px] h-[34px] rounded-[10px] shrink-0"
                        :class="strategy === option.value ? 'bg-[#e8f1ff] text-[#3f8cff]' : 'bg-[#f4f6f9] text-[#98a2b3]'">
                        <el-icon :size="17">
                            <component :is="strategyIcon(option.value)" />
                        </el-icon>
                    </span>
                    <span class="min-w-0 flex-1 text-left">
                        <span class="flex items-center gap-[8px]">
                            <span class="text-[14px] font-semibold text-[#1f2937]">{{ option.label }}</span>
                            <el-tag v-if="!option.implemented" size="small" type="info" effect="plain">开发中</el-tag>
                        </span>
                        <span class="mt-[4px] block text-[12px] leading-[1.7] text-[#98a2b3]">
                            {{ option.description }}
                        </span>
                    </span>
                    <el-icon v-if="strategy === option.value" :size="18" color="#3f8cff" class="mt-[2px] shrink-0">
                        <CircleCheckFilled />
                    </el-icon>
                </button>
            </div>

            <p class="mt-[14px] mb-0 flex items-start gap-[7px] text-[12px] leading-[1.7] text-[#a8b0bd]">
                <el-icon class="mt-[2px] shrink-0">
                    <InfoFilled />
                </el-icon>
                能力、健康、密钥与配额过滤始终优先——调度方式只决定通过过滤后的先后顺序，
                所以换策略不会让一个用不了的供应商被选中。
            </p>
        </SectionPanel>

        <SectionPanel title="判断模型" subtitle="「模型判断」用它挑首选供应商，每次未命中缓存的 auto 搜索调用一次">
            <template #icon>
                <el-icon>
                    <MagicStick />
                </el-icon>
            </template>
            <template #extra>
                <el-tag size="small" :type="router?.configured ? 'success' : 'info'" effect="plain">
                    {{ router?.configured ? '已配置' : '未配置' }}
                </el-tag>
            </template>

            <div v-loading="loading" class="grid grid-cols-1 xl:grid-cols-[minmax(0,1fr)_340px] gap-[22px] items-start">
                <el-form label-position="top" size="default" class="min-w-0">
                    <el-form-item label="模型类型">
                        <el-radio-group v-model="form.backend">
                            <el-radio-button v-for="option in router?.backends ?? []" :key="option.value"
                                :value="option.value">
                                {{ option.label }}
                            </el-radio-button>
                        </el-radio-group>
                    </el-form-item>

                    <p v-if="backendOption"
                        class="mt-[-4px] mb-[16px] rounded-[10px] border border-[#e9eef5] bg-[#f8fafd] px-[12px] py-[9px] text-[12px] leading-[1.7] text-[#667085]">
                        {{ backendOption.description }}
                        <template v-if="form.backend === 'jev'">
                            Key 在
                            <el-link type="primary" href="https://console.typesafe.ai/" target="_blank"
                                class="align-baseline! text-[12px]!">TypeSafe 控制台</el-link>
                            获取。
                        </template>
                    </p>

                    <el-row :gutter="14">
                        <el-col :xs="24" :md="15">
                            <el-form-item label="接口地址">
                                <el-input v-model="form.endpoint" clearable
                                    :placeholder="backendOption?.defaultEndpoint || 'https://api.example.com/v1/chat/completions'" />
                            </el-form-item>
                        </el-col>
                        <el-col :xs="24" :md="9">
                            <el-form-item label="模型">
                                <el-input v-model="form.model" clearable
                                    :placeholder="backendOption?.defaultModel || '例如 deepseek-chat'" />
                            </el-form-item>
                        </el-col>
                    </el-row>
                    <p v-if="backendOption?.endpointHint"
                        class="mt-[-6px] mb-[16px] text-[12px] leading-[1.6] text-[#a8b0bd]">
                        {{ backendOption.endpointHint }}
                    </p>

                    <el-form-item>
                        <template #label>
                            API Key
                            <span class="ml-[6px] text-[12px] font-normal text-[#a8b0bd]">
                                留空表示不修改；切换类型或改了接口地址后要重新填，否则旧 Key 会被清空
                            </span>
                        </template>
                        <div class="flex w-full gap-[10px]">
                            <el-input v-model="form.apiKey" show-password clearable :disabled="form.clearApiKey"
                                :placeholder="keyPlaceholder" />
                            <el-button v-if="router?.apiKeyMasked && !form.clearApiKey"
                                @click="form.clearApiKey = true">清除</el-button>
                            <el-button v-if="form.clearApiKey" @click="form.clearApiKey = false">撤销</el-button>
                        </div>
                    </el-form-item>

                    <!-- 思维强度做成按钮：点一下就是选中，不用记任何字段名 -->
                    <el-form-item v-if="form.backend === 'openai'">
                        <template #label>
                            思维强度
                            <span class="ml-[6px] text-[12px] font-normal text-[#a8b0bd]">
                                点一下即选中；选「不指定」则不发这个字段，用模型自己的默认档
                            </span>
                        </template>
                        <div class="w-full">
                            <div class="flex flex-wrap gap-[8px]">
                                <button v-for="choice in effortChoices" :key="choice.value || 'unset'" type="button"
                                    class="h-[32px] px-[16px] rounded-[8px] border text-[13px] transition-all duration-150"
                                    :class="effortButtonClass(choice.value)"
                                    @click="form.reasoningEffort = choice.value">
                                    {{ choice.label }}
                                </button>
                            </div>
                            <p class="mt-[9px] mb-0 text-[12px] leading-[1.7] text-[#98a2b3]">{{ effortHint }}</p>
                        </div>
                    </el-form-item>

                    <el-row :gutter="14">
                        <el-col :xs="24" :md="12">
                            <el-form-item label="置信度阈值（0–1，低于它这次按负载均衡）">
                                <el-input-number v-model="form.minConfidence" :min="0" :max="1" :step="0.05"
                                    :precision="2" class="w-full" />
                            </el-form-item>
                        </el-col>
                        <el-col :xs="24" :md="12">
                            <el-form-item :label="`超时（毫秒，0 用默认值 ${backendOption?.defaultTimeoutMs ?? '-'}）`">
                                <el-input-number v-model="form.timeoutMs" :min="0" :max="30000" :step="500"
                                    class="w-full" />
                            </el-form-item>
                        </el-col>
                    </el-row>

                    <el-form-item>
                        <template #label>
                            提示词
                            <span class="ml-[6px] text-[12px] font-normal text-[#a8b0bd]">
                                只写判断要求；候选列表与回答格式由网关自动附加
                            </span>
                            <el-button v-if="form.prompt !== router?.defaultPrompt" link type="primary" size="small"
                                class="ml-[6px]" @click="form.prompt = router?.defaultPrompt ?? ''">恢复默认</el-button>
                        </template>
                        <el-input v-model="form.prompt" type="textarea" :autosize="{ minRows: 2, maxRows: 8 }" />
                    </el-form-item>

                    <el-form-item label="供应商描述（模型只看得到名字和这段话，写清各家擅长什么）">
                        <el-collapse class="w-full">
                            <el-collapse-item v-for="name in providerNames" :key="name" :name="name">
                                <template #title>
                                    <span class="text-[13px] text-[#475467]">{{ name }}</span>
                                    <el-tag v-if="form.profiles[name] !== router?.defaultProfiles[name]" size="small"
                                        effect="plain" class="ml-[8px]">已修改</el-tag>
                                </template>
                                <el-input v-model="form.profiles[name]" type="textarea"
                                    :autosize="{ minRows: 2, maxRows: 6 }" />
                                <el-button v-if="form.profiles[name] !== router?.defaultProfiles[name]" link
                                    type="primary" size="small" class="mt-[6px]"
                                    @click="form.profiles[name] = router?.defaultProfiles[name] ?? ''">恢复默认</el-button>
                            </el-collapse-item>
                        </el-collapse>
                    </el-form-item>

                    <div class="flex flex-wrap items-center gap-[12px] pt-[2px]">
                        <el-button type="primary" :loading="saving" @click="onSaveRouter">保存</el-button>
                        <span class="text-[12px] text-[#a8b0bd]">保存后立即生效，不需要重启</span>
                    </div>
                </el-form>

                <div class="flex flex-col gap-[14px]">
                    <div class="rounded-[12px] border px-[14px] py-[13px]" :class="statusView.box">
                        <p class="m-0 flex items-center gap-[7px] text-[13px] font-semibold" :class="statusView.titleClass">
                            <el-icon :size="15">
                                <component :is="statusView.icon" />
                            </el-icon>
                            {{ statusView.title }}
                        </p>
                        <div class="mt-[7px] flex flex-col gap-[2px]">
                            <span v-for="line in statusView.lines" :key="line"
                                class="text-[12px] leading-[1.7] text-[#667085]">{{ line }}</span>
                        </div>
                    </div>

                    <div class="rounded-[12px] border border-[#edf0f3] bg-white px-[14px] py-[13px]">
                        <p class="m-0 text-[13px] font-semibold text-[#1f2937]">测试判断</p>
                        <p class="mt-[4px] mb-[10px] text-[12px] leading-[1.6] text-[#98a2b3]">
                            用已保存的配置问一次，不发起真实搜索
                        </p>
                        <el-input v-model="probeQuery" placeholder="留空用示例搜索词" clearable
                            @keyup.enter="onTestRouter" />
                        <el-button class="mt-[10px] w-full" :loading="testing" :disabled="!router?.configured"
                            @click="onTestRouter">跑一次</el-button>

                        <div v-if="probe" class="mt-[14px] border-t border-[#f2f4f7] pt-[12px]">
                            <p class="m-0 text-[12px] leading-[1.75] text-[#475467]">
                                「{{ probe.query }}」→
                                <span class="font-semibold text-[#1f2937]">{{ probe.choice || '没有选择' }}</span>
                            </p>
                            <p class="mt-[3px] mb-0 text-[12px] leading-[1.75]">
                                <span v-if="probe.calibrated" class="text-[#667085]">
                                    置信度 {{ probe.confidence.toFixed(2) }}，
                                    {{ probe.accepted ? '会采用' : '低于阈值，不会采用' }}
                                </span>
                                <span v-else class="text-[#b54708]">接口没返回概率，阈值不生效，会直接采用</span>
                            </p>
                            <p class="mt-[2px] mb-0 text-[12px] text-[#a8b0bd]">
                                {{ probe.model }} · {{ probe.latencyMs }} ms · {{ probe.inputTokens }} tokens
                            </p>
                            <div v-for="[name, probability] in probeRanking" :key="name" class="mt-[10px]">
                                <div class="mb-[4px] flex items-center justify-between text-[12px] text-[#667085]">
                                    <span>{{ name }}</span>
                                    <span class="tabular-nums">{{ Math.round(probability * 100) }}%</span>
                                </div>
                                <el-progress :percentage="Math.round(probability * 100)" :stroke-width="6"
                                    :show-text="false" />
                            </div>
                        </div>
                    </div>
                </div>
            </div>
        </SectionPanel>
    </div>
</template>

<script setup lang="ts">
import { computed, markRaw, onMounted, reactive, ref } from 'vue';
import {
    CircleCheck,
    CircleCheckFilled,
    Histogram,
    InfoFilled,
    MagicStick,
    Sort,
    Switch,
    WarningFilled
} from '@element-plus/icons-vue';
import { notify } from '@/utils/notification';
import SectionPanel from './SectionPanel.vue';
import { formatTime } from './format';
import {
    getGatewaySettings,
    testModelRouter,
    updateGatewaySettings,
    type ModelRouterSettings,
    type ReasoningEffort,
    type RouterBackend,
    type RouterProbeResult,
    type RoutingStrategy,
    type StrategyOption
} from '@/api/gateway';

const strategy = ref<RoutingStrategy>('balanced');
const strategies = ref<StrategyOption[]>([]);
const router = ref<ModelRouterSettings | null>(null);
const loading = ref(false);

// Key 只往上写不往回读：表单里永远是空的，留空保存即保留原 Key
const form = reactive({
    backend: 'jev' as RouterBackend,
    endpoint: '',
    apiKey: '',
    clearApiKey: false,
    model: '',
    prompt: '',
    profiles: {} as Record<string, string>,
    minConfidence: 0.5,
    timeoutMs: 0,
    reasoningEffort: '' as ReasoningEffort
});
const saving = ref(false);
const probeQuery = ref('');
const testing = ref(false);
const probe = ref<RouterProbeResult | null>(null);

const activeStrategyLabel = computed(
    () => strategies.value.find((item) => item.value === strategy.value)?.label ?? strategy.value
);
const backendOption = computed(() => router.value?.backends.find((item) => item.value === form.backend));
const providerNames = computed(() => Object.keys(router.value?.defaultProfiles ?? {}).sort());
const keyPlaceholder = computed(() => {
    if (form.clearApiKey) return '保存后清除';
    if (router.value?.apiKeyMasked && form.backend === router.value.backend) return router.value.apiKeyMasked;
    return backendOption.value?.keyRequired ? '粘贴 API Key' : '可选，本地服务可不填';
});
const probeRanking = computed(() =>
    Object.entries(probe.value?.probabilities ?? {}).sort((left, right) => right[1] - left[1])
);

// 「不指定」放在最前，其余档位以后端下发的为准——页面上不自己维护那份列表
const effortChoices = computed(() => [
    { value: '' as ReasoningEffort, label: '不指定' },
    ...(router.value?.reasoningEfforts ?? []).map((item) => ({ value: item.value, label: item.label }))
]);
const effortHint = computed(() => {
    const choice = router.value?.reasoningEfforts.find((item) => item.value === form.reasoningEffort);
    if (!choice) return '不发送 reasoning_effort，是否思考由模型自己决定。';
    const wire = choice.wireValue === choice.value ? '' : `请求里写作 "${choice.wireValue}"。`;
    return `${choice.description}。${wire}`;
});

const statusView = computed(() => {
    const status = router.value?.status;
    if (status?.lastError) {
        return {
            title: '最近一次判断失败',
            lines: [status.lastError, formatTime(status.lastErrorAt)],
            icon: markRaw(WarningFilled),
            box: 'border-[#fecdca] bg-[#fef3f2]',
            titleClass: 'text-[#b42318]'
        };
    }
    if (status?.lastDecidedAt) {
        return {
            title: '最近一次判断',
            lines: [
                `选中 ${status.lastProvider || '（空）'}`,
                status.lastCalibrated
                    ? `置信度 ${status.lastConfidence.toFixed(2)}，${status.lastAccepted ? '已采用' : '低于阈值未采用'}`
                    : '接口没返回概率，阈值不生效，直接采用',
                `${status.lastModel || '未知模型'} · ${formatTime(status.lastDecidedAt)}`
            ],
            icon: markRaw(CircleCheck),
            box: 'border-[#abefc6] bg-[#ecfdf3]',
            titleClass: 'text-[#067647]'
        };
    }
    return {
        title: '还没有判断记录',
        lines: ['保存配置后点右边的「跑一次」，这里会显示它选了哪家、有多大把握。'],
        icon: markRaw(InfoFilled),
        box: 'border-[#e9eef5] bg-[#f8fafd]',
        titleClass: 'text-[#667085]'
    };
});

function strategyIcon(value: RoutingStrategy) {
    if (value === 'priority') return markRaw(Sort);
    if (value === 'model') return markRaw(MagicStick);
    return markRaw(Histogram);
}

function fillForm(settings: ModelRouterSettings) {
    Object.assign(form, {
        backend: settings.backend,
        endpoint: settings.endpoint,
        apiKey: '',
        clearApiKey: false,
        model: settings.model,
        prompt: settings.prompt,
        profiles: { ...settings.profiles },
        minConfidence: settings.minConfidence,
        reasoningEffort: settings.reasoningEffort,
        // 回显的是实际生效值；与默认值相同时显示 0，保存后仍跟随默认
        timeoutMs: settings.timeoutMs === settings.backends.find((item) => item.value === settings.backend)?.defaultTimeoutMs
            ? 0 : settings.timeoutMs
    });
}

async function load() {
    loading.value = true;
    try {
        const settings = await getGatewaySettings();
        strategy.value = settings.routingStrategy;
        strategies.value = settings.strategies;
        router.value = settings.modelRouter;
        fillForm(settings.modelRouter);
    } finally {
        loading.value = false;
    }
}

async function onSaveRouter() {
    saving.value = true;
    try {
        await updateGatewaySettings({ modelRouter: { ...form, profiles: { ...form.profiles } } });
        notify.success('判断模型配置已保存');
        await load();
    } finally {
        saving.value = false;
    }
}

async function onTestRouter() {
    if (!router.value?.configured) return;
    testing.value = true;
    try {
        probe.value = await testModelRouter(probeQuery.value);
        // 测试也会刷新「最近一次判断」，只更新状态，不覆盖正在编辑的表单
        const latest = (await getGatewaySettings()).modelRouter;
        if (router.value) router.value.status = latest.status;
    } finally {
        testing.value = false;
    }
}

// 未接入的方式在界面上就点不动，后端也会拒绝——否则等于显示的和实际执行的不是一回事
async function onPick(value: RoutingStrategy) {
    if (value === strategy.value) return;
    const option = strategies.value.find((item) => item.value === value);
    if (!option?.implemented) return;

    const previous = strategy.value;
    strategy.value = value;
    try {
        await updateGatewaySettings({ routingStrategy: value });
    } catch (error) {
        strategy.value = previous;
        throw error;
    }
    notify.success(`调度方式已切换为「${option.label}」`);
}

/*
 * 每个状态给一份完整的样式串，而不是拆成多个条件类。
 * 原因：Tailwind 无法靠 class 顺序覆盖同属性（border-color / background-color 的声明顺序由
 * 生成结果的排序决定），同属性写两处会随机有一处失效。锁定态优先判断。
 */
function strategyCardClass(option: StrategyOption) {
    const base = 'flex items-start gap-[12px] p-[14px] rounded-[12px] border transition-all duration-150';
    if (!option.implemented) {
        return `${base} cursor-not-allowed border-[#eef0f3] bg-[#fafbfc] opacity-70`;
    }
    if (option.value === strategy.value) {
        return `${base} cursor-pointer border-[#3f8cff] bg-[#f5f9ff] shadow-[0_0_0_3px_rgba(63,140,255,0.10)]`;
    }
    return `${base} cursor-pointer border-[#edf0f3] bg-white hover:border-[#b9d4ff] hover:shadow-[0_2px_10px_rgba(16,24,40,0.06)]`;
}

function effortButtonClass(value: ReasoningEffort) {
    if (form.reasoningEffort === value) {
        return 'cursor-pointer border-[#3f8cff] bg-[#f5f9ff] font-medium text-[#2563eb] shadow-[0_0_0_3px_rgba(63,140,255,0.10)]';
    }
    return 'cursor-pointer border-[#e6e9ee] bg-white text-[#475467] hover:border-[#b9d4ff] hover:text-[#3f8cff]';
}

onMounted(load);
</script>
