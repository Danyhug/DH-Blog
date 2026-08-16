package aigateway

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"dh-blog/internal/platform/mcp"
	"dh-blog/internal/platform/search"

	"github.com/sirupsen/logrus"
)

// mcpToolWebSearch is the tool's name, what both tools/list and tools/call key
// on. MCP clients namespace it under the server name automatically, so no
// prefix is needed.
const mcpToolWebSearch = "web_search"

// webSearchTool adapts the gateway's search path to MCP. Definition is rendered
// per caller because the provider enum depends on which providers the
// authenticated key can reach, and Call routes through Service.SearchFrom so
// MCP traffic keeps the same metering, quota, caching and request logs as the
// HTTP path — only the log's endpoint label differs.
type webSearchTool struct {
	service *Service
}

func (t *webSearchTool) Name() string { return mcpToolWebSearch }

func (t *webSearchTool) Definition(ctx context.Context) mcp.Definition {
	key := mcpKeyFromContext(ctx)
	providers, err := t.service.ProviderStatuses(ctx, key)
	if err != nil {
		// Degrade the description to "no provider available" rather than
		// failing tools/list: a temporary status-read error is not worth
		// breaking the whole tool advertisement over.
		logrus.Warnf("获取 MCP 工具定义的供应商状态失败: %v", err)
		providers = nil
	}
	return webSearchDefinition(providers)
}

func (t *webSearchTool) Call(ctx context.Context, args json.RawMessage) mcp.Result {
	// 工具参数与 HTTP 接口的 body 是同一套字段，校验也就共用同一份实现
	var arguments mcpSearchArguments
	if len(args) > 0 {
		if err := json.Unmarshal(args, &arguments); err != nil {
			return mcp.ToolError("arguments 解析失败: " + err.Error())
		}
	}
	req, invalid := normalizeSearch(arguments.merged())
	if invalid != nil {
		return mcp.ToolError(invalid.Message)
	}

	result, searchErr := t.service.SearchFrom(ctx, mcpKeyFromContext(ctx), req, mcpClientIPFromContext(ctx), "mcp/search")
	if searchErr != nil {
		// 限流、配额、上游故障都是执行期失败：写进结果让模型自己决定要不要换个问法或放弃
		return mcp.ToolError(mcpFailureText(searchErr))
	}
	return mcp.Text(renderSearchResult(result))
}

// webSearchDefinition renders the tool's advertised schema from the providers
// this particular key can actually reach, so the model never sees a `provider`
// value that the gateway would reject, and knows up front who can return an
// answer or page text.
func webSearchDefinition(providers []providerStatus) mcp.Definition {
	return mcp.Definition{
		Name:        mcpToolWebSearch,
		Title:       "联网搜索",
		Description: webSearchDescription(providers),
		InputSchema: webSearchInputSchema(providers),
	}
}

func webSearchDescription(providers []providerStatus) string {
	var builder strings.Builder
	// 说明写成"这就是本环境的联网搜索"，而不是"博客自建的一个网关"。
	// 后者读起来像个可选的附加工具，模型多半会绕开它去用内置搜索——
	// 而这个 MCP 存在的意义正是取代内置搜索。
	builder.WriteString("本环境的联网搜索工具。需要现网信息时用它：查文档与 API 用法、" +
		"查报错原因、确认某个库的最新版本与用法变更、查新闻与时效性事实，" +
		"以及任何超出模型已有知识的问题。\n")
	builder.WriteString("返回标题、链接和摘要。摘要通常已经够回答问题，不必再逐条抓取网页；" +
		"确实要读全文时把 include_raw_content 设为 true，可以省掉单独的抓取步骤。\n")

	if len(providers) == 0 {
		builder.WriteString("注意：当前没有可用的搜索供应商，调用会直接失败。")
		return builder.String()
	}

	answering := make([]string, 0, len(providers))
	for _, provider := range providers {
		if provider.SupportsAnswer {
			answering = append(answering, provider.Name)
		}
	}
	if len(answering) > 0 {
		builder.WriteString("把 include_answer 设为 true 可以额外要一段直接答案（由 " +
			strings.Join(answering, "、") + " 提供）。\n")
	}
	// provider 是给排障用的旁路，日常调用不该让模型在这上面花心思
	builder.WriteString("provider 留空即可，网关会自己选路。")
	return builder.String()
}

// mcpSearchArguments is the tool-call argument shape.
//
// The domain filters are accepted under two names: the ones Claude Code's
// built-in WebSearch uses, and the gateway's own. A model carrying habits from
// the built-in tool would otherwise have its filter silently dropped, because
// json.Unmarshal ignores fields it does not recognise — the search would run
// unfiltered and look like it worked.
type mcpSearchArguments struct {
	searchBody
	AllowedDomains []string `json:"allowed_domains"`
	BlockedDomains []string `json:"blocked_domains"`
}

// merged folds the aliases into the body the HTTP path already validates.
func (a mcpSearchArguments) merged() searchBody {
	body := a.searchBody
	body.IncludeDomains = append(body.IncludeDomains, a.AllowedDomains...)
	body.ExcludeDomains = append(body.ExcludeDomains, a.BlockedDomains...)
	return body
}

// webSearchInputSchema mirrors the HTTP body's fields, minus allow_fallback and
// no_cache: both have sane defaults and neither is a decision a model should be
// spending tokens on.
func webSearchInputSchema(providers []providerStatus) map[string]any {
	enum := make([]string, 0, len(providers)+1)
	enum = append(enum, providerAuto)
	for _, provider := range providers {
		enum = append(enum, provider.Name)
	}

	domainList := func(description string) map[string]any {
		return map[string]any{
			"type":        "array",
			"description": description,
			"items":       map[string]any{"type": "string"},
			"maxItems":    maxDomainFilter,
		}
	}

	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"query": map[string]any{
				"type":        "string",
				"description": "搜索词，最长 400 字符。支持 site:、filetype:、intitle: 等操作符，带操作符时网关会优先选支持它们的供应商。",
				"maxLength":   maxQueryLength,
			},
			"max_results": map[string]any{
				"type":        "integer",
				"description": "返回条数，默认 5。",
				"minimum":     1,
				"maximum":     20,
			},
			"provider": map[string]any{
				"type":        "string",
				"description": "指定供应商，一般不需要填。留空或 auto 表示交给网关选路。",
				"enum":        enum,
			},
			"topic": map[string]any{
				"type":        "string",
				"description": "检索主题，news 会偏向新闻源。",
				"enum":        []string{search.TopicGeneral, search.TopicNews},
			},
			"freshness": map[string]any{
				"type":        "string",
				"description": "时间范围：day、week、month、year，或 YYYY-MM-DDtoYYYY-MM-DD。",
			},
			"country": map[string]any{
				"type":        "string",
				"description": "2 位 ISO 国家代码，用于地域相关的查询。",
				"minLength":   2,
				"maxLength":   2,
			},
			"language": map[string]any{
				"type":        "string",
				"description": "语言代码，例如 zh、en。",
			},
			"allowed_domains": domainList("只在这些域名内搜索，最多 50 项。"),
			"blocked_domains": domainList("排除这些域名，最多 50 项。"),
			// 网关 HTTP 接口用的名字，同时收下，便于照着接口文档写的调用直接可用
			"include_domains": domainList("allowed_domains 的别名。"),
			"exclude_domains": domainList("blocked_domains 的别名。"),
			"include_answer": map[string]any{
				"type":        "boolean",
				"description": "让供应商额外给一段直接答案。只有支持该能力的供应商会被选中。",
			},
			"include_raw_content": map[string]any{
				"type":        "boolean",
				"description": "连网页正文一起返回，省掉再抓一次的步骤。内容很长，只在确实要读全文时开启。",
			},
		},
		"required":             []string{"query"},
		"additionalProperties": false,
	}
}

// renderSearchResult turns a gateway result into the text the model reads.
// Plain text rather than raw JSON: it costs fewer tokens and the fields an
// agent actually acts on (link, snippet) stay readable.
func renderSearchResult(result SearchResult) string {
	var builder strings.Builder
	fmt.Fprintf(&builder, "来源 %s", result.Provider)
	if result.Meta.FallbackFrom != "" {
		fmt.Fprintf(&builder, "（由 %s 回退）", result.Meta.FallbackFrom)
	}
	if result.Meta.Cached {
		builder.WriteString(" · 缓存命中")
	}
	fmt.Fprintf(&builder, " · 耗时 %dms\n", result.Meta.LatencyMS)

	if result.Answer != "" {
		builder.WriteString("\n答案：" + result.Answer + "\n")
	}

	if len(result.Results) == 0 {
		builder.WriteString("\n没有找到结果。")
		return builder.String()
	}

	for index, item := range result.Results {
		fmt.Fprintf(&builder, "\n%d. %s\n   %s\n", index+1, item.Title, item.URL)
		if item.PublishedAt != nil {
			fmt.Fprintf(&builder, "   发布于 %s\n", item.PublishedAt.Format("2006-01-02"))
		}
		if item.Content != "" {
			builder.WriteString("   " + collapseWhitespace(item.Content) + "\n")
		}
		if item.RawContent != "" {
			builder.WriteString("   正文：\n" + item.RawContent + "\n")
		}
	}
	return builder.String()
}

// collapseWhitespace keeps a snippet on one line so the numbered list stays scannable.
func collapseWhitespace(text string) string {
	return strings.Join(strings.Fields(text), " ")
}
