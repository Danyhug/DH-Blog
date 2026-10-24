package aigateway

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"dh-blog/internal/platform/mcp"

	"github.com/sirupsen/logrus"
)

const mcpToolWebFetch = "web_fetch"

// Piece sizes for web_fetch. A page is returned in pieces because a long
// document read whole would crowd everything else out of the model's context;
// the default is about one long article, the cap about a short book chapter.
const (
	mcpFetchDefaultLength = 20000
	mcpFetchMaxLength     = 100000
)

// webFetchTool adapts the gateway's page fetch to MCP. Like web_search it goes
// through the service, so it shares the key's rate limit, quota, cache and
// request log; the log's endpoint reads mcp/fetch.
type webFetchTool struct {
	service *Service
}

func (t *webFetchTool) Name() string { return mcpToolWebFetch }

func (t *webFetchTool) Definition(ctx context.Context) mcp.Definition {
	providers, err := t.service.ProviderStatuses(ctx, mcpKeyFromContext(ctx))
	if err != nil {
		logrus.Warnf("获取 MCP 工具定义的供应商状态失败: %v", err)
		providers = nil
	}
	available := false
	for _, provider := range providers {
		available = available || provider.SupportsFetch
	}
	return webFetchDefinition(available)
}

func webFetchDefinition(available bool) mcp.Definition {
	var description strings.Builder
	description.WriteString("读取一个网页的正文，返回干净的 Markdown。会渲染 JavaScript、处理常见的反爬页面，" +
		"静态抓取拿不到内容的页面也能读。\n")
	description.WriteString("先用 web_search 找到链接，需要细读其中某一条时再调用它；只是确认一个事实时，" +
		"web_search 的摘要通常已经够了。要好几条结果的全文时，用 web_search 的 include_raw_content 一次拿齐更省。\n")
	fmt.Fprintf(&description, "长页面分段返回，每段默认 %d 字符。结果末尾会给出下一段的 start_index，"+
		"原样传入即可接着读；同一页面的后续分段走缓存，不再计费。", mcpFetchDefaultLength)
	if !available {
		description.WriteString("\n注意：当前没有可用的网页抓取供应商，调用会直接失败。")
	}
	return mcp.Definition{
		Name:        mcpToolWebFetch,
		Title:       "读取网页",
		Description: description.String(),
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"url": map[string]any{
					"type":        "string",
					"description": "要读取的网页地址，只支持公网的 http / https。",
					"format":      "uri",
					"maxLength":   maxFetchURLLength,
				},
				"max_length": map[string]any{
					"type":        "integer",
					"description": fmt.Sprintf("这一段最多返回多少字符，默认 %d。", mcpFetchDefaultLength),
					"minimum":     1,
					"maximum":     mcpFetchMaxLength,
				},
				"start_index": map[string]any{
					"type":        "integer",
					"description": "从第几个字符开始读，默认 0。读下一段时填上一次结果给出的值。",
					"minimum":     0,
				},
			},
			"required":             []string{"url"},
			"additionalProperties": false,
		},
	}
}

func (t *webFetchTool) Call(ctx context.Context, args json.RawMessage) mcp.Result {
	var body fetchBody
	if len(args) > 0 {
		if err := json.Unmarshal(args, &body); err != nil {
			return mcp.ToolError("arguments 解析失败: " + err.Error())
		}
	}
	// no_cache is not in the schema: a model has no business deciding to pay
	// for a page twice.
	body.NoCache = false
	req, invalid := normalizeFetch(body, mcpFetchDefaultLength, mcpFetchMaxLength)
	if invalid != nil {
		return mcp.ToolError(invalid.Message)
	}

	result, err := t.service.Fetch(ctx, mcpKeyFromContext(ctx), req, mcpClientIPFromContext(ctx), "mcp/fetch")
	if err != nil {
		return mcp.ToolError(strings.Replace(mcpFailureText(err), "搜索失败", "读取失败", 1))
	}
	return mcp.Text(renderFetchResult(result))
}

// renderFetchResult puts what the model needs to judge the page — where it
// really came from, whether it is an error page, how much is left — above the
// content, and the instruction for the next piece where it will be looked for.
func renderFetchResult(result FetchResult) string {
	var builder strings.Builder
	fmt.Fprintf(&builder, "来源 %s", result.Provider)
	if result.Meta.FallbackFrom != "" {
		fmt.Fprintf(&builder, "（由 %s 回退）", result.Meta.FallbackFrom)
	}
	if result.Meta.Cached {
		builder.WriteString(" · 缓存命中")
	}
	fmt.Fprintf(&builder, " · 耗时 %dms\n", result.Meta.LatencyMS)
	if result.Title != "" {
		builder.WriteString("标题：" + result.Title + "\n")
	}
	builder.WriteString("链接：" + result.URL + "\n")
	if result.StatusCode >= 400 {
		fmt.Fprintf(&builder, "注意：目标网站返回 HTTP %d，下面可能是错误页而不是正文。\n", result.StatusCode)
	}

	if result.TotalLength == 0 {
		builder.WriteString("\n页面没有可提取的正文。")
		return builder.String()
	}
	end := result.StartIndex + len([]rune(result.Content))
	if result.StartIndex > 0 || result.NextStartIndex > 0 {
		fmt.Fprintf(&builder, "第 %d–%d 字符，共 %d 字符\n", result.StartIndex, end, result.TotalLength)
	}
	if result.Content == "" {
		fmt.Fprintf(&builder, "\nstart_index 已超出页面长度（共 %d 字符）。", result.TotalLength)
		return builder.String()
	}
	builder.WriteString("\n" + result.Content + "\n")
	if result.NextStartIndex > 0 {
		fmt.Fprintf(&builder, "\n（未完，还剩 %d 字符。继续读请把 start_index 设为 %d）",
			result.TotalLength-result.NextStartIndex, result.NextStartIndex)
	}
	return builder.String()
}
