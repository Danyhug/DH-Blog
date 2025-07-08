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
                    <el-form-item label="AI提示词">
                        <el-input v-model="config.ai_prompt" type="textarea"></el-input>
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
import { ref, onMounted } from 'vue';
import { getSystemConfig, updateSystemConfig } from '@/api/admin';
import type { SystemConfig } from '@/types/SystemConfig';
import { ElMessage } from 'element-plus';

const activeTab = ref('site');
const config = ref<SystemConfig>({});

onMounted(async () => {
    const res = await getSystemConfig();
    config.value = res;
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
</style>
