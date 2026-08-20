<template>
    <div class="p-5">
        <el-tabs v-model="activeTab">
            <el-tab-pane name="overview" lazy>
                <template #label>
                    <span class="tab-label">
                        <el-icon>
                            <DataAnalysis />
                        </el-icon> 概览
                    </span>
                </template>
                <GatewayOverview :providers="providers" />
            </el-tab-pane>

            <el-tab-pane name="providers" lazy>
                <template #label>
                    <span class="tab-label">
                        <el-icon>
                            <Connection />
                        </el-icon> 供应商
                    </span>
                </template>
                <GatewayProviders :providers="providers" @refresh="loadProviders" />
            </el-tab-pane>

            <el-tab-pane name="routing" lazy>
                <template #label>
                    <span class="tab-label">
                        <el-icon>
                            <Switch />
                        </el-icon> 调度策略
                    </span>
                </template>
                <GatewayRouting />
            </el-tab-pane>

            <el-tab-pane name="mcp" lazy>
                <template #label>
                    <span class="tab-label">
                        <el-icon>
                            <Cpu />
                        </el-icon> MCP 能力
                    </span>
                </template>
                <GatewayMcp />
            </el-tab-pane>

            <el-tab-pane name="keys" lazy>
                <template #label>
                    <span class="tab-label">
                        <el-icon>
                            <Key />
                        </el-icon> 接入密钥
                    </span>
                </template>
                <GatewayKeys :providers="providers" />
            </el-tab-pane>

            <el-tab-pane name="logs" lazy>
                <template #label>
                    <span class="tab-label">
                        <el-icon>
                            <Document />
                        </el-icon> 请求日志
                    </span>
                </template>
                <GatewayLogs :providers="providers" />
            </el-tab-pane>
        </el-tabs>
    </div>
</template>

<script setup lang="ts">
import { onMounted, ref, watch } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import { Connection, Cpu, DataAnalysis, Document, Key, Switch } from '@element-plus/icons-vue';
import GatewayOverview from '@/components/backend/gateway/GatewayOverview.vue';
import GatewayProviders from '@/components/backend/gateway/GatewayProviders.vue';
import GatewayRouting from '@/components/backend/gateway/GatewayRouting.vue';
import GatewayMcp from '@/components/backend/gateway/GatewayMcp.vue';
import GatewayKeys from '@/components/backend/gateway/GatewayKeys.vue';
import GatewayLogs from '@/components/backend/gateway/GatewayLogs.vue';
import { getGatewayProviders, type GatewayProvider } from '@/api/gateway';

const route = useRoute();
const router = useRouter();

const tabs = ['overview', 'providers', 'routing', 'mcp', 'keys', 'logs'];
const activeTab = ref(tabs.includes(String(route.query.tab)) ? String(route.query.tab) : 'overview');

// 标签写回地址栏，刷新页面或把链接发给自己时还停在同一屏
watch(activeTab, (tab) => {
    router.replace({ query: { ...route.query, tab } });
});

// 供应商列表是几个标签页共用的底料：概览要展示名与 logo，密钥要下拉选项，日志要筛选项
const providers = ref<GatewayProvider[]>([]);

async function loadProviders() {
    providers.value = await getGatewayProviders();
}

onMounted(loadProviders);
</script>

<style scoped>
.tab-label {
    display: inline-flex;
    align-items: center;
    gap: 6px;
}

:deep(.el-tabs__header) {
    margin-bottom: 20px;
}

:deep(.el-tabs__nav-wrap::after) {
    height: 1px;
    background-color: #e9edf2;
}

:deep(.el-tabs__item) {
    font-size: 14px;
    padding: 0 20px;
    color: #667085;
}

:deep(.el-tabs__item.is-active) {
    font-weight: 600;
    color: #3f8cff;
}

:deep(.el-input-number) {
    width: 100%;
}

:deep(.el-input-number .el-input__inner) {
    text-align: left;
}

/* 表格在这些面板里是内容而不是控件，去掉默认的重边框让整页更干净 */
:deep(.el-table th.el-table__cell) {
    background-color: #fbfcfd;
    color: #667085;
    font-weight: 500;
}

:deep(.el-table) {
    --el-table-border-color: #f2f4f7;
}

:deep(code) {
    padding: 1px 6px;
    border-radius: 4px;
    background-color: #f4f6f8;
    color: #476582;
    font-size: 12px;
}
</style>
