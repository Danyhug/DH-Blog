<template>
    <div class="system-view">
        <!-- 删除页面头部（面包屑和标题）部分 -->

        <el-tabs v-model="activeTab" class="system-tabs">
            <el-tab-pane label="站点设置" name="site">
                <el-card shadow="hover" class="settings-card">
                    <template #header>
                        <div class="card-header">
                            <span><el-icon><Setting /></el-icon> 基本信息设置</span>
                        </div>
                    </template>
                    <el-form :model="blogConfig" label-position="top" size="default">
                        <el-row :gutter="24">
                            <el-col :span="12">
                                <el-form-item label="博客标题">
                                    <el-input 
                                        v-model="blogConfig.blog_title" 
                                        placeholder="请输入博客标题"
                                        clearable
                                        :prefix-icon="Edit"
                                    ></el-input>
                                </el-form-item>
                            </el-col>
                            <el-col :span="12">
                                <el-form-item label="个人签名">
                                    <el-input 
                                        v-model="blogConfig.signature" 
                                        placeholder="请输入个人签名"
                                        clearable
                                        :prefix-icon="Edit"
                                    ></el-input>
                                </el-form-item>
                            </el-col>
                        </el-row>
                        
                        <el-row :gutter="24">
                            <el-col :span="12">
                                <el-form-item label="个人头像">
                                    <div class="avatar-uploader-container">
                                        <div class="avatar-preview" v-if="blogConfig.avatar">
                                            <img :src="blogConfig.avatar" class="avatar-image" />
                                        </div>
                                        <el-input 
                                            v-model="blogConfig.avatar" 
                                            placeholder="请输入头像URL或上传图片"
                                            clearable
                                            :prefix-icon="Picture"
                                        ></el-input>
                                    </div>
                                </el-form-item>
                            </el-col>
                        </el-row>
                        
                        <el-divider content-position="left">社交链接</el-divider>
                        
                        <el-row :gutter="24">
                            <el-col :span="12">
                                <el-form-item label="GitHub链接">
                                    <el-input 
                                        v-model="blogConfig.github_link" 
                                        placeholder="请输入GitHub链接"
                                        clearable
                                    >
                                        <template #prefix>
                                            <el-icon><Link /></el-icon>
                                        </template>
                                    </el-input>
                                </el-form-item>
                            </el-col>
                            <el-col :span="12">
                                <el-form-item label="Bilibili链接">
                                    <el-input 
                                        v-model="blogConfig.bilibili_link" 
                                        placeholder="请输入Bilibili链接"
                                        clearable
                                    >
                                        <template #prefix>
                                            <el-icon><Link /></el-icon>
                                        </template>
                                    </el-input>
                                </el-form-item>
                            </el-col>
                        </el-row>
                    </el-form>
                </el-card>
            </el-tab-pane>
            
            <el-tab-pane label="功能设置" name="features">
                <el-card shadow="hover" class="settings-card">
                    <template #header>
                        <div class="card-header">
                            <span><el-icon><Setting /></el-icon> 功能开关设置</span>
                        </div>
                    </template>
                    <el-form :model="config" label-position="top" size="default">
                        <el-row :gutter="24">
                            <el-col :span="8">
                                <el-form-item label="开放博客">
                                    <div class="switch-container">
                                        <el-switch
                                            v-model="config.open_blog"
                                            active-color="#13ce66"
                                            inactive-color="#ff4949">
                                        </el-switch>
                                        <div class="form-item-tip">开启后博客将对外可访问</div>
                                    </div>
                                </el-form-item>
                            </el-col>
                            <el-col :span="8">
                                <el-form-item label="开放评论">
                                    <div class="switch-container">
                                        <el-switch
                                            v-model="config.open_comment"
                                            active-color="#13ce66"
                                            inactive-color="#ff4949">
                                        </el-switch>
                                        <div class="form-item-tip">开启后访客可以评论文章</div>
                                    </div>
                                </el-form-item>
                            </el-col>
                            <el-col :span="8">
                                <el-form-item label="评论邮件通知">
                                    <div class="switch-container">
                                        <el-switch
                                            v-model="config.comment_email_notify"
                                            active-color="#13ce66"
                                            inactive-color="#ff4949">
                                        </el-switch>
                                        <div class="form-item-tip">开启后收到评论将发送邮件通知</div>
                                    </div>
                                </el-form-item>
                            </el-col>
                        </el-row>
                    </el-form>
                </el-card>
            </el-tab-pane>
            
            <el-tab-pane label="邮箱设置" name="email">
                <el-card shadow="hover" class="settings-card">
                    <template #header>
                        <div class="card-header">
                            <span><el-icon><Message /></el-icon> 邮件服务设置</span>
                        </div>
                    </template>
                    <el-form :model="emailConfig" label-position="top" size="default">
                        <el-row :gutter="24">
                            <el-col :span="12">
                                <el-form-item label="SMTP主机">
                                    <el-input 
                                        v-model="emailConfig.smtp_host" 
                                        placeholder="例如: smtp.gmail.com"
                                        clearable
                                        :prefix-icon="Connection"
                                    ></el-input>
                                </el-form-item>
                            </el-col>
                            <el-col :span="12">
                                <el-form-item label="SMTP端口">
                                    <el-input-number
                                        v-model="emailConfig.smtp_port"
                                        :min="1"
                                        :max="65535"
                                        placeholder="例如: 587"
                                        style="width: 100%"
                                    ></el-input-number>
                                </el-form-item>
                            </el-col>
                        </el-row>
                        
                        <el-row :gutter="24">
                            <el-col :span="12">
                                <el-form-item label="SMTP用户">
                                    <el-input 
                                        v-model="emailConfig.smtp_user" 
                                        placeholder="请输入邮箱账号"
                                        clearable
                                        :prefix-icon="User"
                                    ></el-input>
                                </el-form-item>
                            </el-col>
                            <el-col :span="12">
                                <el-form-item label="SMTP密码">
                                    <el-input 
                                        v-model="emailConfig.smtp_pass" 
                                        type="password" 
                                        placeholder="请输入邮箱密码或授权码"
                                        show-password
                                        :prefix-icon="Lock"
                                    ></el-input>
                                </el-form-item>
                            </el-col>
                        </el-row>
                        
                        <el-row :gutter="24">
                            <el-col :span="12">
                                <el-form-item label="SMTP发送者">
                                    <el-input 
                                        v-model="emailConfig.smtp_sender" 
                                        placeholder="发送者名称"
                                        clearable
                                        :prefix-icon="User"
                                    ></el-input>
                                </el-form-item>
                            </el-col>
                        </el-row>
                        
                        <el-row>
                            <el-col :span="24">
                                <div class="test-email-container">
                                    <el-button type="primary" size="default">
                                        <el-icon><Message /></el-icon> 测试邮件发送
                                    </el-button>
                                    <span class="test-email-tip">点击按钮发送测试邮件到当前SMTP用户邮箱</span>
                                </div>
                            </el-col>
                        </el-row>
                    </el-form>
                </el-card>
            </el-tab-pane>
            
            <el-tab-pane label="文件存储" name="storage">
                <el-card shadow="hover" class="settings-card">
                    <template #header>
                        <div class="card-header">
                            <span><el-icon><Folder /></el-icon> 文件存储设置</span>
                        </div>
                    </template>
                    <el-form :model="storageConfig" label-position="top" size="default">
                        <el-form-item label="存储路径">
                            <el-input 
                                v-model="storageConfig.file_storage_path" 
                                placeholder="/path/to/your/storage/directory"
                                :prefix-icon="Folder"
                            >
                                <template #append>
                                    <el-button @click="openDirectorySelector">
                                        <el-icon><FolderOpened /></el-icon> 选择路径
                                    </el-button>
                                </template>
                            </el-input>
                            <div class="info-text">
                                <el-icon><InfoFilled /></el-icon>
                                请设置一个服务器上有读写权限的绝对路径，用于存储上传的文件。保存后即时生效，不需要重启服务器。
                            </div>
                            <div class="warning-text">
                                <el-icon><WarningFilled /></el-icon>
                                注意：更改存储路径会清空文件表，所有文件记录将被删除并重新扫描！
                            </div>
                        </el-form-item>
                        
                        <el-form-item label="WebDAV分片大小">
                            <div class="config-item-container">
                                <el-input-number 
                                    v-model="storageConfig.webdav_chunk_size" 
                                    :min="1024" 
                                    :max="102400"
                                    :step="1024"
                                    controls-position="both"
                                    style="width: 120px"
                                    size="large"
                                ></el-input-number>
                                <span class="unit-text">KB</span>
                                <span class="size-hint">({{ storageConfig.webdav_chunk_size ? (Math.round(storageConfig.webdav_chunk_size / 1024 * 100) / 100) : '5.00' }} MB)</span>
                            </div>
                            <div class="info-text enhanced">
                                <el-icon><InfoFilled /></el-icon>
                                <div class="info-content">
                                    <p><strong>推荐范围：</strong>5MB - 10MB (5120KB - 10240KB)</p>
                                    <p><strong>大文件建议：</strong>较大的分片可以提高大文件上传效率，减少请求次数</p>
                                    <p><strong>小文件建议：</strong>较小的分片适合小文件上传，避免资源浪费</p>
                                </div>
                            </div>
                        </el-form-item>
                    </el-form>
                    
                    <!-- 目录选择对话框 -->
                    <el-dialog
                        v-model="directoryDialogVisible"
                        title="选择存储路径"
                        width="60%"
                        destroy-on-close
                    >
                        <div class="directory-selector">
                            <div class="directory-path">
                                <el-tag type="info" size="default">当前路径: {{ currentPath || '/' }}</el-tag>
                                <el-button size="default" @click="loadParentDirectory" :disabled="!currentPath">
                                    <el-icon><Back /></el-icon> 上一级
                                </el-button>
                            </div>
                            
                            <el-tree
                                :data="directoryTree"
                                node-key="path"
                                :props="{
                                    label: 'name',
                                    children: 'children',
                                    isLeaf: (data: any) => !data.isDir
                                }"
                                @node-click="handleNodeClick"
                                :load="loadNode"
                                lazy
                                :default-expanded-keys="expandedKeys"
                                class="directory-tree"
                            >
                                <template #default="{ node, data }">
                                    <span class="custom-tree-node">
                                        <span>
                                            <el-icon>
                                                <component :is="data.isDir ? Folder : Document" />
                                            </el-icon>
                                            {{ node.label }}
                                        </span>
                                        <el-button
                                            v-if="data.isDir"
                                            size="small"
                                            type="primary"
                                            plain
                                            @click.stop="selectDirectory(data.path)"
                                        >
                                            选择
                                        </el-button>
                                    </span>
                                </template>
                            </el-tree>
                        </div>
                        <template #footer>
                            <span class="dialog-footer">
                                <el-button @click="directoryDialogVisible = false">取消</el-button>
                                <el-button type="primary" @click="confirmDirectorySelection">
                                    确认
                                </el-button>
                            </span>
                        </template>
                    </el-dialog>
                </el-card>
            </el-tab-pane>
            
            <el-tab-pane label="AI设置" name="ai">
                <el-card shadow="hover" class="settings-card">
                    <template #header>
                        <div class="card-header">
                            <span><el-icon><Cpu /></el-icon> AI服务设置</span>
                        </div>
                    </template>
                    <el-form :model="aiConfig" label-position="top" size="default">
                        <el-row :gutter="24">
                            <el-col :span="8">
                                <el-form-item label="API 地址">
                                    <el-input 
                                        v-model="aiConfig.ai_api_url" 
                                        placeholder="https://xxx.xin/v1/chat/completions"
                                        clearable
                                        :prefix-icon="Connection"
                                    ></el-input>
                                </el-form-item>
                            </el-col>
                            <el-col :span="10">
                                <el-form-item label="API 秘钥">
                                    <el-input 
                                        v-model="aiConfig.ai_api_key" 
                                        show-password
                                        :prefix-icon="Key"
                                        placeholder="请输入API秘钥"
                                    ></el-input>
                                </el-form-item>
                            </el-col>
                            <el-col :span="6">
                                <el-form-item label="模型">
                                    <el-input 
                                        v-model="aiConfig.ai_model" 
                                        placeholder="gpt-3.5-turbo"
                                        clearable
                                        :prefix-icon="Cpu"
                                    ></el-input>
                                </el-form-item>
                            </el-col>
                        </el-row>
                        
                        <el-divider content-position="left">提示词设置</el-divider>
                        
                        <el-form-item label="提示词选择">
                            <div class="prompt-tags-container">
                                <el-tag
                                    v-for="tag in promptTags"
                                    :key="tag.label"
                                    class="prompt-tag"
                                    :class="{ 'is-active': selectedPromptLabel === tag.label }"
                                    effect="light"
                                    round
                                    @click="selectPrompt(tag)"
                                >
                                    <el-icon><MagicStick /></el-icon> {{ tag.label }}
                                </el-tag>
                            </div>
                        </el-form-item>

                        <el-form-item label="提示词内容" v-if="selectedPrompt">
                            <div class="prompt-editor-container">
                                <div v-show="!isEditingPrompt" class="ai-prompt-display" @click="startEditing">
                                    <div v-html="highlightedPrompt"></div>
                                    <el-button class="edit-button" type="primary" link>
                                        <el-icon><Edit /></el-icon>
                                        编辑
                                    </el-button>
                                </div>
                                <el-input
                                    v-show="isEditingPrompt"
                                    ref="promptInputRef"
                                    v-model="selectedPrompt.prompt"
                                    type="textarea"
                                    :autosize="{ minRows: 4 }"
                                    class="prompt-textarea"
                                    @blur="isEditingPrompt = false"
                                ></el-input>
                            </div>
                        </el-form-item>
                        
                    </el-form>
                </el-card>
            </el-tab-pane>

            <el-tab-pane label="系统配置" name="settings">
                <el-card shadow="hover" class="settings-card">
                    <template #header>
                        <div class="card-header">
                            <span><el-icon><Setting /></el-icon> 系统配置管理</span>
                        </div>
                    </template>
                    <el-table :data="systemSettings" style="width: 100%" size="default">
                        <el-table-column prop="settingKey" label="Key" min-width="180" />
                        <el-table-column prop="settingValue" label="Value" min-width="220" />
                        <el-table-column prop="configType" label="类型" min-width="120" />
                        <el-table-column label="操作" width="160">
                            <template #default="scope">
                                <el-button size="small" @click="onEditSetting(scope.row)">编辑</el-button>
                                <el-button size="small" type="danger" @click="onDeleteSetting(scope.row)">删除</el-button>
                            </template>
                        </el-table-column>
                    </el-table>
                    <div style="margin-top: 16px; text-align: right;">
                        <el-button type="primary" @click="onAddSetting">新增配置</el-button>
                    </div>
                </el-card>
            </el-tab-pane>
        </el-tabs>
        
        <div class="action-buttons">
            <el-button @click="activeTab = 'site'" size="default">取消</el-button>
            <el-button type="primary" @click="saveConfig" size="default">保存设置</el-button>
        </div>

        <el-dialog v-model="settingDialogVisible" :title="settingDialogMode === 'add' ? '新增配置项' : '编辑配置项'" width="400px" @close="settingFormRef?.resetFields()">
            <el-form :model="settingForm" :rules="settingFormRules" ref="settingFormRef" label-width="80px" size="default">
                <el-form-item label="Key" prop="settingKey">
                    <el-input v-model="settingForm.settingKey" :disabled="settingDialogMode==='edit'" clearable></el-input>
                </el-form-item>
                <el-form-item label="Value" prop="settingValue">
                    <el-input v-model="settingForm.settingValue" clearable></el-input>
                </el-form-item>
                <el-form-item label="类型">
                    <el-input v-model="settingForm.configType" clearable></el-input>
                </el-form-item>
            </el-form>
            <template #footer>
                <el-button @click="settingDialogVisible = false">取消</el-button>
                <el-button type="primary" @click="onSettingDialogOk">确定</el-button>
            </template>
        </el-dialog>
    </div>
</template>

<script setup lang="ts">
import { ref, onMounted, computed, watch, nextTick } from 'vue';
import { 
    getSystemConfig, updateSystemConfig, 
    getBlogConfig, updateBlogConfig,
    getEmailConfig, updateEmailConfig,
    getAIConfig, updateAIConfig,
    getStorageConfig, updateStorageConfig,
    getSystemSettings, addSystemSetting, updateSystemSetting, deleteSystemSetting,
    getAIPromptTags // 导入获取AI提示词标签的API
} from '@/api/admin';
import { getDirectoryTree } from '@/api/file';
import type { SystemConfig, BlogConfig, EmailConfig, AIConfig, StorageConfig } from '@/types/SystemConfig';
import { ElMessage, ElMessageBox } from 'element-plus';
// 导入 Element Plus 图标
import { 
    Edit, Picture, Link, Connection, User, Lock, Key, 
    Folder, FolderOpened, Back, Document, InfoFilled, 
    WarningFilled, Message, Setting, Cpu, MagicStick
} from '@element-plus/icons-vue';

// HTML 转义函数
const escapeHtml = (unsafe: string) => {
    return unsafe
        .replace(/&/g, "&amp;")
        .replace(/</g, "&lt;")
        .replace(/>/g, "&gt;")
        .replace(/"/g, "&quot;")
        .replace(/'/g, "&#039;");
};

const activeTab = ref('site');
const config = ref<SystemConfig>({});
const blogConfig = ref<BlogConfig>({});
const emailConfig = ref<EmailConfig>({});
const aiConfig = ref<AIConfig>({});
const storageConfig = ref<StorageConfig>({ webdav_chunk_size: 5120 }); // 默认5MB
const isEditingPrompt = ref(false);
const systemSettings = ref<any[]>([]);

const selectedPrompt = ref<{ label: string, prompt: string } | null>(null);
const selectedPromptLabel = ref('');
const promptInputRef = ref();

const settingDialogVisible = ref(false);
const settingDialogMode = ref<'add' | 'edit'>('add');
const settingForm = ref<{ id: number | undefined, settingKey: string, settingValue: string, configType?: string }>({ id: undefined, settingKey: '', settingValue: '', configType: '' });
const settingFormRules = {
    settingKey: [{ required: true, message: '请输入Key', trigger: 'blur' }],
    settingValue: [{ required: true, message: '请输入Value', trigger: 'blur' }],
};
const settingFormRef = ref();

function onAddSetting() {
    settingDialogMode.value = 'add';
    settingForm.value = { id: undefined, settingKey: '', settingValue: '', configType: '' };
    settingDialogVisible.value = true;
}
function onEditSetting(row: any) {
    settingDialogMode.value = 'edit';
    settingForm.value = { ...row };
    settingDialogVisible.value = true;
}
async function onDeleteSetting(row: any) {
    ElMessageBox.confirm('确定要删除该配置项吗？', '提示', { type: 'warning' })
        .then(async () => {
            await deleteSystemSetting(row.id);
            ElMessage.success('删除成功');
            loadSystemSettings();
        })
        .catch(() => {});
}
async function onSettingDialogOk() {
    // 校验表单
    const valid = await settingFormRef.value.validate().catch(() => false);
    if (!valid) return;
    if (settingDialogMode.value === 'add') {
        await addSystemSetting(settingForm.value);
        ElMessage.success('新增成功');
    } else {
        await updateSystemSetting(settingForm.value);
        ElMessage.success('更新成功');
    }
    settingDialogVisible.value = false;
    loadSystemSettings();
}

// 目录选择相关
const directoryDialogVisible = ref(false);
const directoryTree = ref<any[]>([]);
const currentPath = ref('');
const selectedPath = ref('');
const expandedKeys = ref<string[]>([]);

const promptTags = ref<{ label: string, prompt: string }[]>([]);

const highlightedPrompt = computed(() => {
    if (!selectedPrompt.value) return '';
    let processedPrompt = escapeHtml(selectedPrompt.value.prompt);
    return processedPrompt.replace(/\{\{\.[a-zA-Z0-9_]+\}\}/g, '<span class="text-highlight">$&</span>');
});

const startEditing = async () => {
    isEditingPrompt.value = true;
    await nextTick();
    promptInputRef.value?.focus();
};

const selectPrompt = (tag: { label: string, prompt: string }) => {
    selectedPrompt.value = { ...tag };
    selectedPromptLabel.value = tag.label;
    isEditingPrompt.value = false;
};

// 打开目录选择器
const openDirectorySelector = async () => {
    directoryDialogVisible.value = true;
    currentPath.value = '';
    selectedPath.value = '';
    expandedKeys.value = [];
    
    try {
        const res = await getDirectoryTree();
        directoryTree.value = [res];
    } catch (error) {
        ElMessage.error('获取目录树失败');
    }
};

// 加载节点的子节点
const loadNode = async (node: any, resolve: (data: any[]) => void) => {
    if (node.level === 0) {
        resolve(directoryTree.value);
        return;
    }
    
    try {
        const res = await getDirectoryTree(node.data.path, 1);
        if (res && res.children) {
            resolve(res.children);
        } else {
            resolve([]);
        }
    } catch (error) {
        ElMessage.error('加载子目录失败');
        resolve([]);
    }
};

// 处理节点点击
const handleNodeClick = (data: { isDir: boolean, path: string }) => {
    if (data.isDir) {
        currentPath.value = data.path;
    }
};

// 加载父目录
const loadParentDirectory = async () => {
    if (!currentPath.value) return;
    
    const parentPath = currentPath.value.substring(0, currentPath.value.lastIndexOf('/'));
    try {
        const res = await getDirectoryTree(parentPath, 1);
        directoryTree.value = [res];
        currentPath.value = parentPath || '';
    } catch (error) {
        ElMessage.error('加载父目录失败');
    }
};

// 选择目录
const selectDirectory = (path: string) => {
    selectedPath.value = path;
};

// 确认目录选择
const confirmDirectorySelection = () => {
    if (selectedPath.value) {
        storageConfig.value.file_storage_path = selectedPath.value;
    }
    directoryDialogVisible.value = false;
};

// 系统配置项加载与刷新
async function loadSystemSettings() {
    try {
        const res = await getSystemSettings();
        systemSettings.value = res;
    } catch (e) {
        ElMessage.error('获取系统配置失败');
    }
}

// 监听tab切换，切到系统配置时加载
watch(activeTab, (tab) => {
    if (tab === 'settings') {
        loadSystemSettings();
    }
});

// 监听选项卡变化，按需加载不同类型的配置
watch(activeTab, async (newTab) => {
    if (newTab === 'site') {
        await loadBlogConfig();
    } else if (newTab === 'email') {
        await loadEmailConfig();
    } else if (newTab === 'storage') {
        await loadStorageConfig();
    } else if (newTab === 'ai') {
        await loadAIConfig();
        await loadAIPromptTags(); // 加载AI提示词标签
    }
});

// 加载博客基本配置
const loadBlogConfig = async () => {
    try {
        const res = await getBlogConfig();
        blogConfig.value = res;
    } catch (error) {
        ElMessage.error('加载博客配置失败');
    }
};

// 加载邮件配置
const loadEmailConfig = async () => {
    try {
        const res = await getEmailConfig();
        emailConfig.value = res;
    } catch (error) {
        ElMessage.error('加载邮件配置失败');
    }
};

// 加载AI配置
const loadAIConfig = async () => {
    try {
        const res = await getAIConfig();
        aiConfig.value = res;
    } catch (error) {
        ElMessage.error('加载AI配置失败');
    }
};

// 加载存储配置
const loadStorageConfig = async () => {
    try {
        const res = await getStorageConfig();
        storageConfig.value = {
            ...res,
            webdav_chunk_size: res.webdav_chunk_size || 5120
        };
    } catch (error) {
        ElMessage.error('加载存储配置失败');
    }
};

// 加载AI提示词标签
const loadAIPromptTags = async () => {
    try {
        const res = await getAIPromptTags();
        promptTags.value = res;
        // 默认选中第一个
        if (res.length > 0) {
            selectPrompt(res[0]);
        }
    } catch (error) {
        ElMessage.error('加载AI提示词标签失败');
    }
};

// 保存配置
const saveConfig = async () => {
    try {
        if (activeTab.value === 'site') {
            await updateBlogConfig(blogConfig.value);
        } else if (activeTab.value === 'email') {
            await updateEmailConfig(emailConfig.value);
        } else if (activeTab.value === 'storage') {
            // 直接更新存储配置，不显示确认弹窗
            await updateStorageConfig(storageConfig.value);
        } else if (activeTab.value === 'ai') {
            await updateAIConfig(aiConfig.value);
        }
        ElMessage.success('保存成功');
    } catch (error) {
        ElMessage.error('保存失败');
    }
};

onMounted(async () => {
    // 首先加载全局配置
    const res = await getSystemConfig();
    config.value = res;
    
    // 初始化各个分类配置
    blogConfig.value = {
        blog_title: res.blog_title,
        signature: res.signature,
        avatar: res.avatar,
        github_link: res.github_link,
        bilibili_link: res.bilibili_link,
        open_blog: res.open_blog,
        open_comment: res.open_comment
    };
    
    emailConfig.value = {
        comment_email_notify: res.comment_email_notify,
        smtp_host: res.smtp_host,
        smtp_port: res.smtp_port,
        smtp_user: res.smtp_user,
        smtp_pass: res.smtp_pass,
        smtp_sender: res.smtp_sender
    };
    
    aiConfig.value = {
        ai_api_url: res.ai_api_url,
        ai_api_key: res.ai_api_key,
        ai_model: res.ai_model,
    };
    
    storageConfig.value = {
        file_storage_path: res.file_storage_path,
        webdav_chunk_size: res.webdav_chunk_size || 5120
    };
    
    // 根据当前选项卡加载对应配置
    if (activeTab.value === 'site') {
        await loadBlogConfig();
    } else if (activeTab.value === 'email') {
        await loadEmailConfig();
    } else if (activeTab.value === 'storage') {
        await loadStorageConfig();
    } else if (activeTab.value === 'ai') {
        await loadAIConfig();
    }
    // 组件挂载后立即加载AI提示词标签
    await loadAIPromptTags();
});
</script>

<style scoped lang="less">
.system-view {
    padding: 20px;
}

/* 删除页面头部相关样式 */

.settings-card {
    margin-bottom: 20px;
    
    .card-header {
        display: flex;
        align-items: center;
        
        i {
            margin-right: 8px;
        }
    }
}

.system-tabs {
    margin-bottom: 20px;
}

.action-buttons {
    display: flex;
    justify-content: flex-end;
    margin-top: 20px;
    padding: 15px 0;
    border-top: 1px solid #EBEEF5;
}

.avatar-uploader-container {
    display: flex;
    flex-direction: column;
    
    .avatar-preview {
        width: 100px;
        height: 100px;
        margin-bottom: 10px;
        border-radius: 4px;
        overflow: hidden;
        border: 1px solid #EBEEF5;
        box-shadow: 0 2px 12px 0 rgba(0, 0, 0, 0.1);
        
        .avatar-image {
            width: 100%;
            height: 100%;
            object-fit: cover;
        }
    }
}

.prompt-tags-container {
    display: flex;
    flex-wrap: wrap;
    gap: 12px;
    margin-bottom: 16px;
    
    .prompt-tag {
        cursor: pointer;
        transition: all 0.3s;
        padding: 8px 16px;
        
        &:hover {
            transform: translateY(-2px);
            box-shadow: 0 2px 4px rgba(0, 0, 0, 0.1);
        }

        &.is-active {
            background-color: #409EFF;
            color: #fff;
            border-color: #409EFF;
        }
    }
}

.prompt-editor-container {
    width: 100%;
    position: relative;

    .ai-prompt-display {
        border: 1px solid #DCDFE6;
        border-radius: 4px;
        padding: 12px 16px;
        min-height: 120px;
        cursor: text;
        white-space: pre-wrap;
        word-break: break-word;
        line-height: 1.6;
        font-size: 14px;
        color: #606266;
        transition: all 0.3s;
        position: relative;
        
        &:hover {
            border-color: #C0C4CC;
            box-shadow: 0 0 0 2px rgba(64, 158, 255, 0.1);

            .edit-button {
                opacity: 1;
            }
        }
        
        .text-highlight {
            color: #409EFF;
            font-weight: bold;
            background-color: rgba(64, 158, 255, 0.1);
            padding: 0 2px;
            border-radius: 2px;
        }

        .edit-button {
            position: absolute;
            top: 8px;
            right: 8px;
            opacity: 0;
            transition: opacity 0.3s;
        }
    }
    
    .prompt-textarea {
        /* min-height: 120px; */
    }
}

.directory-selector {
    height: 400px;
    overflow-y: auto;
    
    .directory-path {
        margin-bottom: 15px;
        display: flex;
        justify-content: space-between;
        align-items: center;
        padding: 12px;
        background-color: #f5f7fa;
        border-radius: 4px;
    }
    
    .directory-tree {
        border: 1px solid #EBEEF5;
        border-radius: 4px;
        padding: 12px;
        height: 320px;
        overflow-y: auto;
    }
    
    .custom-tree-node {
        flex: 1;
        display: flex;
        align-items: center;
        justify-content: space-between;
        padding: 8px 0;
        
        i {
            margin-right: 8px;
        }
    }
}

.switch-container {
    display: flex;
    flex-direction: column;
    align-items: flex-start;
}

.form-item-tip {
    font-size: 12px;
    color: #909399;
    margin-top: 8px;
}

.info-text {
    color: #909399;
    font-size: 13px;
    margin-top: 8px;
    display: flex;
    align-items: center;
    
    i {
        margin-right: 4px;
        font-size: 14px;
    }
}

.warning-text {
    color: #F56C6C;
    font-size: 13px;
    margin-top: 8px;
    font-weight: bold;
    display: flex;
    align-items: center;
    
    i {
        margin-right: 4px;
        font-size: 14px;
    }
}

.config-item-container {
    display: flex;
    align-items: center;
    gap: 12px;
    
    .unit-text {
        color: #606266;
        font-weight: 500;
        font-size: 14px;
    }
    
    .size-hint {
        color: #909399;
        font-size: 13px;
        background-color: #f5f7fa;
        padding: 4px 8px;
        border-radius: 4px;
    }
}

.info-text.enhanced {
    background-color: #f5f7fa;
    border: 1px solid #e4e7ed;
    border-radius: 6px;
    padding: 12px 16px;
    margin-top: 8px;
    
    .info-content {
        margin-left: 8px;
        
        p {
            margin: 4px 0;
            line-height: 1.5;
            
            &:first-child {
                margin-top: 0;
            }
            
            &:last-child {
                margin-bottom: 0;
            }
        }
    }
}

.test-email-container {
    display: flex;
    align-items: center;
    margin-top: 16px;
    
    .test-email-tip {
        margin-left: 12px;
        font-size: 13px;
        color: #909399;
    }
}

:deep(.el-form-item__label) {
    font-weight: 500;
    color: #303133;
    margin-bottom: 8px;
}

:deep(.el-input__inner) {
    transition: all 0.3s;
    
    &:hover {
        border-color: #C0C4CC;
    }
    
    &:focus {
        border-color: #409EFF;
        box-shadow: 0 0 0 2px rgba(64, 158, 255, 0.2);
    }
}

:deep(.el-input-number) {
    width: 100%;
}

:deep(.el-input-number .el-input__inner) {
    text-align: left;
}

:deep(.el-form-item) {
    margin-bottom: 22px;
}

:deep(.el-card__header) {
    padding: 16px 20px;
    font-weight: 500;
}

:deep(.el-card__body) {
    padding: 20px;
}

:deep(.el-tabs__header) {
    margin-bottom: 20px;
}

:deep(.el-tabs__item) {
    font-size: 15px;
    padding: 0 20px;
}

:deep(.el-tabs__item.is-active) {
    font-weight: 600;
}

:deep(.el-divider__text) {
    font-weight: 500;
    color: #606266;
}
</style>
