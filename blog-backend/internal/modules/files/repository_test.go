package files

import (
	"context"
	"testing"
)

func TestRepositoryListsAndCountsFilesWithinUserDirectory(t *testing.T) {
	db := openTestDB(t)
	repository := newRepository(db)
	ctx := context.Background()

	records := []*File{
		{UserID: 1, ParentID: "", Name: "z-folder", IsFolder: true, StoragePath: "z-folder"},
		{UserID: 1, ParentID: "", Name: "a-folder", IsFolder: true, StoragePath: "a-folder"},
		{UserID: 1, ParentID: "", Name: "a.txt", StoragePath: "a.txt", Size: 1},
		{UserID: 2, ParentID: "", Name: "other.txt", StoragePath: "other.txt", Size: 1},
	}
	for _, record := range records {
		if err := repository.Create(ctx, record); err != nil {
			t.Fatalf("create %s: %v", record.Name, err)
		}
	}

	got, err := repository.ListByParentID(ctx, 1, "")
	if err != nil {
		t.Fatalf("list files: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("got %d records, want 3", len(got))
	}
	if got[0].Name != "a-folder" || got[1].Name != "z-folder" || got[2].Name != "a.txt" {
		t.Fatalf("unexpected folder-first ordering: %q, %q, %q", got[0].Name, got[1].Name, got[2].Name)
	}

	byPath, err := repository.FindByPath(ctx, 1, "a.txt")
	if err != nil || byPath.Name != "a.txt" {
		t.Fatalf("find by path: file=%v err=%v", byPath, err)
	}
	byName, err := repository.FindByUserIDAndName(ctx, 1, "", "a-folder")
	if err != nil || byName.Name != "a-folder" {
		t.Fatalf("find by name: file=%v err=%v", byName, err)
	}

	count, err := repository.CountByUserID(ctx, 1)
	if err != nil || count != 3 {
		t.Fatalf("count user files: count=%d err=%v", count, err)
	}
	if err := repository.TruncateFiles(ctx); err != nil {
		t.Fatalf("truncate files: %v", err)
	}
	count, err = repository.CountByUserID(ctx, 1)
	if err != nil || count != 0 {
		t.Fatalf("count after truncate: count=%d err=%v", count, err)
	}
}
