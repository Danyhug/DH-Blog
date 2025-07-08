<template>
    <div class="system-view">
        <el-tabs v-model="activeTab" class="system-tabs">
            <el-tab-pane label="站点设置" name="site">
                <el-form :model="config" label-width="120px">
                    <el-form-item label="博客标题">
                        <el-input v-model="config.blog_title"></el-input>
                    </el-form-item>
                    <el-form-item label="个人签名">
                        <el-input v-model="config.signature"></el-input>
                    </el-form-item>
                    <el-form-item label="个人头像">
                        <el-input v-model="config.avatar"></el-input>
                    </el-form-item>
                    <el-form-item label="GitHub链接">
                        <el-input v-model="config.github_link"></el-input>
                    </el-form-item>
                    <el-form-item label="Bilibili链接">
                        <el-input v-model="config.bilibili_link"></el-input>
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
                <el-form :model="config" label-width="120px">
                    <el-form-item label="SMTP主机">
                        <el-input v-model="config.smtp_host"></el-input>
                    </el-form-item>
                    <el-form-item label="SMTP端口">
                        <el-input v-model.number="config.smtp_port"></el-input>
                    </el-form-item>
                    <el-form-item label="SMTP用户">
                        <el-input v-model="config.smtp_user"></el-input>
                    </el-form-item>
                    <el-form-item label="SMTP密码">
                        <el-input v-model="config.smtp_pass" type="password"></el-input>
                    </el-form-item>
                    <el-form-item label="SMTP发送者">
                        <el-input v-model="config.smtp_sender"></el-input>
                    </el-form-item>
                </el-form>
            </el-tab-pane>
            <el-tab-pane label="AI设置" name="ai">
                <el-form :model="config" label-width="120px">
                    <el-row>
                        <el-col :span="8">
                            <el-form-item label="API 地址">
                                <el-input v-model="config.ai_api_url" placeholder="https://xxx.xin/v1/chat/completions"></el-input>
                            </el-form-item>
                        </el-col>
                        <el-col :span="10">
                            <el-form-item label="API 秘钥">
                                <el-input v-model="config.ai_api_key"></el-input>
                            </el-form-item>
                        </el-col>
                        <el-col :span="6">
                            <el-form-item label="模型">
                                <el-input v-model="config.ai_model" placeholder="gpt-3.5-turbo"></el-input>
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
                            v-model="config.ai_prompt"
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
import { ref, onMounted, computed } from 'vue';
import { getSystemConfig, updateSystemConfig } from '@/api/admin';
import type { SystemConfig } from '@/types/SystemConfig';
import { ElMessage } from 'element-plus';

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
const isEditingPrompt = ref(false);

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
    if (!config.value.ai_prompt) {
        return '<span style="color: #909399;">点击此处输入AI提示词，填充符格式为 &lbrace;&lbrace;.变量名&rbrace;&rbrace;</span>';
    }
    let processedPrompt = escapeHtml(config.value.ai_prompt);
    return processedPrompt.replace(/\{\{\.[a-zA-Z0-9_]+\}\}/g, '<span class="text-highlight">$&</span>');
});

const startEditing = () => {
    isEditingPrompt.value = true;
};

const stopEditing = () => {
    isEditingPrompt.value = false;
};

const selectPrompt = (prompt: string) => {
    config.value.ai_prompt = prompt;
    isEditingPrompt.value = false; // Switch back to display mode after selecting
};

onMounted(async () => {
    const res = await getSystemConfig();
    config.value = res;
    console.log("Highlighted Prompt HTML:", highlightedPrompt.value); // Add console.log here
});

const saveConfig = async () => {
    await updateSystemConfig(config.value);
    ElMessage.success('保存成功');
};
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
</style>
