package media

import (
	"context"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"dh-blog/internal/model"

	"github.com/dhowden/tag"
	"github.com/sirupsen/logrus"
)

// errBadRequest marks validation failures that the handler reports as 400.
var errBadRequest = errors.New("参数错误")

// errMediaNotFound marks a file that does not exist or is not the user's.
var errMediaNotFound = errors.New("文件不存在")

func badRequest(message string) error { return fmt.Errorf("%w: %s", errBadRequest, message) }

// parseWorkers bounds concurrent tag reads during a library refresh. Tag
// reading is I/O bound; a handful of workers hides disk latency without
// thrashing a spinning disk.
const parseWorkers = 4

type service struct {
	repo  *repository
	files FileCatalog
	// refreshMu serialises metadata refreshes so two tabs opening the library
	// at once do not both parse the same thousand files.
	refreshMu sync.Mutex
}

func newService(repo *repository, files FileCatalog) *service {
	return &service{repo: repo, files: files}
}

func itoa(id int) string { return strconv.Itoa(id) }

func parseID(raw string) (int, error) {
	id, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || id <= 0 {
		return 0, badRequest("无效的文件ID")
	}
	return id, nil
}

// ---------------------------------------------------------------------------
// Library scanning

type mediaFile struct {
	entry   *Entry
	id      int
	size    int64
	modTime int64
}

// Missing describes media files the drive index lists but the disk does not
// have (moved away outside DH-Blog, or a storage path that changed). They
// cannot be played, but silently dropping them leaves an empty library with
// no hint why, so the count and a few names are reported to the client.
type Missing struct {
	Count   int      `json:"count"`
	Samples []string `json:"samples"`
}

const missingSampleLimit = 5

type library struct {
	index   *libraryIndex
	files   []mediaFile
	scoped  bool
	missing Missing
}

// loadLibrary indexes the drive and returns the media files of one kind,
// limited to the user's chosen library folders when there are any.
func (s *service) loadLibrary(ctx context.Context, userID uint64, kind string) (*library, error) {
	entries, err := s.files.ListEntries(ctx, userID)
	if err != nil {
		return nil, err
	}
	index := newLibraryIndex(entries)

	folders, err := s.repo.libraryFolders(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("读取媒体库设置失败: %w", err)
	}
	roots := make(map[string]bool)
	for _, folder := range folders {
		// A chosen folder that has since been deleted no longer scopes
		// anything; if every chosen folder is gone the library falls back to
		// the whole drive instead of silently going empty.
		if entry, ok := index.byID[folder.FolderID]; folder.Kind == kind && ok && entry.IsFolder {
			roots[folder.FolderID] = true
		}
	}

	result := &library{index: index, missing: Missing{Samples: []string{}}}
	for i := range entries {
		entry := &entries[i]
		if entry.IsFolder || kindOf(entry.Name) != kind || !index.within(entry, roots) {
			continue
		}
		id, err := strconv.Atoi(entry.ID)
		if err != nil {
			continue
		}
		info, err := os.Stat(entry.Path)
		if err != nil || info.IsDir() {
			result.missing.Count++
			if len(result.missing.Samples) < missingSampleLimit {
				result.missing.Samples = append(result.missing.Samples, joinPath(index, entry))
			}
			continue
		}
		result.files = append(result.files, mediaFile{entry: entry, id: id, size: info.Size(), modTime: info.ModTime().UnixNano()})
	}
	result.scoped = len(roots) > 0
	if result.missing.Count > 0 {
		logrus.Warnf("媒体库: %d 个%s文件在索引中但磁盘上不存在，例如 %s", result.missing.Count, kind, strings.Join(result.missing.Samples, "、"))
	}
	return result, nil
}

func joinPath(index *libraryIndex, entry *Entry) string {
	if folder := joinSegments(index.folderPath(entry.ParentID)); folder != "" {
		return folder + "/" + entry.Name
	}
	return entry.Name
}

// fresh reports whether a cached row still describes the file on disk.
func (f mediaFile) fresh(row Metadata) bool {
	return row.Size == f.size && row.ModTime == f.modTime
}

// ensureAudioMetadata returns tag metadata for every file, reading tags only
// for files that are new or changed since the last scan.
func (s *service) ensureAudioMetadata(ctx context.Context, files []mediaFile) (map[int]Metadata, error) {
	s.refreshMu.Lock()
	defer s.refreshMu.Unlock()

	ids := make([]int, len(files))
	for i, file := range files {
		ids[i] = file.id
	}
	cached, err := s.repo.metadataByIDs(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("读取媒体元数据失败: %w", err)
	}

	var stale []mediaFile
	for _, file := range files {
		if row, ok := cached[file.id]; !ok || !row.Parsed || !file.fresh(row) {
			stale = append(stale, file)
		}
	}
	if len(stale) == 0 {
		return cached, nil
	}

	jobs := make(chan mediaFile)
	results := make(chan Metadata)
	var workers sync.WaitGroup
	for range min(parseWorkers, len(stale)) {
		workers.Go(func() {
			for file := range jobs {
				results <- parseAudioFile(file, cached[file.id])
			}
		})
	}
	go func() {
		for _, file := range stale {
			jobs <- file
		}
		close(jobs)
		workers.Wait()
		close(results)
	}()

	parsed := make([]Metadata, 0, len(stale))
	for row := range results {
		parsed = append(parsed, row)
		cached[row.FileID] = row
	}
	if err := s.repo.saveMetadata(ctx, parsed); err != nil {
		// The library is still usable from the freshly parsed rows; the next
		// request will just parse them again.
		logrus.Warnf("保存媒体元数据失败: %v", err)
	}
	return cached, nil
}

func parseAudioFile(file mediaFile, previous Metadata) Metadata {
	row := Metadata{
		FileID:    file.id,
		Kind:      kindAudio,
		Size:      file.size,
		ModTime:   file.modTime,
		Parsed:    true,
		UpdatedAt: time.Now(),
	}
	// A reported duration stays valid as long as the bytes did not change.
	if file.fresh(previous) {
		row.Duration = previous.Duration
	}
	tags, err := readAudioTags(file.entry.Path)
	if err != nil {
		// Unreadable tags are remembered as "parsed, nothing found" so a
		// broken file is not reopened on every library load.
		logrus.Debugf("读取音频标签失败 %s: %v", file.entry.Name, err)
		return row
	}
	row.Title = tags.Title
	row.Artist = tags.Artist
	row.Album = tags.Album
	row.AlbumArtist = tags.AlbumArtist
	row.Genre = tags.Genre
	row.Year = tags.Year
	row.TrackNo = tags.TrackNo
	row.DiscNo = tags.DiscNo
	row.HasPicture = tags.HasPicture
	row.HasLyrics = tags.HasLyrics
	return row
}

// ---------------------------------------------------------------------------
// Music

// Track is one song as the music app shows it.
type Track struct {
	ID               string          `json:"id"`
	Name             string          `json:"name"`
	Title            string          `json:"title"`
	Artist           string          `json:"artist"`
	Album            string          `json:"album"`
	AlbumArtist      string          `json:"album_artist,omitempty"`
	Genre            string          `json:"genre,omitempty"`
	Year             int             `json:"year,omitempty"`
	TrackNo          int             `json:"track_no,omitempty"`
	DiscNo           int             `json:"disc_no,omitempty"`
	Duration         float64         `json:"duration,omitempty"`
	Size             int64           `json:"size"`
	MimeType         string          `json:"mime_type,omitempty"`
	FolderID         string          `json:"folder_id"`
	FolderName       string          `json:"folder_name"`
	FolderPath       string          `json:"folder_path"`
	CoverFileID      string          `json:"cover_file_id,omitempty"`
	HasEmbeddedCover bool            `json:"has_embedded_cover"`
	HasLyrics        bool            `json:"has_lyrics"`
	PlayCount        int             `json:"play_count,omitempty"`
	LastPlayedAt     *model.JSONTime `json:"last_played_at,omitempty"`
	AddedAt          *model.JSONTime `json:"added_at,omitempty"`
}

// MusicLibrary is the payload of GET /api/media/music.
type MusicLibrary struct {
	Tracks []Track `json:"tracks"`
	// Scoped is true when the user limited the music library to chosen folders.
	Scoped  bool    `json:"scoped"`
	Missing Missing `json:"missing"`
}

const (
	unknownArtist = "未知艺人"
	unknownAlbum  = "未知专辑"
)

func (s *service) Music(ctx context.Context, userID uint64) (*MusicLibrary, error) {
	lib, err := s.loadLibrary(ctx, userID, kindAudio)
	if err != nil {
		return nil, err
	}
	index, files := lib.index, lib.files
	metadata, err := s.ensureAudioMetadata(ctx, files)
	if err != nil {
		return nil, err
	}
	progress, err := s.repo.progressByUser(ctx, userID, kindAudio)
	if err != nil {
		return nil, fmt.Errorf("读取播放记录失败: %w", err)
	}

	folderCovers := make(map[string]string)
	tracks := make([]Track, 0, len(files))
	for _, file := range files {
		entry := file.entry
		row := metadata[file.id]
		fallback := parseTrackName(entry.Name)
		siblings := index.siblings(entry)

		track := Track{
			ID:               entry.ID,
			Name:             entry.Name,
			Title:            firstNonEmpty(row.Title, fallback.Title, stem(entry.Name)),
			Artist:           firstNonEmpty(row.Artist, row.AlbumArtist, fallback.Artist, unknownArtist),
			Genre:            row.Genre,
			Year:             row.Year,
			TrackNo:          row.TrackNo,
			DiscNo:           row.DiscNo,
			Duration:         row.Duration,
			Size:             entry.Size,
			MimeType:         entry.MimeType,
			FolderID:         entry.ParentID,
			FolderName:       index.folderName(entry.ParentID),
			FolderPath:       joinSegments(index.folderPath(entry.ParentID)),
			HasEmbeddedCover: row.HasPicture,
			HasLyrics:        row.HasLyrics || hasSidecarLyrics(entry, siblings),
			AddedAt:          jsonTime(entry.CreatedAt),
		}
		if track.TrackNo == 0 {
			track.TrackNo = fallback.TrackNo
		}
		// Untagged albums are usually one folder per album, so the folder
		// name is a far better guess than lumping everything together.
		albumFallback := unknownAlbum
		if entry.ParentID != "" {
			albumFallback = track.FolderName
		}
		track.Album = firstNonEmpty(row.Album, albumFallback)
		// Left empty when untagged: the client then groups albums by folder,
		// which keeps a compilation without an album-artist tag together.
		track.AlbumArtist = row.AlbumArtist

		track.CoverFileID = findImage(siblings, []string{stem(entry.Name)})
		if track.CoverFileID == "" {
			cover, ok := folderCovers[entry.ParentID]
			if !ok {
				cover = findImage(siblings, albumCoverNames)
				folderCovers[entry.ParentID] = cover
			}
			track.CoverFileID = cover
		}
		if played, ok := progress[file.id]; ok {
			track.PlayCount = played.PlayCount
			track.LastPlayedAt = jsonTime(played.UpdatedAt)
			if track.Duration == 0 {
				track.Duration = played.Duration
			}
		}
		tracks = append(tracks, track)
	}
	return &MusicLibrary{Tracks: tracks, Scoped: lib.scoped, Missing: lib.missing}, nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func jsonTime(t time.Time) *model.JSONTime {
	if t.IsZero() {
		return nil
	}
	return &model.JSONTime{Time: t}
}

// ---------------------------------------------------------------------------
// Videos

// VideoProgress is where the user left off.
type VideoProgress struct {
	Position  float64         `json:"position"`
	Duration  float64         `json:"duration"`
	Finished  bool            `json:"finished"`
	UpdatedAt *model.JSONTime `json:"updated_at"`
}

// Video is one playable video as the cinema shows it.
type Video struct {
	ID       string  `json:"id"`
	Name     string  `json:"name"`
	Title    string  `json:"title"`
	Year     int     `json:"year,omitempty"`
	Season   int     `json:"season,omitempty"`
	Episode  int     `json:"episode,omitempty"`
	Size     int64   `json:"size"`
	MimeType string  `json:"mime_type,omitempty"`
	Duration float64 `json:"duration,omitempty"`
	// Collection groups episodes of one show; "Show/Season 1/x.mkv" belongs
	// to "Show", not to "Season 1".
	CollectionID   string          `json:"collection_id"`
	CollectionName string          `json:"collection_name"`
	FolderPath     string          `json:"folder_path"`
	PosterFileID   string          `json:"poster_file_id,omitempty"`
	BackdropFileID string          `json:"backdrop_file_id,omitempty"`
	Subtitles      []Subtitle      `json:"subtitles"`
	Progress       *VideoProgress  `json:"progress,omitempty"`
	AddedAt        *model.JSONTime `json:"added_at,omitempty"`
}

// VideoLibrary is the payload of GET /api/media/videos.
type VideoLibrary struct {
	Videos  []Video `json:"videos"`
	Scoped  bool    `json:"scoped"`
	Missing Missing `json:"missing"`
}

var seasonFolderPattern = regexp.MustCompile(`(?i)^(season\s*(\d{1,2})|s(\d{1,2})|第\s*(\d{1,2})\s*季|specials?|特别篇)$`)

func (s *service) Videos(ctx context.Context, userID uint64) (*VideoLibrary, error) {
	lib, err := s.loadLibrary(ctx, userID, kindVideo)
	if err != nil {
		return nil, err
	}
	index, files := lib.index, lib.files
	ids := make([]int, len(files))
	for i, file := range files {
		ids[i] = file.id
	}
	metadata, err := s.repo.metadataByIDs(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("读取媒体元数据失败: %w", err)
	}
	progress, err := s.repo.progressByUser(ctx, userID, kindVideo)
	if err != nil {
		return nil, fmt.Errorf("读取播放记录失败: %w", err)
	}

	videos := make([]Video, 0, len(files))
	for _, file := range files {
		entry := file.entry
		name := parseVideoName(entry.Name)
		siblings := index.siblings(entry)
		base := stem(entry.Name)

		video := Video{
			ID:           entry.ID,
			Name:         entry.Name,
			Title:        name.Title,
			Year:         name.Year,
			Season:       name.Season,
			Episode:      name.Episode,
			Size:         entry.Size,
			MimeType:     entry.MimeType,
			CollectionID: entry.ParentID,
			FolderPath:   joinSegments(index.folderPath(entry.ParentID)),
			Subtitles:    findSubtitles(entry, siblings),
			AddedAt:      jsonTime(entry.CreatedAt),
		}

		showFolder := entry.ParentID
		var showSiblings []*Entry
		if folder, ok := index.byID[entry.ParentID]; ok {
			if match := seasonFolderPattern.FindStringSubmatch(strings.TrimSpace(folder.Name)); match != nil {
				showFolder = folder.ParentID
				showSiblings = index.siblings(folder)
				if video.Season == 0 {
					video.Season, _ = strconv.Atoi(firstNonEmpty(match[2], match[3], match[4]))
				}
			}
		}
		video.CollectionID = showFolder
		video.CollectionName = index.folderName(showFolder)

		video.PosterFileID = firstNonEmpty(
			findImage(siblings, withSuffixes(base, posterSuffixes)),
			findImage(siblings, folderPosterNames),
			findImage(showSiblings, folderPosterNames),
		)
		video.BackdropFileID = firstNonEmpty(
			findImage(siblings, withSuffixes(base, backdropSuffixes)),
			findImage(siblings, folderBackdropNames),
			findImage(showSiblings, folderBackdropNames),
		)

		if row, ok := metadata[file.id]; ok && file.fresh(row) {
			video.Duration = row.Duration
		}
		if watched, ok := progress[file.id]; ok {
			video.Progress = &VideoProgress{
				Position:  watched.Position,
				Duration:  watched.Duration,
				Finished:  watched.Finished,
				UpdatedAt: jsonTime(watched.UpdatedAt),
			}
			if video.Duration == 0 {
				video.Duration = watched.Duration
			}
		}
		videos = append(videos, video)
	}
	return &VideoLibrary{Videos: videos, Scoped: lib.scoped, Missing: lib.missing}, nil
}

// ---------------------------------------------------------------------------
// Cover art and lyrics

func (s *service) ownedMedia(ctx context.Context, userID uint64, fileID string, kind string) (Entry, int, error) {
	id, err := parseID(fileID)
	if err != nil {
		return Entry{}, 0, err
	}
	entry, err := s.files.GetEntry(ctx, userID, fileID)
	if err != nil {
		logrus.Debugf("媒体文件 %s 不可用: %v", fileID, err)
		return Entry{}, 0, errMediaNotFound
	}
	if entryKind := kindOf(entry.Name); entryKind == "" || (kind != "" && entryKind != kind) {
		return Entry{}, 0, badRequest("不是可播放的媒体文件")
	}
	return entry, id, nil
}

// Cover returns the embedded artwork of a track, or nil when it has none.
func (s *service) Cover(ctx context.Context, userID uint64, fileID string) (*tag.Picture, error) {
	entry, _, err := s.ownedMedia(ctx, userID, fileID, kindAudio)
	if err != nil {
		return nil, err
	}
	return readPicture(entry.Path)
}

// Lyrics prefers a sidecar .lrc (usually time-synced) over embedded lyrics.
func (s *service) Lyrics(ctx context.Context, userID uint64, fileID string) (string, error) {
	entry, _, err := s.ownedMedia(ctx, userID, fileID, kindAudio)
	if err != nil {
		return "", err
	}
	want := strings.ToLower(stem(filepath.Base(entry.Path))) + ".lrc"
	dir := filepath.Dir(entry.Path)
	if siblings, err := os.ReadDir(dir); err == nil {
		for _, sibling := range siblings {
			if !sibling.IsDir() && strings.ToLower(sibling.Name()) == want {
				return readLyricsFile(filepath.Join(dir, sibling.Name()))
			}
		}
	}
	return readEmbeddedLyrics(entry.Path)
}

// ---------------------------------------------------------------------------
// Playback progress and durations

// finishedRatio is how far into a video counts as "watched": end credits
// should not leave a title stuck in 继续观看 forever.
const finishedRatio = 0.95

func validSeconds(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0) && value >= 0 && value < 1e7
}

// SaveProgress records the playback position. started marks the beginning of
// a new play so the play count only grows once per listen.
func (s *service) SaveProgress(ctx context.Context, userID uint64, fileID string, position, duration float64, started bool) error {
	if !validSeconds(position) || !validSeconds(duration) {
		return badRequest("无效的播放进度")
	}
	entry, id, err := s.ownedMedia(ctx, userID, fileID, "")
	if err != nil {
		return err
	}
	existing, err := s.repo.findProgress(ctx, userID, id)
	if err != nil {
		return fmt.Errorf("读取播放记录失败: %w", err)
	}
	row := &Progress{UserID: userID, FileID: id, Kind: kindOf(entry.Name)}
	if existing != nil {
		row.PlayCount = existing.PlayCount
		if duration == 0 {
			duration = existing.Duration
		}
	}
	if started {
		row.PlayCount++
	}
	if duration > 0 && position > duration {
		position = duration
	}
	row.Position = position
	row.Duration = duration
	row.Finished = duration > 0 && position >= duration*finishedRatio
	row.UpdatedAt = time.Now()
	if err := s.repo.saveProgress(ctx, row); err != nil {
		return fmt.Errorf("保存播放进度失败: %w", err)
	}
	return nil
}

func (s *service) RemoveProgress(ctx context.Context, userID uint64, fileID string) error {
	id, err := parseID(fileID)
	if err != nil {
		return err
	}
	return s.repo.deleteProgress(ctx, userID, id)
}

// DurationReport is one duration measured by the browser.
type DurationReport struct {
	ID       string  `json:"id"`
	Duration float64 `json:"duration"`
}

// maxDurationReports bounds one report batch; each item costs a lookup.
const maxDurationReports = 100

func (s *service) ReportDurations(ctx context.Context, userID uint64, reports []DurationReport) error {
	if len(reports) > maxDurationReports {
		reports = reports[:maxDurationReports]
	}
	rows := make([]Metadata, 0, len(reports))
	for _, report := range reports {
		if !validSeconds(report.Duration) || report.Duration == 0 {
			continue
		}
		entry, id, err := s.ownedMedia(ctx, userID, report.ID, "")
		if err != nil {
			continue
		}
		info, err := os.Stat(entry.Path)
		if err != nil {
			continue
		}
		file := mediaFile{entry: &entry, id: id, size: info.Size(), modTime: info.ModTime().UnixNano()}
		existing, err := s.repo.findMetadata(ctx, id)
		if err != nil {
			return fmt.Errorf("读取媒体元数据失败: %w", err)
		}
		row := Metadata{FileID: id, Kind: kindOf(entry.Name), Size: file.size, ModTime: file.modTime}
		// Keep parsed tags when they still describe this file; otherwise the
		// row is marked unparsed and the next library scan reads it again.
		if existing != nil && file.fresh(*existing) {
			row = *existing
		}
		row.Duration = report.Duration
		row.UpdatedAt = time.Now()
		rows = append(rows, row)
	}
	if err := s.repo.saveMetadata(ctx, rows); err != nil {
		return fmt.Errorf("保存媒体时长失败: %w", err)
	}
	return nil
}

// ---------------------------------------------------------------------------
// Playlists

const maxPlaylistNameRunes = 100

func (s *service) ListPlaylists(ctx context.Context, userID uint64) ([]*Playlist, error) {
	playlists, err := s.repo.listPlaylists(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("读取歌单失败: %w", err)
	}
	return playlists, nil
}

// PlaylistInput is the writable part of a playlist; nil fields stay unchanged.
type PlaylistInput struct {
	Name        *string   `json:"name"`
	Description *string   `json:"description"`
	TrackIDs    *[]string `json:"track_ids"`
}

func (s *service) CreatePlaylist(ctx context.Context, userID uint64, input PlaylistInput) (*Playlist, error) {
	if input.Name == nil {
		return nil, badRequest("歌单名称不能为空")
	}
	playlist := &Playlist{UserID: userID}
	if err := s.applyPlaylistInput(ctx, userID, playlist, input); err != nil {
		return nil, err
	}
	return playlist, nil
}

func (s *service) UpdatePlaylist(ctx context.Context, userID uint64, rawID string, input PlaylistInput) (*Playlist, error) {
	id, err := strconv.Atoi(rawID)
	if err != nil {
		return nil, badRequest("无效的歌单ID")
	}
	playlist, err := s.repo.findPlaylist(ctx, userID, id)
	if err != nil {
		return nil, err
	}
	if err := s.applyPlaylistInput(ctx, userID, playlist, input); err != nil {
		return nil, err
	}
	return playlist, nil
}

func (s *service) applyPlaylistInput(ctx context.Context, userID uint64, playlist *Playlist, input PlaylistInput) error {
	if input.Name != nil {
		name := strings.TrimSpace(*input.Name)
		if name == "" {
			return badRequest("歌单名称不能为空")
		}
		if utf8.RuneCountInString(name) > maxPlaylistNameRunes {
			return badRequest("歌单名称过长")
		}
		playlist.Name = name
	}
	if input.Description != nil {
		playlist.Description = strings.TrimSpace(*input.Description)
	}
	var trackIDs []int
	if input.TrackIDs != nil {
		ids, err := s.ownedTrackIDs(ctx, userID, *input.TrackIDs)
		if err != nil {
			return err
		}
		trackIDs = ids
		playlist.TrackIDs = make([]string, len(ids))
		for i, id := range ids {
			playlist.TrackIDs[i] = itoa(id)
		}
	}
	if err := s.repo.savePlaylist(ctx, playlist, trackIDs); err != nil {
		return fmt.Errorf("保存歌单失败: %w", err)
	}
	if trackIDs == nil {
		// Metadata-only update: report the stored tracks back unchanged.
		if err := s.repo.attachTracks(ctx, []*Playlist{playlist}); err != nil {
			return fmt.Errorf("读取歌单失败: %w", err)
		}
	}
	return nil
}

// ownedTrackIDs keeps the requested order, drops duplicates, and rejects ids
// that are not audio files of this user.
func (s *service) ownedTrackIDs(ctx context.Context, userID uint64, raw []string) ([]int, error) {
	entries, err := s.files.ListEntries(ctx, userID)
	if err != nil {
		return nil, err
	}
	audio := make(map[string]bool)
	for _, entry := range entries {
		if !entry.IsFolder && kindOf(entry.Name) == kindAudio {
			audio[entry.ID] = true
		}
	}
	seen := make(map[int]bool, len(raw))
	ids := make([]int, 0, len(raw))
	for _, value := range raw {
		if !audio[strings.TrimSpace(value)] {
			return nil, badRequest("歌单里包含不存在的歌曲")
		}
		id, _ := strconv.Atoi(strings.TrimSpace(value))
		if !seen[id] {
			seen[id] = true
			ids = append(ids, id)
		}
	}
	return ids, nil
}

func (s *service) DeletePlaylist(ctx context.Context, userID uint64, rawID string) error {
	id, err := strconv.Atoi(rawID)
	if err != nil {
		return badRequest("无效的歌单ID")
	}
	return s.repo.deletePlaylist(ctx, userID, id)
}

// ---------------------------------------------------------------------------
// Library settings

// FolderRef names a folder chosen as a library root.
type FolderRef struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Path string `json:"path"`
}

// LibrarySettings is the payload of GET /api/media/settings.
type LibrarySettings struct {
	MusicFolders []FolderRef `json:"music_folders"`
	VideoFolders []FolderRef `json:"video_folders"`
}

func (s *service) Settings(ctx context.Context, userID uint64) (*LibrarySettings, error) {
	entries, err := s.files.ListEntries(ctx, userID)
	if err != nil {
		return nil, err
	}
	index := newLibraryIndex(entries)
	rows, err := s.repo.libraryFolders(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("读取媒体库设置失败: %w", err)
	}
	settings := &LibrarySettings{MusicFolders: []FolderRef{}, VideoFolders: []FolderRef{}}
	for _, row := range rows {
		folder, ok := index.byID[row.FolderID]
		if !ok || !folder.IsFolder {
			continue
		}
		ref := FolderRef{ID: folder.ID, Name: folder.Name, Path: joinSegments(index.folderPath(folder.ID))}
		if row.Kind == kindAudio {
			settings.MusicFolders = append(settings.MusicFolders, ref)
		} else {
			settings.VideoFolders = append(settings.VideoFolders, ref)
		}
	}
	return settings, nil
}

func (s *service) SaveSettings(ctx context.Context, userID uint64, musicFolderIDs, videoFolderIDs []string) (*LibrarySettings, error) {
	entries, err := s.files.ListEntries(ctx, userID)
	if err != nil {
		return nil, err
	}
	index := newLibraryIndex(entries)
	var rows []LibraryFolder
	seen := make(map[string]bool)
	add := func(kind string, ids []string) error {
		for _, id := range ids {
			id = strings.TrimSpace(id)
			folder, ok := index.byID[id]
			if !ok || !folder.IsFolder {
				return badRequest("所选文件夹不存在")
			}
			if seen[kind+"/"+id] {
				continue
			}
			seen[kind+"/"+id] = true
			rows = append(rows, LibraryFolder{UserID: userID, Kind: kind, FolderID: id})
		}
		return nil
	}
	if err := add(kindAudio, musicFolderIDs); err != nil {
		return nil, err
	}
	if err := add(kindVideo, videoFolderIDs); err != nil {
		return nil, err
	}
	if err := s.repo.replaceLibraryFolders(ctx, userID, rows); err != nil {
		return nil, fmt.Errorf("保存媒体库设置失败: %w", err)
	}
	return s.Settings(ctx, userID)
}
