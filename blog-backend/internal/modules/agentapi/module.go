package agentapi

import (
	"fmt"
	"time"

	"dh-blog/internal/platform/mcp"
	"dh-blog/internal/router"

	"gorm.io/gorm"
)

// GrantService is the authorization surface the MCP tools will consume in
// Task 6. It is defined here, on the consumer side of agentapi, so the tool
// layer depends on a narrow interface rather than on the repository.
type GrantService interface {
	Grant(articleID int, note string) (*EditGrant, error)
	Validate(articleID int, token string) (*EditGrant, error)
	Revoke(id int) error
	Reveal(id int) (string, error)
	List(now time.Time) ([]EditGrant, error)
}

// Dependencies wires agentapi into the application. Article, image and event
// log collaborators arrive in later steps (Tasks 6-8).
type Dependencies struct {
	DB *gorm.DB
}

// Module owns temporary edit grants and, from Task 6 on, the agent-facing MCP
// tools that consume them.
type Module struct {
	service *grantService
	handler *grantHandler
}

// New builds the module.
func New(deps Dependencies) (*Module, error) {
	if deps.DB == nil {
		return nil, fmt.Errorf("agentapi: DB is required")
	}
	service := newGrantService(&grantRepository{db: deps.DB})
	return &Module{service: service, handler: newGrantHandler(service)}, nil
}

// RegisterRoutes mounts the grant administration surface under the JWT-protected
// admin group.
func (m *Module) RegisterRoutes(routes *router.Routes) {
	admin := routes.AdminAPI.Group("/agent")
	admin.POST("/grants", m.handler.createGrant)
	admin.GET("/grants", m.handler.listGrants)
	admin.GET("/grants/:id/reveal", m.handler.revealGrant)
	admin.DELETE("/grants/:id", m.handler.revokeGrant)
}

// MigrationModels declares the tables owned by this module.
func (m *Module) MigrationModels() []any {
	return []any{&EditGrant{}}
}

// Grants exposes the authorization service to the tool layer.
func (m *Module) Grants() GrantService { return m.service }

// MCPTools returns the agent-facing tools. Placeholder until Task 6 fills in
// the five write tools.
func (m *Module) MCPTools() []mcp.Tool { return nil }
