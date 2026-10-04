package media

import (
	"dh-blog/internal/router"

	"gorm.io/gorm"
)

// Dependencies are the collaborators the media library needs.
type Dependencies struct {
	DB    *gorm.DB
	Files FileCatalog
}

// Module is the personal cinema: a video library, a music library with
// playlists, and playback progress, all built on top of the drive's files.
// Streaming itself stays on /api/files/download, which already serves Range
// requests; this module only adds what a player needs around the bytes.
type Module struct {
	handler *handler
}

func New(deps Dependencies) *Module {
	return &Module{handler: newHandler(newService(newRepository(deps.DB), deps.Files))}
}

// MigrationModels declares the tables owned by this module.
func MigrationModels() []any {
	return []any{&Metadata{}, &Progress{}, &Playlist{}, &PlaylistItem{}, &LibraryFolder{}}
}

func (m *Module) RegisterRoutes(routes *router.Routes) {
	api := routes.AuthenticatedAPI("/api/media")
	api.GET("/videos", withUser(m.handler.Videos))
	api.GET("/music", withUser(m.handler.Music))
	api.GET("/cover/:id", withUser(m.handler.Cover))
	api.GET("/lyrics/:id", withUser(m.handler.Lyrics))
	api.PUT("/progress/:id", withUser(m.handler.SaveProgress))
	api.DELETE("/progress/:id", withUser(m.handler.RemoveProgress))
	api.POST("/durations", withUser(m.handler.ReportDurations))
	api.GET("/playlists", withUser(m.handler.ListPlaylists))
	api.POST("/playlists", withUser(m.handler.CreatePlaylist))
	api.PUT("/playlists/:id", withUser(m.handler.UpdatePlaylist))
	api.DELETE("/playlists/:id", withUser(m.handler.DeletePlaylist))
	api.GET("/settings", withUser(m.handler.Settings))
	api.PUT("/settings", withUser(m.handler.SaveSettings))
}
