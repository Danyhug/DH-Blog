package task

import (
	"context"
	"testing"
	"time"

	articlemodule "dh-blog/internal/modules/article"
)

var _ articlemodule.TagTaskScheduler = (*TaskManager)(nil)

func TestTaskManagerDispatchesRegisteredTagGenerationTask(t *testing.T) {
	manager := NewTaskManager()
	received := make(chan struct {
		articleID int
		content   string
	}, 1)
	manager.RegisterTagGenerationHandler(func(_ context.Context, articleID int, content string) error {
		received <- struct {
			articleID int
			content   string
		}{articleID: articleID, content: content}
		return nil
	})
	manager.Start()
	t.Cleanup(manager.Stop)
	manager.SubmitTagGeneration(42, "article body")

	select {
	case got := <-received:
		if got.articleID != 42 || got.content != "article body" {
			t.Fatalf("dispatched task = %#v, want article 42 and original content", got)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("registered tag-generation handler was not invoked")
	}
}

func TestTaskManagerLifecycleIsIdempotent(t *testing.T) {
	manager := NewTaskManager()
	manager.Start()
	manager.Start()
	manager.Stop()
	manager.Stop()
}
