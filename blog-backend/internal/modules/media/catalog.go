package media

import (
	"context"
	"strings"
	"time"
)

// Entry is one file or folder of the drive, as the media library sees it.
type Entry struct {
	ID        string
	ParentID  string
	Name      string
	IsFolder  bool
	Size      int64
	MimeType  string
	Path      string // absolute path on disk
	CreatedAt time.Time
}

// FileCatalog is the only drive access the media library needs. The files
// module implements it through an adapter in the composition root.
type FileCatalog interface {
	ListEntries(ctx context.Context, userID uint64) ([]Entry, error)
	// GetEntry returns one file owned by the user, checked to exist on disk.
	GetEntry(ctx context.Context, userID uint64, fileID string) (Entry, error)
}

// Segment is one folder on the path from the drive root.
type Segment struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// maxFolderDepth guards path resolution against a corrupted index that
// contains a parent cycle.
const maxFolderDepth = 64

// libraryIndex is an in-memory view of the user's whole drive built once per
// request, so sibling lookups (covers, subtitles, lyrics) cost a map access.
type libraryIndex struct {
	byID     map[string]*Entry
	children map[string][]*Entry
	paths    map[string][]Segment
}

func newLibraryIndex(entries []Entry) *libraryIndex {
	index := &libraryIndex{
		byID:     make(map[string]*Entry, len(entries)),
		children: make(map[string][]*Entry),
		paths:    make(map[string][]Segment),
	}
	for i := range entries {
		entry := &entries[i]
		entry.ParentID = normalizeParentID(entry.ParentID)
		index.byID[entry.ID] = entry
		index.children[entry.ParentID] = append(index.children[entry.ParentID], entry)
	}
	return index
}

func normalizeParentID(parentID string) string {
	trimmed := strings.TrimSpace(parentID)
	if trimmed == "." {
		return ""
	}
	return trimmed
}

// folderPath returns the segments from the root down to folderID inclusive.
// The root itself ("") has an empty path.
func (ix *libraryIndex) folderPath(folderID string) []Segment {
	if folderID == "" {
		return nil
	}
	if cached, ok := ix.paths[folderID]; ok {
		return cached
	}
	var reversed []Segment
	visited := make(map[string]bool)
	for current := folderID; current != "" && len(reversed) < maxFolderDepth; {
		if visited[current] {
			break
		}
		visited[current] = true
		folder, ok := ix.byID[current]
		if !ok || !folder.IsFolder {
			break
		}
		reversed = append(reversed, Segment{ID: folder.ID, Name: folder.Name})
		current = folder.ParentID
	}
	path := make([]Segment, len(reversed))
	for i := range reversed {
		path[i] = reversed[len(reversed)-1-i]
	}
	ix.paths[folderID] = path
	return path
}

// folderName is the display name of a folder; the root is "我的网盘".
func (ix *libraryIndex) folderName(folderID string) string {
	if folder, ok := ix.byID[folderID]; ok && folder.IsFolder {
		return folder.Name
	}
	return "我的网盘"
}

func joinSegments(path []Segment) string {
	names := make([]string, len(path))
	for i, segment := range path {
		names[i] = segment.Name
	}
	return strings.Join(names, "/")
}

// within reports whether the entry sits in one of roots (at any depth).
// An empty roots set means "the whole drive".
func (ix *libraryIndex) within(entry *Entry, roots map[string]bool) bool {
	if len(roots) == 0 {
		return true
	}
	for _, segment := range ix.folderPath(entry.ParentID) {
		if roots[segment.ID] {
			return true
		}
	}
	return false
}

// siblings returns the files next to entry, folders excluded.
func (ix *libraryIndex) siblings(entry *Entry) []*Entry {
	all := ix.children[entry.ParentID]
	files := make([]*Entry, 0, len(all))
	for _, candidate := range all {
		if !candidate.IsFolder {
			files = append(files, candidate)
		}
	}
	return files
}
