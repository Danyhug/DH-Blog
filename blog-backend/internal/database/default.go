package database

import (
	"fmt"

	"dh-blog/internal/model"

	"gorm.io/gorm"
)

const (
	DefaultTagsPrompt     = "# 角色\n你是一位顶级的概念抽象与标签生成专家，尤其擅长为博客文章提取核心标签。\n\n# 任务\n分析文本，提取其核心概念，生成一个高度精炼、无语义重复的标签列表，用于博客分类和SEO。\n\n# 核心规则\n1.  **概念合并**: 必须将所有语义相似的自生成标签合并成一个最具代表性的。例如，将“笔记方法”和“卡片盒笔记法”合并为更具体的“Zettelkasten”。\n2.  **复用现有**: 如果生成的概念与 `[现有的标签列表]` 中的标签语义相同（如“效率” vs “效率提升”），必须使用现有标签，放弃自己生成的。\n3.  **数量限制**: 最终输出的标签数量严格控制在 **1 到 10 个** 之间。\n4.  **质量要求**: 标签必须是高度概括的核心概念，并按重要性降序排列。\n\n# 输出格式\n严格以JSON格式返回，绝不包含任何额外说明（尽量生成中文标签，除非是专业名词）。\n`[\"标签1\", \"标签2\", ...]`\n如果无法提取有效标签，则返回 `[]`。\n\n---\n### 示例演示 (博客场景)\n\n**[待分析的文本]:**\n\"在今天这个信息爆炸的时代，如何有效管理知识成了每个人的必修课。我最近在实践一种名为“Zettelkasten”的笔记方法，它也被称为“卡片盒笔记法”。通过使用Obsidian这款工具，我将每一个想法或知识点制作成一张独立的卡片，并建立它们之间的链接。这个过程不仅是简单的记录，更是在构建我的“第二大脑”，极大地提升了我的思考效率和文章输出质量。\"\n\n**[现有的标签列表]:**\n[\"效率提升\", \"知识管理\", \"软件工具\", \"学习方法\"]\n\n**# 期望的输出:**\n[\"知识管理\", \"Zettelkasten\", \"效率提升\", \"Obsidian\"]\n---\n*(内部思考逻辑：从文本中提取出“知识管理”、“Zettelkasten”、“Obsidian”、“效率”、“第二大脑”等概念。其中，“效率”与现有标签“效率提升”语义重合，故使用后者。“第二大脑”与“知识管理”概念重合，故使用更通用的现有标签“知识管理”。“Zettelkasten”和“Obsidian”是新的核心概念，予以保留。最终筛选排序得到结果。)*\n\n### 正式开始\n\n**[待分析的文本]:**\n{{.Article}}\n\n**[现有的标签列表]:**\n{{.Tags}}"
	DefaultAbstractPrompt = "# 指令\n为下方「待处理的文章」生成一段核心摘要。\n\n# 格式要求\n1.  摘要必须以“本文讲述了”这五个字作为开头。\n2.  你的最终回复必须被一个方括号 `[]` 完全包裹。\n3.  严格遵循 `[本文讲述了...摘要内容...]` 的格式，不要添加任何在方括号之外的文字、解释或标题。\n\n# 示例\n- **文章**: “全球供应链正面临数十年来最严峻的考验。受疫情、地缘政治和极端天气影响，海运成本飙升，港口拥堵严重，导致从汽车芯片到日常消费品的各种商品出现短缺。企业被迫重新评估其‘即时生产’（Just-in-Time）策略，转向更加多元化和区域化的供应链布局以增强韧性。”\n- **输出**: `[本文讲述了全球供应链如何因疫情、地缘政治等多重因素面临严峻挑战，表现为成本飙升和商品短缺，并促使企业反思和调整其供应链策略以增强韧性。]`\n\n# 待处理的文章\n{{.ArticleContent}}"
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
			{SettingKey: model.SettingKeyBlogTitle, SettingValue: "DH-Blog"},
			{SettingKey: model.SettingKeySignature, SettingValue: "我们原神玩家是这样的"},
			{SettingKey: model.SettingKeyOpenComment, SettingValue: "true"},
			{SettingKey: model.SettingKeyAiModel, SettingValue: "gpt-4.1-mini"},
			{SettingKey: model.SettingKeyAiApiURL, SettingValue: "https://tbai.xin/v1/chat/completions"},
			{SettingKey: model.SettingKeyAiPromptGetTags, SettingValue: DefaultTagsPrompt},
			{SettingKey: model.SettingKeyAiPromptGetAbstract, SettingValue: DefaultAbstractPrompt},
		}
		if err := db.Create(&defaultSettings).Error; err != nil {
			return fmt.Errorf("创建默认系统设置失败: %w", err)
		}
		fmt.Println("已创建默认系统设置")
	}

	return nil
}
