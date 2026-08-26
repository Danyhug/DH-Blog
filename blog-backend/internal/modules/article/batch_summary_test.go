package article

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"gorm.io/gorm"
)

// batchTestAI records how many summary calls overlap so the concurrency cap can
// be asserted, and can fail selected articles to exercise the failure counter.
type batchTestAI struct {
	mu     sync.Mutex
	active int
	peak   int
	calls  int
	hold   time.Duration
	fail   func(content string) bool
}

func (a *batchTestAI) GenerateTags(string, []string) ([]string, error) { return nil, nil }

func (a *batchTestAI) GenerateSummary(content string) (string, error) {
	a.mu.Lock()
	a.active++
	a.calls++
	if a.active > a.peak {
		a.peak = a.active
	}
	a.mu.Unlock()
	defer func() {
		a.mu.Lock()
		a.active--
		a.mu.Unlock()
	}()

	if a.hold > 0 {
		time.Sleep(a.hold)
	}
	if a.fail != nil && a.fail(content) {
		return "", errors.New("模拟的 AI 调用失败")
	}
	return "摘要-" + content, nil
}

func (a *batchTestAI) stats() (peak, calls int) {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.peak, a.calls
}

func newBatchTestHandler(t *testing.T, db *gorm.DB, ai AIService) *Handler {
	t.Helper()
	module, err := New(Dependencies{
		DB: db, Cache: newTestCache(), AI: ai, CommentCounter: testComments{}, Tasks: &testTasks{},
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(module.Shutdown)
	return module.handler
}

// seedSummaryArticles creates count articles; summaryFor decides each one's
// stored summary (empty means "never generated").
func seedSummaryArticles(t *testing.T, db *gorm.DB, count int, summaryFor func(i int) string) []int {
	t.Helper()
	ids := make([]int, 0, count)
	for i := 0; i < count; i++ {
		article := Article{
			Title:   fmt.Sprintf("文章%d", i),
			Content: fmt.Sprintf("正文%d", i),
			Summary: summaryFor(i),
		}
		if err := db.Create(&article).Error; err != nil {
			t.Fatalf("seed article %d: %v", i, err)
		}
		ids = append(ids, article.ID)
	}
	return ids
}

func waitForBatchSummary(t *testing.T, h *Handler) BatchSummaryStatus {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		if status := h.batchSummary.snapshot(); !status.Running {
			return status
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("batch summary run did not finish in time")
	return BatchSummaryStatus{}
}

func storedSummary(t *testing.T, db *gorm.DB, id int) string {
	t.Helper()
	var article Article
	if err := db.First(&article, id).Error; err != nil {
		t.Fatalf("load article %d: %v", id, err)
	}
	return article.Summary
}

func TestBatchSummaryFillOnlyTouchesArticlesWithoutSummary(t *testing.T) {
	db := openArticleTestDB(t)
	// 0 和 2 已有摘要，1 从未生成过
	ids := seedSummaryArticles(t, db, 3, func(i int) string {
		if i == 1 {
			return ""
		}
		return fmt.Sprintf("已有摘要%d", i)
	})
	handler := newBatchTestHandler(t, db, &batchTestAI{})

	total, err := handler.StartBatchSummaryGeneration(context.Background(), BatchSummaryModeFill)
	if err != nil {
		t.Fatal(err)
	}
	if total != 1 {
		t.Fatalf("fill mode targeted %d articles, want 1", total)
	}
	status := waitForBatchSummary(t, handler)
	if status.Done != 1 || status.Failed != 0 {
		t.Fatalf("status = %+v, want 1 done and 0 failed", status)
	}

	if got := storedSummary(t, db, ids[1]); got != "摘要-正文1" {
		t.Fatalf("missing summary was not filled: %q", got)
	}
	for _, i := range []int{0, 2} {
		want := fmt.Sprintf("已有摘要%d", i)
		if got := storedSummary(t, db, ids[i]); got != want {
			t.Fatalf("existing summary of article %d was overwritten: %q", i, got)
		}
	}
}

func TestBatchSummaryOverwriteRegeneratesEveryArticle(t *testing.T) {
	db := openArticleTestDB(t)
	ids := seedSummaryArticles(t, db, 3, func(i int) string { return fmt.Sprintf("旧摘要%d", i) })
	handler := newBatchTestHandler(t, db, &batchTestAI{})

	total, err := handler.StartBatchSummaryGeneration(context.Background(), BatchSummaryModeOverwrite)
	if err != nil {
		t.Fatal(err)
	}
	if total != 3 {
		t.Fatalf("overwrite mode targeted %d articles, want 3", total)
	}
	status := waitForBatchSummary(t, handler)
	if status.Done != 3 || status.Failed != 0 {
		t.Fatalf("status = %+v, want 3 done and 0 failed", status)
	}
	for i, id := range ids {
		want := fmt.Sprintf("摘要-正文%d", i)
		if got := storedSummary(t, db, id); got != want {
			t.Fatalf("article %d summary = %q, want %q", i, got, want)
		}
	}
}

func TestBatchSummaryCapsConcurrencyAndKeepsGoingAfterFailures(t *testing.T) {
	db := openArticleTestDB(t)
	const articleCount = 12
	ids := seedSummaryArticles(t, db, articleCount, func(int) string { return "" })
	// 每 3 篇失败一次：正文3/6/9 共 3 篇
	ai := &batchTestAI{hold: 30 * time.Millisecond, fail: func(content string) bool {
		return strings.HasSuffix(content, "3") || strings.HasSuffix(content, "6") || strings.HasSuffix(content, "9")
	}}
	handler := newBatchTestHandler(t, db, ai)

	if _, err := handler.StartBatchSummaryGeneration(context.Background(), BatchSummaryModeFill); err != nil {
		t.Fatal(err)
	}
	status := waitForBatchSummary(t, handler)

	if status.Total != articleCount || status.Done != articleCount {
		t.Fatalf("status = %+v, want all %d articles processed", status, articleCount)
	}
	if status.Failed != 3 {
		t.Fatalf("status.Failed = %d, want 3", status.Failed)
	}
	peak, calls := ai.stats()
	if peak > batchSummaryWorkers {
		t.Fatalf("concurrent AI calls peaked at %d, want at most %d", peak, batchSummaryWorkers)
	}
	if peak < 2 {
		t.Fatalf("concurrent AI calls peaked at %d, batch did not run in parallel", peak)
	}
	if calls != articleCount+3 {
		t.Fatalf("AI was called %d times, want %d (%d failed articles retried once)",
			calls, articleCount+3, 3)
	}
	// 失败的文章保持无摘要，成功的文章已写回
	for i, id := range ids {
		got := storedSummary(t, db, id)
		failed := i == 3 || i == 6 || i == 9
		if failed && got != "" {
			t.Fatalf("failed article %d unexpectedly got summary %q", i, got)
		}
		if !failed && got != fmt.Sprintf("摘要-正文%d", i) {
			t.Fatalf("article %d summary = %q", i, got)
		}
	}
}

func TestBatchSummaryRejectsSecondRunWhileBusy(t *testing.T) {
	db := openArticleTestDB(t)
	seedSummaryArticles(t, db, 6, func(int) string { return "" })
	handler := newBatchTestHandler(t, db, &batchTestAI{hold: 100 * time.Millisecond})

	if _, err := handler.StartBatchSummaryGeneration(context.Background(), BatchSummaryModeFill); err != nil {
		t.Fatal(err)
	}
	if _, err := handler.StartBatchSummaryGeneration(context.Background(), BatchSummaryModeOverwrite); !errors.Is(err, errBatchSummaryRunning) {
		t.Fatalf("second batch error = %v, want errBatchSummaryRunning", err)
	}
	waitForBatchSummary(t, handler)

	// 上一批结束后可以再次提交
	if _, err := handler.StartBatchSummaryGeneration(context.Background(), BatchSummaryModeOverwrite); err != nil {
		t.Fatalf("batch after completion was rejected: %v", err)
	}
	waitForBatchSummary(t, handler)
}

func TestBatchSummaryRejectsInvalidModeAndMissingAI(t *testing.T) {
	db := openArticleTestDB(t)
	seedSummaryArticles(t, db, 1, func(int) string { return "" })

	handler := newBatchTestHandler(t, db, &batchTestAI{})
	if _, err := handler.StartBatchSummaryGeneration(context.Background(), "all"); !errors.Is(err, ErrInvalidParams) {
		t.Fatalf("invalid mode error = %v, want ErrInvalidParams", err)
	}

	withoutAI := newBatchTestHandler(t, openArticleTestDB(t), nil)
	if _, err := withoutAI.StartBatchSummaryGeneration(context.Background(), BatchSummaryModeFill); !errors.Is(err, errBatchSummaryNoAI) {
		t.Fatalf("missing AI error = %v, want errBatchSummaryNoAI", err)
	}
}

func TestBatchSummaryReportsNothingToDo(t *testing.T) {
	db := openArticleTestDB(t)
	seedSummaryArticles(t, db, 2, func(i int) string { return fmt.Sprintf("摘要%d", i) })
	ai := &batchTestAI{}
	handler := newBatchTestHandler(t, db, ai)

	total, err := handler.StartBatchSummaryGeneration(context.Background(), BatchSummaryModeFill)
	if err != nil {
		t.Fatal(err)
	}
	if total != 0 {
		t.Fatalf("fill mode targeted %d articles, want 0", total)
	}
	if status := handler.batchSummary.snapshot(); status.Running {
		t.Fatal("an empty batch should not be marked running")
	}
	if _, calls := ai.stats(); calls != 0 {
		t.Fatalf("AI was called %d times for an empty batch", calls)
	}
}

func TestNormalizeSummaryClampsToPromptLimit(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"trims surrounding space", "  一段摘要。 ", "一段摘要。"},
		{
			"keeps summaries within the limit",
			strings.Repeat("字", summaryMaxRunes),
			strings.Repeat("字", summaryMaxRunes),
		},
		{
			"cuts back to the last sentence end",
			strings.Repeat("字", 100) + "。" + strings.Repeat("尾", 40),
			strings.Repeat("字", 100) + "。",
		},
		{
			"hard cuts when no sentence ends late enough",
			strings.Repeat("字", 200),
			strings.Repeat("字", summaryMaxRunes),
		},
		{
			"ignores a sentence end in the first half",
			"短句。" + strings.Repeat("字", 200),
			"短句。" + strings.Repeat("字", summaryMaxRunes-3),
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := normalizeSummary(test.in)
			if got != test.want {
				t.Fatalf("normalizeSummary() = %q (%d runes), want %q (%d runes)",
					got, len([]rune(got)), test.want, len([]rune(test.want)))
			}
			if len([]rune(got)) > summaryMaxRunes {
				t.Fatalf("normalizeSummary() returned %d runes, over the %d cap", len([]rune(got)), summaryMaxRunes)
			}
		})
	}
}
