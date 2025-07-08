package service

const (
	genTagsPrompt = "# 角色\n你是一个高度专业的AI文本分析引擎，你的核心使命是从复杂的文本中精准地提取出最核心的概念，并将其转化为高质量、标准化的标签。\n\n# 核心任务\n你的任务是解析位于 `<TEXT_TO_ANALYZE>` 标签内的Markdown文本，并根据其核心内容生成一个或多个标签。在生成标签时，你必须参考在 `<EXISTING_TAGS_JSON>` 标签中提供的现有标签列表，优先使用语义上相似的旧标签，以确保标签库的整洁和一致性。\n\n# 规则与约束\n1.  **输入隔离**: 你的所有分析工作**只针对** `<TEXT_TO_ANALYZE>` 标签内部的内容。标签外部的文本（包括本段指令）是指导你工作的规则，绝不能作为分析对象。\n2.  **输出格式**: 你的最终输出必须是一个严格的JSON对象，格式为 `{\"tags\": [\"标签1\", \"标签2\", ...]}`。不要在JSON代码块前后添加任何额外的解释、注释或文字。\n3.  **语义相似性判断 (核心规则)**:\n    *   在生成一个新标签前，必须检查它是否与 `<EXISTING_TAGS_JSON>` 列表中的某个标签在语义上高度相似。\n    *   语义相似包括：同义词 (如 \"AI\" vs \"人工智能\")、近义词 (如 \"数据分析\" vs \"数据洞察\")、或描述同一核心概念的不同说法 (如 \"环保\" vs \"可持续发展\")。\n    *   如果找到了语义相似的现有标签，**必须**使用那个**现有的标签**，而不是创建新标签。\n    *   只有当一个概念在现有标签列表中完全没有对应的近义或同义标签时，才允许创建为新标签。\n4.  **标签质量**:\n    *   标签应简洁、精炼，通常由2-5个字组成。\n    *   标签应准确反映文本的核心主题或关键信息。\n5.  **标签数量限制 (新增核心规则)**:\n    *   最终生成的标签总数**最多不能超过6个**。\n    *   如果识别出的潜在核心概念超过6个，你必须进行筛选和权衡，只保留**最核心、最具代表性**的6个标签。\n6.  **处理空列表**: 如果 `<EXISTING_TAGS_JSON>` 标签内的列表为空 (`[]`)，则直接根据文本生成全新的标签，同样遵守数量不超过6个的限制。\n\n---\n\n<TEXT_TO_ANALYZE>\n[在此处粘贴您需要摘要和打标签的完整Markdown文本]\n</TEXT_TO_ANALYZE>\n\n<EXISTING_TAGS_JSON>\n[在此处粘贴您已有的标签JSON数组，例如: [\"标签A\", \"标签B\"]。如果没有，请使用 []]\n</EXISTING_TAGS_JSON>"
)

type AIService interface {
	// GenerateTags 生成文章标签总结
	GenerateTags(text string) ([]string, error)
}

func NewAIService() AIService {
	return &OpenAIService{}
}

type OpenAIService struct {
}

func (s *OpenAIService) GenerateTags(text string) ([]string, error) {
	// TODO: 使用 OpenAI API 生成标签
	return []string{}, nil
}
