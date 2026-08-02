package aigateway

import (
	"encoding/json"
	"fmt"
	"strings"

	"dh-blog/internal/platform/search"
)

// MCP (Model Context Protocol) over Streamable HTTP.
//
// The gateway only ever answers a request with a response, so it implements the
// JSON-only half of the transport: POST carries JSON-RPC, and GET/DELETE are
// answered with 405 as the spec requires of a server that offers no
// server-initiated stream. That keeps the endpoint stateless — no session id,
// no long-lived connection — which matters because the blog runs a single
// process behind whatever reverse proxy the operator happens to have.
const (
	mcpServerName    = "dh-blog-search-gateway"
	mcpServerVersion = "1.0.0"
	// 协商时优先回显客户端请求的版本，认不出来才退到这个
	mcpDefaultProtocolVersion = "2025-06-18"
	// MCP 请求体只有工具参数，比透传的原始 body 小得多
	maxMCPRequestBody = 64 << 10
)

var mcpSupportedProtocolVersions = map[string]bool{
	"2024-11-05": true,
	"2025-03-26": true,
	"2025-06-18": true,
}

// JSON-RPC 2.0 错误码
const (
	jsonRPCParseError     = -32700
	jsonRPCInvalidRequest = -32600
	jsonRPCMethodNotFound = -32601
	jsonRPCInvalidParams  = -32602
	jsonRPCInternalError  = -32603
)

type jsonRPCRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params"`
}

type jsonRPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

type jsonRPCResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Result  any             `json:"result,omitempty"`
	Error   *jsonRPCError   `json:"error,omitempty"`
}

// nullID is the id a response carries when the request was too broken to have one.
var nullID = json.RawMessage("null")

func rpcResult(id json.RawMessage, result any) jsonRPCResponse {
	if len(id) == 0 {
		id = nullID
	}
	return jsonRPCResponse{JSONRPC: "2.0", ID: id, Result: result}
}

func rpcFailure(id json.RawMessage, code int, message string) jsonRPCResponse {
	if len(id) == 0 {
		id = nullID
	}
	return jsonRPCResponse{JSONRPC: "2.0", ID: id, Error: &jsonRPCError{Code: code, Message: message}}
}

// isNotification reports whether the message expects no response. JSON-RPC
// notifications carry no id; MCP sends `notifications/initialized` this way.
func (r jsonRPCRequest) isNotification() bool {
	trimmed := strings.TrimSpace(string(r.ID))
	return trimmed == "" || trimmed == "null"
}

type mcpServerInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type mcpToolsCapability struct {
	ListChanged bool `json:"listChanged"`
}

type mcpCapabilities struct {
	Tools *mcpToolsCapability `json:"tools,omitempty"`
}

type mcpInitializeParams struct {
	ProtocolVersion string `json:"protocolVersion"`
}

type mcpInitializeResult struct {
	ProtocolVersion string          `json:"protocolVersion"`
	Capabilities    mcpCapabilities `json:"capabilities"`
	ServerInfo      mcpServerInfo   `json:"serverInfo"`
	Instructions    string          `json:"instructions,omitempty"`
}

type mcpTool struct {
	Name        string `json:"name"`
	Title       string `json:"title,omitempty"`
	Description string `json:"description"`
	InputSchema any    `json:"inputSchema"`
}

type mcpToolListResult struct {
	Tools []mcpTool `json:"tools"`
}

type mcpToolCallParams struct {
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments"`
}

type mcpTextContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type mcpToolResult struct {
	Content []mcpTextContent `json:"content"`
	IsError bool             `json:"isError,omitempty"`
}

func mcpText(text string) mcpToolResult {
	return mcpToolResult{Content: []mcpTextContent{{Type: "text", Text: text}}}
}

// mcpToolError reports a failed tool run. MCP wants execution failures inside
// the result rather than as a JSON-RPC error, so the model can read what went
// wrong and adjust instead of the client just surfacing a transport error.
func mcpToolError(text string) mcpToolResult {
	result := mcpText(text)
	result.IsError = true
	return result
}

// negotiateProtocolVersion echoes the client's version when it is one we speak.
func negotiateProtocolVersion(requested string) string {
	if mcpSupportedProtocolVersions[requested] {
		return requested
	}
	return mcpDefaultProtocolVersion
}

const mcpToolWebSearch = "web_search"

// webSearchTool builds the tool definition from the providers this particular
// key can actually reach, so the model never sees a `provider` value that the
// gateway would reject, and knows up front who can return an answer or page text.
func webSearchTool(providers []providerStatus) mcpTool {
	return mcpTool{
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
	builder.WriteString(fmt.Sprintf("来源 %s", result.Provider))
	if result.Meta.FallbackFrom != "" {
		builder.WriteString(fmt.Sprintf("（由 %s 回退）", result.Meta.FallbackFrom))
	}
	if result.Meta.Cached {
		builder.WriteString(" · 缓存命中")
	}
	builder.WriteString(fmt.Sprintf(" · 耗时 %dms\n", result.Meta.LatencyMS))

	if result.Answer != "" {
		builder.WriteString("\n答案：" + result.Answer + "\n")
	}

	if len(result.Results) == 0 {
		builder.WriteString("\n没有找到结果。")
		return builder.String()
	}

	for index, item := range result.Results {
		builder.WriteString(fmt.Sprintf("\n%d. %s\n   %s\n", index+1, item.Title, item.URL))
		if item.PublishedAt != nil {
			builder.WriteString(fmt.Sprintf("   发布于 %s\n", item.PublishedAt.Format("2006-01-02")))
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
