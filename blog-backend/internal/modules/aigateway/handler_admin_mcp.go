package aigateway

import (
	"sort"

	"dh-blog/internal/platform/mcp"

	"github.com/gin-gonic/gin"
)

// The MCP catalog exists so the admin page never has to restate what the
// server offers. Tools are contributed by other modules and filtered per key at
// request time; hardcoding their names, descriptions or scopes in the frontend
// would mean every new tool needs a matching frontend edit to become visible.
// Here the page renders whatever is actually mounted.

type mcpParamView struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Description string `json:"description"`
	Required    bool   `json:"required"`
}

type mcpToolView struct {
	Name  string `json:"name"`
	Title string `json:"title"`
	// Description is verbatim what the model is told, not an admin-facing
	// paraphrase: the point of the page is to show what the agent actually sees.
	Description string         `json:"description"`
	Scope       string         `json:"scope"`
	Params      []mcpParamView `json:"params"`
}

type mcpCatalogView struct {
	ServerName   string            `json:"serverName"`
	Version      string            `json:"version"`
	Instructions string            `json:"instructions"`
	Endpoint     string            `json:"endpoint"`
	Scopes       []ScopeDescriptor `json:"scopes"`
	Tools        []mcpToolView     `json:"tools"`
}

// listMCPTools renders every mounted tool with the scope that gates it.
//
// Definitions are rendered without a key in context, so web_search reports the
// providers the gateway has enabled overall rather than any one key's subset —
// which is the right answer for a catalog.
func (h *handler) listMCPTools(c *gin.Context) {
	ctx := c.Request.Context()
	mounted := append([]mcp.Tool{h.webSearch}, h.extraTools...)
	tools := make([]mcpToolView, 0, len(mounted))
	for _, tool := range mounted {
		definition := tool.Definition(ctx)
		tools = append(tools, mcpToolView{
			Name:        definition.Name,
			Title:       definition.Title,
			Description: definition.Description,
			Scope:       toolScope(tool),
			Params:      schemaParams(definition.InputSchema),
		})
	}
	adminSuccess(c, mcpCatalogView{
		ServerName: mcpServerName,
		Version:    mcpServerVersion,
		// Instructions are trimmed per key at initialize; the catalog shows the
		// full-capability version, which is the only one that mentions every
		// tool listed below it.
		Instructions: mcpInstructionsFor(mounted),
		Endpoint:     "/api/gateway/v1/mcp",
		Scopes:       ScopeCatalog(),
		Tools:        tools,
	})
}

// schemaParams flattens a tool's JSON Schema into the flat parameter list the
// page shows. Schemas are built by hand in Go, so anything that does not match
// the expected shape is skipped rather than reported: a catalog is not the
// place to fail a request over an odd schema.
func schemaParams(schema any) []mcpParamView {
	root, ok := schema.(map[string]any)
	if !ok {
		return nil
	}
	properties, ok := root["properties"].(map[string]any)
	if !ok {
		return nil
	}
	required := requiredSet(root["required"])

	params := make([]mcpParamView, 0, len(properties))
	for name, raw := range properties {
		field, _ := raw.(map[string]any)
		typeName, _ := field["type"].(string)
		description, _ := field["description"].(string)
		params = append(params, mcpParamView{
			Name:        name,
			Type:        typeName,
			Description: description,
			Required:    required[name],
		})
	}
	// Map iteration is random and this response feeds a table; required first,
	// then alphabetical, so the same tool always renders the same way.
	sort.Slice(params, func(i, j int) bool {
		if params[i].Required != params[j].Required {
			return params[i].Required
		}
		return params[i].Name < params[j].Name
	})
	return params
}

// requiredSet accepts both shapes a hand-written schema uses for "required":
// a []string literal, or the []any a round trip through JSON produces.
func requiredSet(raw any) map[string]bool {
	set := map[string]bool{}
	switch names := raw.(type) {
	case []string:
		for _, name := range names {
			set[name] = true
		}
	case []any:
		for _, item := range names {
			if name, ok := item.(string); ok {
				set[name] = true
			}
		}
	}
	return set
}
