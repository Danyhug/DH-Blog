<template>
    <SectionPanel title="调度方式" subtitle="仅在请求未指定 provider（即 auto）时生效">
        <template #icon>
            <el-icon>
                <Switch />
            </el-icon>
        </template>

        <p class="mb-[16px] px-[12px] py-[10px] rounded-[8px] bg-[#f7f9fc] text-[12px] leading-[1.7] text-[#667085]">
            能力、健康、密钥与配额过滤始终优先——调度方式只决定通过过滤后的先后顺序，
            所以换策略不会让一个用不了的供应商被选中。
        </p>

        <div v-loading="loading" class="grid grid-cols-1 lg:grid-cols-3 gap-4">
            <button v-for="option in strategies" :key="option.value" type="button"
                class="flex flex-col gap-[8px] p-[16px] text-left border rounded-[12px] transition-all duration-200 ease-[ease]"
                :class="optionClass(option)" :disabled="!option.implemented" @click="onPick(option.value)">
                <span class="flex items-center gap-[8px]">
                    <span class="text-[14px] font-semibold text-[#1f2937]">{{ option.label }}</span>
                    <el-tag v-if="!option.implemented" size="small" type="info" effect="plain">开发中</el-tag>
                    <el-icon v-else-if="strategy === option.value" :size="17" color="#3f8cff" class="ml-auto">
                        <CircleCheckFilled />
                    </el-icon>
                </span>
                <span class="text-[12px] leading-[1.7] text-[#98a2b3]">{{ option.description }}</span>
            </button>
        </div>
    </SectionPanel>

    <SectionPanel title="判断模型" subtitle="「模型判断」用它挑首选供应商，每次未命中缓存的 auto 搜索调用一次">
        <template #icon>
            <el-icon>
                <MagicStick />
            </el-icon>
        </template>
        <template #extra>
            <el-tag v-if="router?.configured" size="small" type="success" effect="plain">已配置</el-tag>
            <el-tag v-else size="small" type="info" effect="plain">未配置</el-tag>
        </template>

        <el-form v-loading="loading" label-position="top" size="default">
            <el-form-item label="模型类型">
                <el-radio-group v-model="form.backend">
                    <el-radio-button v-for="option in router?.backends ?? []" :key="option.value" :value="option.value">
                        {{ option.label }}
                    </el-radio-button>
                </el-radio-group>
            </el-form-item>
            <p v-if="backendOption"
                class="mt-[-6px] mb-[16px] px-[12px] py-[10px] rounded-[8px] bg-[#f7f9fc] text-[12px] leading-[1.7] text-[#667085]">
                {{ backendOption.description }}
                <template v-if="form.backend === 'jev'">
                    Key 在 <el-link type="primary" href="https://console.typesafe.ai/" target="_blank"
                        class="align-baseline! text-[12px]!">TypeSafe 控制台</el-link> 获取。
                </template>
            </p>

            <el-row :gutter="14">
                <el-col :xs="24" :md="12">
                    <el-form-item label="接口地址">
                        <el-input v-model="form.endpoint" clearable
                            :placeholder="backendOption?.defaultEndpoint || backendOption?.endpointHint" />
                        <span class="text-[12px] leading-[1.6] text-[#a8b0bd]">{{ backendOption?.endpointHint }}</span>
                    </el-form-item>
                </el-col>
                <el-col :xs="24" :md="12">
                    <el-form-item label="模型">
                        <el-input v-model="form.model" clearable
                            :placeholder="backendOption?.defaultModel || '例如 deepseek-chat、qwen-turbo'" />
                    </el-form-item>
                </el-col>
            </el-row>

            <el-form-item>
                <template #label>
                    API Key
                    <span class="ml-[6px] text-[12px] font-normal text-[#a8b0bd]">
                        留空表示不修改；切换类型或改了接口地址后需要重新填写，否则旧 Key 会被清空
                    </span>
                </template>
                <div class="flex w-full gap-[10px]">
                    <el-input v-model="form.apiKey" show-password clearable :disabled="form.clearApiKey"
                        :placeholder="keyPlaceholder" />
                    <el-button v-if="router?.apiKeyMasked && !form.clearApiKey" @click="form.clearApiKey = true">
                        清除
                    </el-button>
                    <el-button v-if="form.clearApiKey" @click="form.clearApiKey = false">撤销清除</el-button>
                </div>
            </el-form-item>

            <el-row :gutter="14">
                <el-col :xs="24" :md="12">
                    <el-form-item label="置信度阈值（0–1，低于它这次按负载均衡）">
                        <el-input-number v-model="form.minConfidence" :min="0" :max="1" :step="0.05" :precision="2"
                            class="w-full" />
                    </el-form-item>
                </el-col>
                <el-col :xs="24" :md="12">
                    <el-form-item :label="`超时（毫秒，0 用默认值 ${backendOption?.defaultTimeoutMs ?? '-'}）`">
                        <el-input-number v-model="form.timeoutMs" :min="0" :max="30000" :step="500" class="w-full" />
                    </el-form-item>
                </el-col>
            </el-row>

            <el-form-item>
                <template #label>
                    提示词
                    <span class="ml-[6px] text-[12px] font-normal text-[#a8b0bd]">
                        只写判断要求；候选列表与回答格式由网关自动附加。英文效果更稳
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
                        <el-input v-model="form.profiles[name]" type="textarea" :autosize="{ minRows: 2, maxRows: 6 }" />
                        <el-button v-if="form.profiles[name] !== router?.defaultProfiles[name]" link type="primary"
                            size="small" class="mt-[6px]"
                            @click="form.profiles[name] = router?.defaultProfiles[name] ?? ''">恢复默认</el-button>
                    </el-collapse-item>
                </el-collapse>
            </el-form-item>

            <el-button type="primary" :loading="saving" @click="onSaveRouter">保存</el-button>
        </el-form>

        <div v-if="router?.status.lastDecidedAt || router?.status.lastError"
            class="mt-[16px] text-[12px] leading-[1.7] text-[#667085]">
            <p v-if="router.status.lastError" class="m-0 text-[#d92d20]">
                最近一次调用失败（{{ formatTime(router.status.lastErrorAt) }}）：{{ router.status.lastError }}
            </p>
            <p v-else class="m-0">
                最近一次判断（{{ formatTime(router.status.lastDecidedAt) }}）：{{ router.status.lastProvider }}，
                <template v-if="router.status.lastCalibrated">
                    置信度 {{ router.status.lastConfidence.toFixed(2) }}，{{ router.status.lastAccepted ? '已采用' : '低于阈值未采用' }}
                </template>
                <template v-else>模型没有返回概率，阈值不生效，已直接采用</template>
                <span class="text-[#98a2b3]">· {{ router.status.lastModel }}</span>
            </p>
        </div>

        <el-divider />

        <div class="flex flex-col sm:flex-row gap-[10px]">
            <el-input v-model="probeQuery" placeholder="输入一个搜索词，看模型会交给哪家（留空用示例）" clearable
                @keyup.enter="onTestRouter" />
            <el-button :loading="testing" :disabled="!router?.configured" @click="onTestRouter">测试判断</el-button>
        </div>
        <div v-if="probe" class="mt-[14px] flex flex-col gap-[8px]">
            <p class="m-0 text-[12px] text-[#667085]">
                「{{ probe.query }}」→ <span class="font-semibold text-[#1f2937]">{{ probe.choice }}</span>，
                <template v-if="probe.calibrated">
                    置信度 {{ probe.confidence.toFixed(2) }}（{{ probe.accepted ? '会采用' : '低于阈值，不会采用' }}）
                </template>
                <template v-else>没有概率（接口不支持 logprobs），会直接采用</template>
                <span class="text-[#98a2b3]">· {{ probe.model }} · {{ probe.latencyMs }} ms · {{ probe.inputTokens }} tokens</span>
            </p>
            <div v-for="[name, probability] in probeRanking" :key="name" class="flex items-center gap-[10px]">
                <span class="w-[80px] shrink-0 text-[12px] text-[#475467]">{{ name }}</span>
                <el-progress class="flex-1" :percentage="Math.round(probability * 100)" :stroke-width="8" />
            </div>
        </div>
    </SectionPanel>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue';
import { CircleCheckFilled, MagicStick, Switch } from '@element-plus/icons-vue';
import { notify } from '@/utils/notification';
import SectionPanel from './SectionPanel.vue';
import { formatTime } from './format';
import {
    getGatewaySettings,
    testModelRouter,
    updateGatewaySettings,
    type ModelRouterSettings,
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
    timeoutMs: 0
});
const saving = ref(false);
const probeQuery = ref('');
const testing = ref(false);
const probe = ref<RouterProbeResult | null>(null);

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

/**
 * 每个状态给一份完整的样式串，而不是拆成多个条件类。
 * 原因：Tailwind 无法靠 class 顺序覆盖同属性（border-color / background-color 的声明顺序由
 * 生成结果的排序决定），同属性写两处会随机有一处失效。锁定态优先判断，与原 CSS 中
 * `.option--locked` 排在 `.option--active` 之后的实际效果一致。
 */
function optionClass(option: StrategyOption) {
    if (!option.implemented) {
        return 'cursor-not-allowed border-[#e6e9ee] bg-[#fafafa] opacity-70';
    }
    if (option.value === strategy.value) {
        return 'cursor-pointer border-[#3f8cff] bg-[#f5f9ff] shadow-[0_0_0_3px_rgba(63,140,255,0.08)]';
    }
    return 'cursor-pointer border-[#e6e9ee] bg-white hover:border-[#b9d4ff]';
}

onMounted(load);
</script>
