package eventlog

import (
	"dh-blog/internal/router"

	"gorm.io/gorm"
)

// Module owns the background-event feed: its table, its history API and the
// WebSocket the admin page listens on.
//
// The feed exists because most of this application's real work happens after
// the response has been sent — AI tag and summary generation, the WebDAV disk
// rescan, the gateway's hourly usage sync. All of it used to fail into the
// server log alone.
type Module struct {
	service *Service
	handler *handler
}

// New assembles the module. db must be non-nil, as everywhere else at the
// composition root. It also hijacks the standard logger, which is why there is
// exactly one of these per process.
func New(db *gorm.DB) *Module {
	service := newService(newRepository(db))
	service.AttachLogrus()
	return &Module{service: service, handler: newHandler(service)}
}

// MigrationModels declares the database tables owned by this module.
func MigrationModels() []any { return []any{&Event{}} }

// Service exposes the publisher to application-level collaborators.
func (m *Module) Service() *Service { return m.service }

// TaskObserver returns the adapter the generic task queue reports through.
func (m *Module) TaskObserver() *TaskObserver { return &TaskObserver{service: m.service} }

// SyncReporter returns the adapter the files module reports disk syncs through.
func (m *Module) SyncReporter() *SyncReporter { return &SyncReporter{service: m.service} }

// GatewayReporter returns the adapter the AI gateway reports rotation changes
// through.
func (m *Module) GatewayReporter() *GatewayReporter { return &GatewayReporter{service: m.service} }

func (m *Module) RegisterRoutes(routes *router.Routes) {
	events := routes.AdminAPI.Group("/events")
	events.GET("", m.handler.list)
	events.GET("/since", m.handler.replay)
	events.GET("/logs", m.handler.logs)
	// The socket rides the same JWT middleware as the rest of AdminAPI; the
	// browser cannot set an Authorization header on a WebSocket, so the token
	// arrives as ?token=, which extractToken already accepts.
	events.GET("/ws", m.handler.stream)
}

// Shutdown drains the writer and closes every attached socket.
func (m *Module) Shutdown() {
	if m != nil && m.service != nil {
		m.service.Shutdown()
	}
}
