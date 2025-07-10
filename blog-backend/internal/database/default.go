package database

import (
	"fmt"

	"dh-blog/internal/model"
	"gorm.io/gorm"
)

const (
	DefaultTagsPrompt = "# 角色\n你是一位顶级的概念抽象与标签生成专家，尤其擅长为博客文章提取核心标签。\n\n# 任务\n分析文本，提取其核心概念，生成一个高度精炼、无语义重复的标签列表，用于博客分类和SEO。\n\n# 核心规则\n1.  **概念合并**: 必须将所有语义相似的自生成标签合并成一个最具代表性的。例如，将“笔记方法”和“卡片盒笔记法”合并为更具体的“Zettelkasten”。\n2.  **复用现有**: 如果生成的概念与 `[现有的标签列表]` 中的标签语义相同（如“效率” vs “效率提升”），必须使用现有标签，放弃自己生成的。\n3.  **数量限制**: 最终输出的标签数量严格控制在 **1 到 10 个** 之间。\n4.  **质量要求**: 标签必须是高度概括的核心概念，并按重要性降序排列。\n\n# 输出格式\n严格以JSON格式返回，绝不包含任何额外说明。\n`[\"标签1\", \"标签2\", ...]`\n如果无法提取有效标签，则返回 `[]`。\n\n---\n### 示例演示 (博客场景)\n\n**[待分析的文本]:**\n\"在今天这个信息爆炸的时代，如何有效管理知识成了每个人的必修课。我最近在实践一种名为“Zettelkasten”的笔记方法，它也被称为“卡片盒笔记法”。通过使用Obsidian这款工具，我将每一个想法或知识点制作成一张独立的卡片，并建立它们之间的链接。这个过程不仅是简单的记录，更是在构建我的“第二大脑”，极大地提升了我的思考效率和文章输出质量。\"\n\n**[现有的标签列表]:**\n[\"效率提升\", \"知识管理\", \"软件工具\", \"学习方法\"]\n\n**# 期望的输出:**\n[\"知识管理\", \"Zettelkasten\", \"效率提升\", \"Obsidian\"]\n---\n*(内部思考逻辑：从文本中提取出“知识管理”、“Zettelkasten”、“Obsidian”、“效率”、“第二大脑”等概念。其中，“效率”与现有标签“效率提升”语义重合，故使用后者。“第二大脑”与“知识管理”概念重合，故使用更通用的现有标签“知识管理”。“Zettelkasten”和“Obsidian”是新的核心概念，予以保留。最终筛选排序得到结果。)*\n\n### 正式开始\n\n**[待分析的文本]:**\n{{.Article}}\n\n**[现有的标签列表]:**\n{{.Tags}}"
)

// insertDefaultData 插入默认数据
func insertDefaultData(db *gorm.DB) error {
	// 插入默认分类
	var categoriesCount int64
	db.Model(&model.Category{}).Count(&categoriesCount)
	if categoriesCount == 0 {
		defaultCategories := []model.Category{{Name: "默认分类", Slug: "default"}}
		if err := db.Create(&defaultCategories).Error; err != nil {
			return fmt.Errorf("创建默认分类失败: %w", err)
		}
		fmt.Println("已创建默认分类")
	}

	// 插入默认系统设置
	var settingsCount int64
	db.Model(&model.SystemSetting{}).Count(&settingsCount)
	if settingsCount == 0 {
		defaultSettings := []model.SystemSetting{
			{SettingKey: "blog_title", SettingValue: "DH-Blog"},
			{SettingKey: "signature", SettingValue: "我们原神玩家是这样的"},
			{SettingKey: "open_comment", SettingValue: "true"},
			{SettingKey: "ai_model", SettingValue: "deepseek-v3"},
			{SettingKey: "ai_api_url", SettingValue: "https://tbai.xin/v1/chat/completions"},
			{SettingKey: "ai_prompt", SettingValue: DefaultTagsPrompt},
		}
		if err := db.Create(&defaultSettings).Error; err != nil {
			return fmt.Errorf("创建默认系统设置失败: %w", err)
		}
		fmt.Println("已创建默认系统设置")
	}

	return nil
}
