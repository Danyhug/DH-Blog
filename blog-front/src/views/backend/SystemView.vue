<template>
    <div class="system-view">
        <el-tabs v-model="activeTab" class="system-tabs">
            <el-tab-pane label="站点设置" name="site">
                <el-form :model="blogConfig" label-width="120px">
                    <el-form-item label="博客标题">
                        <el-input v-model="blogConfig.blog_title"></el-input>
                    </el-form-item>
                    <el-form-item label="个人签名">
                        <el-input v-model="blogConfig.signature"></el-input>
                    </el-form-item>
                    <el-form-item label="个人头像">
                        <el-input v-model="blogConfig.avatar"></el-input>
                    </el-form-item>
                    <el-form-item label="GitHub链接">
                        <el-input v-model="blogConfig.github_link"></el-input>
                    </el-form-item>
                    <el-form-item label="Bilibili链接">
                        <el-input v-model="blogConfig.bilibili_link"></el-input>
                    </el-form-item>
                </el-form>
            </el-tab-pane>
            <el-tab-pane label="功能设置" name="features">
                <el-form :model="config" label-width="120px">
                    <el-form-item label="开放博客">
                        <el-switch v-model="config.open_blog"></el-switch>
                    </el-form-item>
                    <el-form-item label="开放评论">
                        <el-switch v-model="config.open_comment"></el-switch>
                    </el-form-item>
                    <el-form-item label="评论邮件通知">
                        <el-switch v-model="config.comment_email_notify"></el-switch>
                    </el-form-item>
                </el-form>
            </el-tab-pane>
            <el-tab-pane label="邮箱设置" name="email">
                <el-form :model="emailConfig" label-width="120px">
                    <el-form-item label="SMTP主机">
                        <el-input v-model="emailConfig.smtp_host"></el-input>
                    </el-form-item>
                    <el-form-item label="SMTP端口">
                        <el-input v-model.number="emailConfig.smtp_port"></el-input>
                    </el-form-item>
                    <el-form-item label="SMTP用户">
                        <el-input v-model="emailConfig.smtp_user"></el-input>
                    </el-form-item>
                    <el-form-item label="SMTP密码">
                        <el-input v-model="emailConfig.smtp_pass" type="password"></el-input>
                    </el-form-item>
                    <el-form-item label="SMTP发送者">
                        <el-input v-model="emailConfig.smtp_sender"></el-input>
                    </el-form-item>
                </el-form>
            </el-tab-pane>
            <el-tab-pane label="文件存储" name="storage">
                <el-form :model="storageConfig" label-width="120px">
                    <el-form-item label="存储路径">
                        <el-input 
                            v-model="storageConfig.file_storage_path" 
                            placeholder="/path/to/your/storage/directory"
                        >
                            <template #append>
                                <el-button @click="openDirectorySelector">选择路径</el-button>
                            </template>
                        </el-input>
                        <div class="el-form-item__extra" style="color: #909399; font-size: 12px; margin-top: 5px;">
                            请设置一个服务器上有读写权限的绝对路径，用于存储上传的文件。保存后即时生效，不需要重启服务器。
                        </div>
                        <div class="el-form-item__extra" style="color: #F56C6C; font-size: 12px; margin-top: 5px; font-weight: bold;">
                            注意：更改存储路径会清空文件表，所有文件记录将被删除并重新扫描！
                        </div>
                    </el-form-item>
                </el-form>
                
                <!-- 目录选择对话框 -->
                <el-dialog
                    v-model="directoryDialogVisible"
                    title="选择存储路径"
                    width="60%"
                >
                    <div class="directory-selector">
                        <div class="directory-path">
                            <span>当前路径: {{ currentPath || '/' }}</span>
                            <el-button size="small" @click="loadParentDirectory" :disabled="!currentPath">上一级</el-button>
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
                        >
                            <template #default="{ node, data }">
                                <span class="custom-tree-node">
                                    <span>{{ node.label }}</span>
                                    <el-button
                                        v-if="data.isDir"
                                        size="small"
                                        text
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
            </el-tab-pane>
            <el-tab-pane label="AI设置" name="ai">
                <el-form :model="aiConfig" label-width="120px">
                    <el-row>
                        <el-col :span="8">
                            <el-form-item label="API 地址">
                                <el-input v-model="aiConfig.ai_api_url" placeholder="https://xxx.xin/v1/chat/completions"></el-input>
                            </el-form-item>
                        </el-col>
                        <el-col :span="10">
                            <el-form-item label="API 秘钥">
                                <el-input v-model="aiConfig.ai_api_key"></el-input>
                            </el-form-item>
                        </el-col>
                        <el-col :span="6">
                            <el-form-item label="模型">
                                <el-input v-model="aiConfig.ai_model" placeholder="gpt-3.5-turbo"></el-input>
                            </el-form-item>
                        </el-col>
                    </el-row>
                    <el-form-item label="提示词选择">
                        <div class="prompt-tags-container">
                            <el-tag
                                v-for="tag in promptTags"
                                :key="tag.label"
                                class="prompt-tag"
                                effect="plain"
                                round
                                @click="selectPrompt(tag.prompt)"
                            >
                                {{ tag.label }}
                            </el-tag>
                        </div>
                        <div class="el-form-item__extra" style="color: #909399; font-size: 12px; margin-left: 10px">
                            点击上方标签可快速填充AI提示词，方便您快速修改，切勿修改模板填充符
                        </div>
                    </el-form-item>
                    <el-form-item label="提示词">
                        <div v-if="!isEditingPrompt" class="ai-prompt-display" @click="startEditing" v-html="highlightedPrompt"></div>
                        <el-input
                            v-else
                            v-model="aiConfig.ai_prompt"
                            type="textarea"
                            autosize
                            @blur="stopEditing"
                        ></el-input>
                        <div class="el-form-item__extra" style="color: #E6A23C; font-size: 12px; margin-top: 5px;">
                            请勿修改 <span style="font-weight: bold;">&lbrace;&lbrace;.变量名&rbrace;&rbrace;</span> 形式的模板变量，它们将在生成内容时被自动替换。
                        </div>
                    </el-form-item>
                </el-form>
            </el-tab-pane>
        </el-tabs>
        <div class="save-button-container">
            <el-button type="primary" @click="saveConfig">保存</el-button>
        </div>
    </div>
</template>

<script setup lang="ts">
import { ref, onMounted, computed, watch } from 'vue';
import { 
    getSystemConfig, updateSystemConfig, 
    getBlogConfig, updateBlogConfig,
    getEmailConfig, updateEmailConfig,
    getAIConfig, updateAIConfig,
    getStorageConfig, updateStorageConfig,
} from '@/api/admin';
import { getDirectoryTree } from '@/api/file';
import type { SystemConfig, BlogConfig, EmailConfig, AIConfig, StorageConfig } from '@/types/SystemConfig';
import { ElMessage, ElMessageBox } from 'element-plus';

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
const storageConfig = ref<StorageConfig>({});
const isEditingPrompt = ref(false);

// 目录选择相关
const directoryDialogVisible = ref(false);
const directoryTree = ref<any[]>([]);
const currentPath = ref('');
const selectedPath = ref('');
const expandedKeys = ref<string[]>([]);

const promptTags = ref([
    {
        label: '文章标签提取',
        prompt: '请根据以下文章内容，提取出3-5个关键词作为文章标签，用逗号分隔。文章内容：{{.ArticleContent}}'
    },
    {
        label: '文章摘要生成',
        prompt: '请根据以下文章内容，生成一个100字左右的摘要。文章内容：{{.ArticleContent}}'
    },
    {
        label: '评论回复',
        prompt: '请根据以下评论内容，生成一个友好的回复。评论内容：{{.CommentContent}}，回复对象：{{.UserName}}'
    }
]);

const highlightedPrompt = computed(() => {
    if (!aiConfig.value.ai_prompt) {
        return '<span style="color: #909399;">点击此处输入AI提示词，填充符格式为 &lbrace;&lbrace;.变量名&rbrace;&rbrace;</span>';
    }
    let processedPrompt = escapeHtml(aiConfig.value.ai_prompt);
    return processedPrompt.replace(/\{\{\.[a-zA-Z0-9_]+\}\}/g, '<span class="text-highlight">$&</span>');
});

const startEditing = () => {
    isEditingPrompt.value = true;
};

const stopEditing = () => {
    isEditingPrompt.value = false;
};

const selectPrompt = (prompt: string) => {
    aiConfig.value.ai_prompt = prompt;
    isEditingPrompt.value = false; // Switch back to display mode after selecting
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
        storageConfig.value = res;
    } catch (error) {
        ElMessage.error('加载存储配置失败');
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
        ai_prompt: res.ai_prompt
    };
    
    storageConfig.value = {
        file_storage_path: res.file_storage_path
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
});
</script>

<style scoped lang="less">
.system-view {
    padding: 20px;
}

.save-button-container {
    margin-top: 20px;
    text-align: right;
}

.ai-prompt-display {
    border: 1px solid #DCDFE6;
    border-radius: 4px;
    padding: 8px 12px;
    cursor: text;
    white-space: pre-wrap; /* Preserve whitespace and wrap text */
    word-break: break-word; /* Break long words */
    line-height: 1.5;
    font-size: 14px;
    color: #606266;
}

.ai-prompt-display:hover {
    border-color: #C0C4CC;
    cursor: pointer;
}

.prompt-tags-container {
    display: flex;
    flex-wrap: wrap;
    gap: 10px;
    align-items: center;
}

.prompt-tag {
    cursor: pointer;
}

.directory-selector {
    height: 400px;
    overflow-y: auto;
}

.directory-path {
    margin-bottom: 15px;
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 5px 10px;
    background-color: #f5f7fa;
    border-radius: 4px;
}

.custom-tree-node {
    flex: 1;
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding-right: 8px;
}
</style>
