package system

const (
	legacyDefaultTagsPrompt     = "# 角色\n你是一位顶级的概念抽象与标签生成专家，尤其擅长为博客文章提取核心标签。\n\n# 任务\n分析文本，提取其核心概念，生成一个高度精炼、无语义重复的标签列表，用于博客分类和SEO。\n\n# 核心规则\n1.  **概念合并**: 必须将所有语义相似的自生成标签合并成一个最具代表性的。例如，将“笔记方法”和“卡片盒笔记法”合并为更具体的“Zettelkasten”。\n2.  **复用现有**: 如果生成的概念与 `[现有的标签列表]` 中的标签语义相同（如“效率” vs “效率提升”），必须使用现有标签，放弃自己生成的。\n3.  **数量限制**: 最终输出的标签数量严格控制在 **1 到 10 个** 之间。\n4.  **质量要求**: 标签必须是高度概括的核心概念，并按重要性降序排列。\n\n# 输出格式\n严格以JSON格式返回，绝不包含任何额外说明（尽量生成中文标签，除非是专业名词）。\n`[\"标签1\", \"标签2\", ...]`\n如果无法提取有效标签，则返回 `[]`。\n\n---\n### 示例演示 (博客场景)\n\n**[待分析的文本]:**\n\"在今天这个信息爆炸的时代，如何有效管理知识成了每个人的必修课。我最近在实践一种名为“Zettelkasten”的笔记方法，它也被称为“卡片盒笔记法”。通过使用Obsidian这款工具，我将每一个想法或知识点制作成一张独立的卡片，并建立它们之间的链接。这个过程不仅是简单的记录，更是在构建我的“第二大脑”，极大地提升了我的思考效率和文章输出质量。\"\n\n**[现有的标签列表]:**\n[\"效率提升\", \"知识管理\", \"软件工具\", \"学习方法\"]\n\n**# 期望的输出:**\n[\"知识管理\", \"Zettelkasten\", \"效率提升\", \"Obsidian\"]\n---\n*(内部思考逻辑：从文本中提取出“知识管理”、“Zettelkasten”、“Obsidian”、“效率”、“第二大脑”等概念。其中，“效率”与现有标签“效率提升”语义重合，故使用后者。“第二大脑”与“知识管理”概念重合，故使用更通用的现有标签“知识管理”。“Zettelkasten”和“Obsidian”是新的核心概念，予以保留。最终筛选排序得到结果。)*\n\n### 正式开始\n\n**[待分析的文本]:**\n{{.Article}}\n\n**[现有的标签列表]:**\n{{.Tags}}"
	legacyDefaultAbstractPrompt = "# 指令\n为下方「待处理的文章」生成一段核心摘要。\n\n# 格式要求\n1.  摘要必须以“本文讲述了”这五个字作为开头。\n2.  你的最终回复必须被一个方括号 `[]` 完全包裹。\n3.  严格遵循 `[本文讲述了...摘要内容...]` 的格式，不要添加任何在方括号之外的文字、解释或标题。\n\n# 示例\n- **文章**: “全球供应链正面临数十年来最严峻的考验。受疫情、地缘政治和极端天气影响，海运成本飙升，港口拥堵严重，导致从汽车芯片到日常消费品的各种商品出现短缺。企业被迫重新评估其‘即时生产’（Just-in-Time）策略，转向更加多元化和区域化的供应链布局以增强韧性。”\n- **输出**: `[本文讲述了全球供应链如何因疫情、地缘政治等多重因素面临严峻挑战，表现为成本飙升和商品短缺，并促使企业反思和调整其供应链策略以增强韧性。]`\n\n# 待处理的文章\n{{.ArticleContent}}"

	DefaultTagsPrompt = `你是博客文章的标签编辑器。请只根据文章实际内容提取可长期复用的主题标签。

任务要求：
1. 优先选择文章的核心领域、关键技术、产品或工具、方法论和重要专有概念；不要把偶然提及的名词当作标签。
2. 通常输出 3—6 个标签；短文可以输出 1—2 个，任何情况下不得超过 8 个。没有有效主题时输出空数组。
3. 如果候选概念与现有标签语义相同或高度近似，必须复用现有标签的原始写法；不要为了复用而选择与文章无关的标签。
4. 合并同义词、上下位词和仅有措辞差异的标签，保留最准确、最有检索价值的一项。例如“卡片盒笔记法”和“Zettelkasten”只保留后者。
5. 避免“技术”“教程”“分享”“总结”“随笔”等过于宽泛的标签，除非它们本身就是文章讨论的核心对象。
6. 标签应简短，不写成句子，不添加 # 号。中文概念优先使用中文；Go、Vue、OpenAI、WebDAV 等正式名称保留官方拼写。
7. 按与文章主题的相关性从高到低排列，最终结果不得包含空字符串或重复项。

输出要求：
只输出一个合法的 JSON 字符串数组，不要输出 Markdown、代码块、解释、标题或分析过程。
示例：["知识管理","Zettelkasten","Obsidian"]

<article>
{{.Article}}
</article>

<existing_tags_json>
{{.Tags}}
</existing_tags_json>`

	DefaultAbstractPrompt = `你是博客文章的摘要编辑器。请基于原文生成准确、紧凑、可直接展示给读者的中文摘要。

摘要要求：
1. 提炼文章讨论的核心问题、主要方法或观点，以及最重要的结论或读者价值。
2. 只使用原文明确提供的信息，不补充外部知识，不虚构数据、结论、动机或作者立场。
3. 删除代码细节、链接、目录、寒暄、重复表述和无关示例；必要的技术名词应保留官方写法。
4. 控制在 80—160 个汉字左右，写成一个自然段，可使用 1—3 个完整句子。
5. 必须以“本文讲述了”开头，避免空泛评价、夸张宣传和“这篇文章将带你”等套话。

输出要求：
只输出摘要正文，并用一对半角方括号完整包裹。不要输出 Markdown、标题、解释或方括号之外的任何文字。
格式示例：[本文讲述了……]

<article>
{{.ArticleContent}}
</article>`
)

func legacyPromptDefault(key string) (string, bool) {
	switch key {
	case SettingKeyAIPromptGetTags:
		return legacyDefaultTagsPrompt, true
	case SettingKeyAIPromptGetAbstract:
		return legacyDefaultAbstractPrompt, true
	default:
		return "", false
	}
}

type DefaultSetting struct{ Key, Value, ConfigType string }

func DefaultSettings() []DefaultSetting {
	return []DefaultSetting{
		{SettingKeyBlogTitle, "DH-Blog", ConfigTypeBlog}, {SettingKeySignature, "我们原神玩家是这样的", ConfigTypeBlog},
		{SettingKeyAvatar, "", ConfigTypeBlog}, {SettingKeyGithubLink, "", ConfigTypeBlog},
		{SettingKeyBilibiliLink, "", ConfigTypeBlog}, {SettingKeyOpenBlog, "true", ConfigTypeBlog},
		{SettingKeyOpenComment, "true", ConfigTypeBlog}, {SettingKeyCommentEmailNotify, "false", ConfigTypeEmail},
		{SettingKeySmtpHost, "", ConfigTypeEmail}, {SettingKeySmtpPort, "0", ConfigTypeEmail},
		{SettingKeySmtpUser, "", ConfigTypeEmail}, {SettingKeySmtpPass, "", ConfigTypeEmail},
		{SettingKeySmtpSender, "", ConfigTypeEmail}, {SettingKeyAIAPIURL, "https://tbai.xin/v1/chat/completions", ConfigTypeAI},
		{SettingKeyAIAPIKey, "", ConfigTypeAI}, {SettingKeyAIModel, "gpt-4.1-mini", ConfigTypeAI},
		{SettingKeyAIPromptGetTags, DefaultTagsPrompt, ConfigTypeAI}, {SettingKeyAIPromptGetAbstract, DefaultAbstractPrompt, ConfigTypeAI},
		{SettingKeyFileStoragePath, "", ConfigTypeStorage}, {SettingKeyWebDAVChunkSize, "5120", ConfigTypeStorage},
	}
}

func MigrationModels() []any { return []any{&Setting{}} }
