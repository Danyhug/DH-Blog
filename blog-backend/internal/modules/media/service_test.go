package media

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

// fakeCatalog serves a fixed drive whose files live in a temp directory.
type fakeCatalog struct {
	entries []Entry
}

func (f *fakeCatalog) ListEntries(context.Context, uint64) ([]Entry, error) {
	return append([]Entry(nil), f.entries...), nil
}

func (f *fakeCatalog) GetEntry(_ context.Context, _ uint64, fileID string) (Entry, error) {
	for _, entry := range f.entries {
		if entry.ID == fileID && !entry.IsFolder {
			return entry, nil
		}
	}
	return Entry{}, errors.New("文件不存在")
}

type driveBuilder struct {
	t       *testing.T
	root    string
	catalog *fakeCatalog
}

func newDrive(t *testing.T) *driveBuilder {
	return &driveBuilder{t: t, root: t.TempDir(), catalog: &fakeCatalog{}}
}

func (d *driveBuilder) folder(id, parentID, name string) {
	d.catalog.entries = append(d.catalog.entries, Entry{ID: id, ParentID: parentID, Name: name, IsFolder: true})
}

func (d *driveBuilder) file(id, parentID, name string) {
	path := filepath.Join(d.root, id+"_"+name)
	if err := os.WriteFile(path, make([]byte, 512), 0o644); err != nil {
		d.t.Fatal(err)
	}
	d.catalog.entries = append(d.catalog.entries, Entry{ID: id, ParentID: parentID, Name: name, Size: 512, Path: path})
}

func newTestService(t *testing.T, catalog FileCatalog) *service {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(MigrationModels()...); err != nil {
		t.Fatalf("migrate media: %v", err)
	}
	return newService(newRepository(db), catalog)
}

func TestMusicFallsBackToFileAndFolderNames(t *testing.T) {
	drive := newDrive(t)
	drive.folder("10", "", "叶惠美")
	drive.file("11", "10", "03 - 周杰伦 - 晴天.mp3")
	drive.file("12", "10", "cover.jpg")
	drive.file("13", "10", "03 - 周杰伦 - 晴天.lrc")
	drive.file("14", "", "loose.flac")
	drive.file("15", "", "notes.txt")
	svc := newTestService(t, drive.catalog)

	library, err := svc.Music(context.Background(), 1)
	if err != nil {
		t.Fatalf("Music: %v", err)
	}
	if len(library.Tracks) != 2 {
		t.Fatalf("tracks = %+v, want 2", library.Tracks)
	}
	byID := map[string]Track{}
	for _, track := range library.Tracks {
		byID[track.ID] = track
	}
	song := byID["11"]
	if song.Title != "晴天" || song.Artist != "周杰伦" || song.Album != "叶惠美" || song.TrackNo != 3 {
		t.Errorf("song = %+v", song)
	}
	if song.CoverFileID != "12" || !song.HasLyrics || song.FolderPath != "叶惠美" {
		t.Errorf("song sidecars = %+v", song)
	}
	loose := byID["14"]
	if loose.Album != unknownAlbum || loose.Artist != unknownArtist || loose.FolderName != "我的网盘" {
		t.Errorf("loose = %+v", loose)
	}
}

func TestMusicHonoursLibraryFolders(t *testing.T) {
	drive := newDrive(t)
	drive.folder("1", "", "音乐")
	drive.folder("2", "1", "专辑")
	drive.folder("3", "", "录音")
	drive.file("4", "2", "a.mp3")
	drive.file("5", "3", "b.mp3")
	svc := newTestService(t, drive.catalog)
	ctx := context.Background()

	if _, err := svc.SaveSettings(ctx, 1, []string{"1"}, nil); err != nil {
		t.Fatalf("SaveSettings: %v", err)
	}
	library, err := svc.Music(ctx, 1)
	if err != nil {
		t.Fatalf("Music: %v", err)
	}
	if !library.Scoped || len(library.Tracks) != 1 || library.Tracks[0].ID != "4" {
		t.Fatalf("scoped library = %+v", library)
	}
	if _, err := svc.SaveSettings(ctx, 1, []string{"4"}, nil); !errors.Is(err, errBadRequest) {
		t.Errorf("SaveSettings with a file id: err = %v, want bad request", err)
	}
}

func TestVideosGroupSeasonFoldersUnderTheShow(t *testing.T) {
	drive := newDrive(t)
	drive.folder("1", "", "Friends")
	drive.folder("2", "1", "Season 2")
	drive.file("3", "1", "poster.jpg")
	drive.file("4", "2", "Friends.E03.mkv")
	drive.file("5", "2", "Friends.E03.chs.srt")
	svc := newTestService(t, drive.catalog)

	library, err := svc.Videos(context.Background(), 1)
	if err != nil {
		t.Fatalf("Videos: %v", err)
	}
	if len(library.Videos) != 1 {
		t.Fatalf("videos = %+v", library.Videos)
	}
	video := library.Videos[0]
	if video.CollectionID != "1" || video.CollectionName != "Friends" || video.Season != 2 || video.Episode != 3 {
		t.Errorf("video grouping = %+v", video)
	}
	if video.PosterFileID != "3" || len(video.Subtitles) != 1 || video.Subtitles[0].ID != "5" {
		t.Errorf("video sidecars = %+v", video)
	}
}

func TestProgressMarksFinishedAndCountsPlays(t *testing.T) {
	drive := newDrive(t)
	drive.file("1", "", "movie.mp4")
	svc := newTestService(t, drive.catalog)
	ctx := context.Background()

	if err := svc.SaveProgress(ctx, 1, "1", 30, 100, true); err != nil {
		t.Fatalf("SaveProgress: %v", err)
	}
	if err := svc.SaveProgress(ctx, 1, "1", 96, 0, false); err != nil {
		t.Fatalf("SaveProgress: %v", err)
	}
	library, err := svc.Videos(ctx, 1)
	if err != nil {
		t.Fatalf("Videos: %v", err)
	}
	progress := library.Videos[0].Progress
	if progress == nil || progress.Position != 96 || progress.Duration != 100 || !progress.Finished {
		t.Fatalf("progress = %+v", progress)
	}
	if err := svc.SaveProgress(ctx, 1, "99", 1, 1, false); !errors.Is(err, errMediaNotFound) {
		t.Errorf("SaveProgress on unknown file: err = %v", err)
	}
}

func TestReportedDurationSurvivesTagParsing(t *testing.T) {
	drive := newDrive(t)
	drive.file("1", "", "song.mp3")
	svc := newTestService(t, drive.catalog)
	ctx := context.Background()

	if err := svc.ReportDurations(ctx, 1, []DurationReport{{ID: "1", Duration: 215.5}}); err != nil {
		t.Fatalf("ReportDurations: %v", err)
	}
	library, err := svc.Music(ctx, 1)
	if err != nil {
		t.Fatalf("Music: %v", err)
	}
	if got := library.Tracks[0].Duration; got != 215.5 {
		t.Errorf("duration = %v, want 215.5", got)
	}
}

func TestPlaylistsKeepOrderAndRejectForeignTracks(t *testing.T) {
	drive := newDrive(t)
	drive.file("1", "", "a.mp3")
	drive.file("2", "", "b.mp3")
	drive.file("3", "", "c.mp4")
	svc := newTestService(t, drive.catalog)
	ctx := context.Background()

	name := "通勤"
	tracks := []string{"2", "1", "2"}
	playlist, err := svc.CreatePlaylist(ctx, 1, PlaylistInput{Name: &name, TrackIDs: &tracks})
	if err != nil {
		t.Fatalf("CreatePlaylist: %v", err)
	}
	if got := playlist.TrackIDs; len(got) != 2 || got[0] != "2" || got[1] != "1" {
		t.Errorf("track ids = %v, want [2 1]", got)
	}

	video := []string{"3"}
	if _, err := svc.UpdatePlaylist(ctx, 1, itoa(playlist.ID), PlaylistInput{TrackIDs: &video}); !errors.Is(err, errBadRequest) {
		t.Errorf("adding a video: err = %v, want bad request", err)
	}

	renamed := "夜跑"
	updated, err := svc.UpdatePlaylist(ctx, 1, itoa(playlist.ID), PlaylistInput{Name: &renamed})
	if err != nil {
		t.Fatalf("UpdatePlaylist: %v", err)
	}
	if updated.Name != "夜跑" || len(updated.TrackIDs) != 2 {
		t.Errorf("renamed playlist = %+v", updated)
	}

	if _, err := svc.UpdatePlaylist(ctx, 2, itoa(playlist.ID), PlaylistInput{Name: &renamed}); !errors.Is(err, errPlaylistNotFound) {
		t.Errorf("other user's playlist: err = %v, want not found", err)
	}
	if err := svc.DeletePlaylist(ctx, 1, itoa(playlist.ID)); err != nil {
		t.Fatalf("DeletePlaylist: %v", err)
	}
	playlists, err := svc.ListPlaylists(ctx, 1)
	if err != nil || len(playlists) != 0 {
		t.Errorf("playlists after delete = %+v, err %v", playlists, err)
	}
}

func TestLibraryReportsIndexedFilesMissingFromDisk(t *testing.T) {
	drive := newDrive(t)
	drive.folder("1", "", "电影")
	drive.file("2", "1", "在的.mp4")
	drive.catalog.entries = append(drive.catalog.entries, Entry{ID: "3", ParentID: "1", Name: "丢了.mp4", Path: filepath.Join(drive.root, "gone.mp4")})
	svc := newTestService(t, drive.catalog)

	library, err := svc.Videos(context.Background(), 1)
	if err != nil {
		t.Fatalf("Videos: %v", err)
	}
	if len(library.Videos) != 1 || library.Missing.Count != 1 || library.Missing.Samples[0] != "电影/丢了.mp4" {
		t.Fatalf("library = %+v", library)
	}
}
