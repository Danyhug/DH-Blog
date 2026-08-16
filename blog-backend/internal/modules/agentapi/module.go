package agentapi

import (
	"context"
	"fmt"
	"time"

	"dh-blog/internal/modules/article"
	"dh-blog/internal/platform/mcp"
	"dh-blog/internal/router"

	"gorm.io/gorm"
)

// GrantService is the authorization surface the MCP tools consume. It is
// defined here, on the consumer side of agentapi, so the tool layer depends on
// a narrow interface rather than on the repository.
type GrantService interface {
	Grant(articleID int, note string) (*EditGrant, error)
	Validate(articleID int, token string) (*EditGrant, error)
	Revoke(id int) error
	Reveal(id int) (string, error)
	List(now time.Time) ([]EditGrant, error)
}

// Articles is the article-persistence port the content tools need. Its shape
// matches article.ContentService exactly, and the DTO types are the exported
// article package types — agentapi → article is an allowed dependency
// direction.
type Articles interface {
	List(ctx context.Context, keyword string, page, pageSize int) ([]article.ArticleBrief, int64, error)
	Get(ctx context.Context, id int) (*article.ArticleDetail, error)
	Create(ctx context.Context, input article.CreateInput) (int, error)
	Update(ctx context.Context, input article.UpdateInput) error
}

// Images stores an uploaded image under the blog's protected directory. The
// concrete implementation is wired at the composition root (registry); the
// files module's own tests cover real disk persistence.
type Images interface {
	SaveBlogImage(ctx context.Context, fileName string, data []byte) (url string, err error)
}

// TaskSubmitter hands work to the background queue: tag and summary generation
// for articles created without a summary.
type TaskSubmitter interface {
	SubmitTagGeneration(articleID int, content string)
	SubmitSummaryGeneration(articleID int, content string)
}

// ContentReporter records agent write actions for the backend event feed.
// Task 8 wires the eventlog implementation; until then New() substitutes a
// no-op, so every call site stays nil-safe.
type ContentReporter interface {
	ArticleCreated(agent, title string, articleID int)
	ArticleUpdated(agent, title string, articleID int, viaGrant bool)
	ArticleUpdateDenied(agent, title string, articleID int, reason string)
}

// noopContentReporter swallows events. Deliberate: the report stream does not
// exist yet, and a missing reporter must never block a write tool.
type noopContentReporter struct{}

func (noopContentReporter) ArticleCreated(string, string, int)              {}
func (noopContentReporter) ArticleUpdated(string, string, int, bool)        {}
func (noopContentReporter) ArticleUpdateDenied(string, string, int, string) {}

// Dependencies wires agentapi into the application. Every collaborator is a
// narrow port defined here, so the module never reaches into another module's
// internals.
type Dependencies struct {
	DB       *gorm.DB
	Articles Articles
	Images   Images
	Tasks    TaskSubmitter
	Events   ContentReporter
}

// Module owns temporary edit grants and the agent-facing MCP tools.
type Module struct {
	service *grantService
	handler *grantHandler
	tools   []mcp.Tool
}

// New builds the module. DB, Articles, Images and Tasks are required; Events
// may be nil and falls back to a no-op reporter.
func New(deps Dependencies) (*Module, error) {
	if deps.DB == nil {
		return nil, fmt.Errorf("agentapi: DB is required")
	}
	if deps.Articles == nil {
		return nil, fmt.Errorf("agentapi: Articles is required")
	}
	if deps.Images == nil {
		return nil, fmt.Errorf("agentapi: Images is required")
	}
	if deps.Tasks == nil {
		return nil, fmt.Errorf("agentapi: Tasks is required")
	}
	if deps.Events == nil {
		deps.Events = noopContentReporter{}
	}
	service := newGrantService(&grantRepository{db: deps.DB})
	return &Module{
		service: service,
		handler: newGrantHandler(service),
		tools: []mcp.Tool{
			&listArticlesTool{articles: deps.Articles},
			&getArticleTool{articles: deps.Articles},
			&createArticleTool{articles: deps.Articles, tasks: deps.Tasks, events: deps.Events},
			&updateArticleTool{articles: deps.Articles, grants: service, events: deps.Events},
			&uploadImageTool{images: deps.Images},
		},
	}, nil
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

// MCPTools returns the five agent-facing content tools.
func (m *Module) MCPTools() []mcp.Tool { return m.tools }
