package aigateway

import (
	"fmt"

	"dh-blog/internal/router"
)

// Module owns the AI gateway's persistence, routing policy, agent-facing API
// and admin API.
type Module struct {
	service *Service
	handler *handler
	enabled bool
}

// New builds the module. Provider rows are seeded on first run so the admin
// page has something to configure.
func New(deps Dependencies) (*Module, error) {
	if deps.DB == nil {
		return nil, fmt.Errorf("aigateway: DB is required")
	}
	if deps.Cache == nil {
		return nil, fmt.Errorf("aigateway: cache is required")
	}
	service, err := newService(deps)
	if err != nil {
		return nil, fmt.Errorf("初始化 AI 网关模块失败: %w", err)
	}
	return &Module{service: service, handler: newHandler(service), enabled: true}, nil
}

// Service exposes gateway operations to application-level collaborators.
func (m *Module) Service() *Service { return m.service }

// SetEnabled toggles the agent-facing endpoints. The admin surface stays
// available so the gateway can be configured before it is switched on.
func (m *Module) SetEnabled(enabled bool) { m.enabled = enabled }

func (m *Module) RegisterRoutes(routes *router.Routes) {
	if m.enabled {
		// The gateway authenticates with its own API keys, so it is mounted on
		// the engine directly rather than under the JWT-protected groups.
		gateway := routes.Engine.Group("/api/gateway/v1")
		gateway.Use(m.handler.authenticate)
		gateway.POST("/search", m.handler.Search)
		gateway.GET("/search", m.handler.SearchGET)
		gateway.GET("/providers", m.handler.Providers)

		// Native passthrough: same auth, metering and accounting, but the
		// request and response stay in the provider's own format so an existing
		// SDK only needs its base URL repointed here.
		gateway.POST("/tavily/search", m.handler.TavilyPassthrough)
		gateway.GET("/brave/web/search", m.handler.BravePassthrough)
		gateway.POST("/exa/search", m.handler.ExaPassthrough)

		// MCP: lets Claude Code and other MCP clients mount the gateway as a
		// tool server. GET/DELETE exist only to answer 405 — the gateway never
		// opens a server-initiated stream.
		gateway.POST("/mcp", m.handler.MCP)
		gateway.GET("/mcp", m.handler.MCPNotAllowed)
		gateway.DELETE("/mcp", m.handler.MCPNotAllowed)
	}

	admin := routes.AdminAPI.Group("/gateway")
	admin.GET("/settings", m.handler.getSettings)
	admin.PUT("/settings", m.handler.updateSettings)
	admin.GET("/providers", m.handler.listProviders)
	admin.PUT("/providers/:name", m.handler.updateProvider)
	admin.POST("/providers/:name/test", m.handler.testProvider)
	// Mounted outside /providers so it cannot collide with the :name wildcard.
	admin.POST("/usage/sync", m.handler.syncUsage)
	admin.POST("/providers/:name/keys", m.handler.createProviderKey)
	admin.PUT("/providers/:name/keys/:id", m.handler.updateProviderKey)
	admin.DELETE("/providers/:name/keys/:id", m.handler.deleteProviderKey)
	admin.GET("/keys", m.handler.listAPIKeys)
	admin.POST("/keys", m.handler.createAPIKey)
	admin.GET("/keys/:id/reveal", m.handler.revealAPIKey)
	admin.PUT("/keys/:id", m.handler.updateAPIKey)
	admin.DELETE("/keys/:id", m.handler.deleteAPIKey)
	admin.GET("/logs", m.handler.listLogs)
	admin.GET("/stats", m.handler.stats)
}

// Shutdown drains the asynchronous log writer.
func (m *Module) Shutdown() {
	if m != nil && m.service != nil {
		m.service.Shutdown()
	}
}
