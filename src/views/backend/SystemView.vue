<template>
  <div class="form-container">
    <el-card class="form-card" shadow="none">
      <el-form :model="formData" label-width="120px" ref="formRef" class="form-content" :disabled="isDisabled">
        <!-- 折叠面板 -->
        <el-collapse v-model="activeNames" class="form-collapse">
          <!-- 站点设置 -->
          <el-collapse-item title="站点设置" name="site-settings">
            <div class="horizontal-form">
              <el-form-item label="博客标题" class="form-item">
                <el-input v-model="formData.blogTitle" placeholder="输入博客标题" />
              </el-form-item>
              <el-form-item label="签名" class="form-item">
                <el-input v-model="formData.signature" placeholder="输入签名" />
              </el-form-item>
              <el-form-item label="预览文章数" class="form-item">
                <el-input-number v-model="formData.homeArticleCount" :min="1" :max="50" />
              </el-form-item>
            </div>
          </el-collapse-item>
          <!-- SEO 设置 -->
          <el-collapse-item title="SEO 设置" name="seo">
            <div class="horizontal-form">
              <el-form-item label="网站关键词" class="form-item">
                <el-input v-model="formData.seoKeywords" placeholder="输入网站关键词" />
              </el-form-item>
              <el-form-item label="网站描述" class="form-item">
                <el-input v-model="formData.seoDescription" placeholder="输入网站描述" />
              </el-form-item>
            </div>
          </el-collapse-item>
          <!-- 站点功能设置 -->
          <el-collapse-item title="站点功能设置" name="site-functions">
            <div class="horizontal-form btn-group">
              <el-form-item label="开放博客" class="form-item2">
                <el-switch v-model="formData.isBlogOpen" />
              </el-form-item>
              <el-form-item label="开放评论" class="form-item2">
                <el-switch v-model="formData.isCommentsOpen" />
              </el-form-item>
              <el-form-item label="评论邮件通知" class="form-item2">
                <el-switch v-model="formData.notifyCommenter" />
              </el-form-item>
            </div>
          </el-collapse-item>
          <!-- 邮箱设置 -->
          <el-collapse-item title="邮箱设置" name="email-settings">
            <div class="vertical-form mail">
              <el-form-item label="邮箱地址" class="form-item">
                <el-input v-model="formData.email" placeholder="输入邮箱地址" />
              </el-form-item>
            </div>
          </el-collapse-item>
        </el-collapse>
      </el-form>

      <!-- 提交按钮 -->
      <div class="form-footer">
        <el-button type="primary" @click="toggleEdit" class="submit-btn">
          {{ isDisabled ? '编辑' : '保存' }}
        </el-button>
      </div>
    </el-card>
  </div>
</template>
<script setup>
import { reactive, ref } from "vue";
import { ElMessage } from "element-plus";

const formData = reactive({
  avatarMD5: "",
  blogTitle: "",
  // 其他表单数据
});

const activeNames = ref(["site-settings", /* 其他面板名称 */]);
const isDisabled = ref(true);
const formRef = ref(null);

const toggleEdit = () => {
  if (isDisabled.value) {
    isDisabled.value = false;
  } else {
    submitForm();
  }
};

const submitForm = () => {
  formRef.value.validate((valid) => {
    if (valid) {
      // 上传数据的逻辑
      console.log('提交表单数据：', formData);
      // 模拟数据上传成功后禁用表单
      ElMessage.success('数据保存成功');
      isDisabled.value = true;
    } else {
      ElMessage.error('表单验证失败');
      return false;
    }
  });
};
</script>
<style scoped lang="less">
/* 基础布局 */
.form-card {
  padding: 0 20px;
}

/* 水平表单布局 */
.horizontal-form {
  display: flex;
  flex-wrap: wrap;
  gap: 20px;

  .form-item {
    flex: 1 1 calc(50% - 20px);
    min-width: 200px;
    max-width: 400px;
  }

  .form-item2 {
    flex: 1 1 calc(25% - 20px);
    min-width: 200px;
    max-width: 400px;
  }
}


/* 表单页脚 */
.form-footer {
  text-align: center;
  margin-top: 20px;

  .submit-btn {
    width: 200px;
  }
}

/* 美化 input 样式 */
.el-input__inner {
  border: 1px solid #ccc;
  background-color: #f9f9f9;
  border-radius: 6px;
  height: 40px;
  transition: all 0.3s ease;
  font-size: 14px;
  color: #333;
}

.mail :deep(.el-input) {
  width: 360px;
}
</style>