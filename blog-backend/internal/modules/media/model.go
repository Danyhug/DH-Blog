package media

import (
	"time"

	"dh-blog/internal/model"
)

// Metadata caches what was read out of a media file so the library does not
// reopen every file on each request. Size and ModTime identify the file
// version the row describes; when either changes the tags are read again.
type Metadata struct {
	FileID  int    `gorm:"primaryKey;autoIncrement:false"`
	Kind    string `gorm:"type:varchar(16)"`
	Size    int64
	ModTime int64
	// Parsed is false for rows that only exist because a client reported a
	// duration before the library ever read the file's tags.
	Parsed      bool
	Title       string `gorm:"type:varchar(512)"`
	Artist      string `gorm:"type:varchar(512)"`
	Album       string `gorm:"type:varchar(512)"`
	AlbumArtist string `gorm:"type:varchar(512)"`
	Genre       string `gorm:"type:varchar(255)"`
	Year        int
	TrackNo     int
	DiscNo      int
	HasPicture  bool
	HasLyrics   bool
	// Duration is reported by the browser (seconds). Tag parsing cannot give a
	// reliable duration for every container, the media element always can.
	Duration  float64
	UpdatedAt time.Time
}

func (Metadata) TableName() string { return "media_metadata" }

// Progress is one user's playback position in one file. It drives the
// "继续观看" row for videos and "最近播放" for music.
type Progress struct {
	UserID    uint64 `gorm:"primaryKey;autoIncrement:false"`
	FileID    int    `gorm:"primaryKey;autoIncrement:false"`
	Kind      string `gorm:"type:varchar(16);index"`
	Position  float64
	Duration  float64
	Finished  bool
	PlayCount int
	UpdatedAt time.Time `gorm:"index"`
}

func (Progress) TableName() string { return "media_progress" }

// Playlist is a user-curated music list ("指定"的歌单). Auto-detected lists
// (by folder / album / artist) are derived on the client from the track list
// and never stored.
type Playlist struct {
	model.BaseModel
	UserID      uint64   `gorm:"index" json:"-"`
	Name        string   `gorm:"type:varchar(255)" json:"name"`
	Description string   `gorm:"type:varchar(1024)" json:"description"`
	TrackIDs    []string `gorm:"-" json:"track_ids"`
}

func (Playlist) TableName() string { return "media_playlists" }

// PlaylistItem keeps playlist order explicitly; a playlist is always written
// as a whole, so Position is simply the index in the submitted list.
type PlaylistItem struct {
	ID         int `gorm:"primaryKey;autoIncrement"`
	PlaylistID int `gorm:"index"`
	FileID     int
	Position   int
}

func (PlaylistItem) TableName() string { return "media_playlist_items" }

// LibraryFolder restricts a library kind to chosen folders. With no rows for
// a kind the whole drive is scanned, which is the "自动识别" default.
type LibraryFolder struct {
	UserID   uint64 `gorm:"primaryKey;autoIncrement:false"`
	Kind     string `gorm:"primaryKey;type:varchar(16)"`
	FolderID string `gorm:"primaryKey;type:varchar(64)"`
}

func (LibraryFolder) TableName() string { return "media_library_folders" }

const (
	kindAudio = "audio"
	kindVideo = "video"
)
