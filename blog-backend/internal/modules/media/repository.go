package media

import (
	"context"
	"errors"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var errPlaylistNotFound = errors.New("歌单不存在")

type repository struct {
	db *gorm.DB
}

func newRepository(db *gorm.DB) *repository {
	return &repository{db: db}
}

// metadataByIDs loads cached rows for the given files. SQLite caps bound
// parameters, so large libraries are queried in chunks.
func (r *repository) metadataByIDs(ctx context.Context, ids []int) (map[int]Metadata, error) {
	result := make(map[int]Metadata, len(ids))
	const chunk = 500
	for start := 0; start < len(ids); start += chunk {
		end := min(start+chunk, len(ids))
		var rows []Metadata
		if err := r.db.WithContext(ctx).Where("file_id IN ?", ids[start:end]).Find(&rows).Error; err != nil {
			return nil, err
		}
		for _, row := range rows {
			result[row.FileID] = row
		}
	}
	return result, nil
}

func (r *repository) saveMetadata(ctx context.Context, rows []Metadata) error {
	if len(rows) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).
		Clauses(clause.OnConflict{UpdateAll: true}).
		CreateInBatches(rows, 200).Error
}

func (r *repository) findMetadata(ctx context.Context, fileID int) (*Metadata, error) {
	var row Metadata
	err := r.db.WithContext(ctx).Where("file_id = ?", fileID).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *repository) progressByUser(ctx context.Context, userID uint64, kind string) (map[int]Progress, error) {
	var rows []Progress
	if err := r.db.WithContext(ctx).Where("user_id = ? AND kind = ?", userID, kind).Find(&rows).Error; err != nil {
		return nil, err
	}
	result := make(map[int]Progress, len(rows))
	for _, row := range rows {
		result[row.FileID] = row
	}
	return result, nil
}

func (r *repository) findProgress(ctx context.Context, userID uint64, fileID int) (*Progress, error) {
	var row Progress
	err := r.db.WithContext(ctx).Where("user_id = ? AND file_id = ?", userID, fileID).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *repository) saveProgress(ctx context.Context, row *Progress) error {
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{UpdateAll: true}).Create(row).Error
}

func (r *repository) deleteProgress(ctx context.Context, userID uint64, fileID int) error {
	return r.db.WithContext(ctx).Where("user_id = ? AND file_id = ?", userID, fileID).Delete(&Progress{}).Error
}

func (r *repository) listPlaylists(ctx context.Context, userID uint64) ([]*Playlist, error) {
	var playlists []*Playlist
	if err := r.db.WithContext(ctx).Where("user_id = ?", userID).Order("id ASC").Find(&playlists).Error; err != nil {
		return nil, err
	}
	if err := r.attachTracks(ctx, playlists); err != nil {
		return nil, err
	}
	return playlists, nil
}

// attachTracks fills TrackIDs of the given playlists in stored order.
func (r *repository) attachTracks(ctx context.Context, playlists []*Playlist) error {
	if len(playlists) == 0 {
		return nil
	}
	ids := make([]int, len(playlists))
	byID := make(map[int]*Playlist, len(playlists))
	for i, playlist := range playlists {
		ids[i] = playlist.ID
		playlist.TrackIDs = []string{}
		byID[playlist.ID] = playlist
	}
	var items []PlaylistItem
	if err := r.db.WithContext(ctx).Where("playlist_id IN ?", ids).Order("playlist_id ASC, position ASC").Find(&items).Error; err != nil {
		return err
	}
	for _, item := range items {
		if playlist, ok := byID[item.PlaylistID]; ok {
			playlist.TrackIDs = append(playlist.TrackIDs, itoa(item.FileID))
		}
	}
	return nil
}

func (r *repository) findPlaylist(ctx context.Context, userID uint64, id int) (*Playlist, error) {
	var playlist Playlist
	err := r.db.WithContext(ctx).Where("id = ? AND user_id = ?", id, userID).First(&playlist).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, errPlaylistNotFound
	}
	if err != nil {
		return nil, err
	}
	return &playlist, nil
}

// savePlaylist writes the playlist row and, when trackIDs is non-nil,
// replaces its items in one transaction.
func (r *repository) savePlaylist(ctx context.Context, playlist *Playlist, trackIDs []int) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(playlist).Error; err != nil {
			return err
		}
		if trackIDs == nil {
			return nil
		}
		if err := tx.Where("playlist_id = ?", playlist.ID).Delete(&PlaylistItem{}).Error; err != nil {
			return err
		}
		if len(trackIDs) == 0 {
			return nil
		}
		items := make([]PlaylistItem, len(trackIDs))
		for i, fileID := range trackIDs {
			items[i] = PlaylistItem{PlaylistID: playlist.ID, FileID: fileID, Position: i}
		}
		return tx.CreateInBatches(items, 200).Error
	})
}

func (r *repository) deletePlaylist(ctx context.Context, userID uint64, id int) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		result := tx.Where("id = ? AND user_id = ?", id, userID).Delete(&Playlist{})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return errPlaylistNotFound
		}
		return tx.Where("playlist_id = ?", id).Delete(&PlaylistItem{}).Error
	})
}

func (r *repository) libraryFolders(ctx context.Context, userID uint64) ([]LibraryFolder, error) {
	var rows []LibraryFolder
	err := r.db.WithContext(ctx).Where("user_id = ?", userID).Find(&rows).Error
	return rows, err
}

func (r *repository) replaceLibraryFolders(ctx context.Context, userID uint64, rows []LibraryFolder) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("user_id = ?", userID).Delete(&LibraryFolder{}).Error; err != nil {
			return err
		}
		if len(rows) == 0 {
			return nil
		}
		return tx.Create(&rows).Error
	})
}
