package article

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// Modes accepted by the batch summary endpoint.
const (
	BatchSummaryModeOverwrite = "overwrite" // 重新生成全部文章的摘要
	BatchSummaryModeFill      = "fill"      // 只补齐还没有摘要的文章
)

const (
	// batchSummaryWorkers caps concurrent AI calls. The batch deliberately runs
	// outside the shared task queue: it would otherwise hold one of the five
	// generic workers for minutes and starve single-article tasks, and the
	// dispatcher's 30s timeout plus 10 retries would fight a long batch.
	batchSummaryWorkers = 5
	// batchSummaryArticleTimeout bounds one article's round trip.
	batchSummaryArticleTimeout = 90 * time.Second
)

var (
	errBatchSummaryRunning = errors.New("已有摘要批量生成任务在运行，请等待其完成")
	errBatchSummaryNoAI    = errors.New("AI摘要服务未配置，请先在系统设置中填写 AI 接口信息")
)

// BatchSummaryStatus is the progress snapshot polled by the admin UI.
type BatchSummaryStatus struct {
	Running bool   `json:"running"`
	Mode    string `json:"mode"`
	Total   int    `json:"total"`
	Done    int    `json:"done"`
	Failed  int    `json:"failed"`
}

// batchSummaryRunner serialises batch runs and tracks their progress. The state
// is in-memory on purpose: a batch is a one-off admin action, so a restart just
// leaves the remaining articles to the next run.
type batchSummaryRunner struct {
	mu     sync.Mutex
	status BatchSummaryStatus
	cancel context.CancelFunc
	done   chan struct{}
}

func newBatchSummaryRunner() *batchSummaryRunner { return &batchSummaryRunner{} }

// begin claims the single batch slot and returns the context shared by workers.
func (r *batchSummaryRunner) begin(mode string, total int) (context.Context, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.status.Running {
		return nil, errBatchSummaryRunning
	}
	ctx, cancel := context.WithCancel(context.Background())
	r.status = BatchSummaryStatus{Running: true, Mode: mode, Total: total}
	r.cancel, r.done = cancel, make(chan struct{})
	return ctx, nil
}

func (r *batchSummaryRunner) record(failed bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.status.Done++
	if failed {
		r.status.Failed++
	}
}

func (r *batchSummaryRunner) finish() {
	r.mu.Lock()
	r.status.Running = false
	cancel, done := r.cancel, r.done
	r.cancel, r.done = nil, nil
	r.mu.Unlock()
	if cancel != nil {
		cancel()
	}
	if done != nil {
		close(done)
	}
}

// snapshot keeps reporting the last run after it ends so the UI's next poll can
// show the final counts.
func (r *batchSummaryRunner) snapshot() BatchSummaryStatus {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.status
}

// stop cancels an in-flight batch and waits for its workers, so shutdown does
// not abandon goroutines mid-write.
func (r *batchSummaryRunner) stop() {
	r.mu.Lock()
	cancel, done := r.cancel, r.done
	r.mu.Unlock()
	if cancel == nil {
		return
	}
	cancel()
	<-done
}

// StartBatchSummaryGeneration starts a background batch and reports how many
// articles it will process. A zero count means nothing matched the mode.
func (h *Handler) StartBatchSummaryGeneration(ctx context.Context, mode string) (int, error) {
	onlyMissing, err := batchSummaryOnlyMissing(mode)
	if err != nil {
		return 0, err
	}
	if h.ai == nil {
		return 0, errBatchSummaryNoAI
	}
	ids, err := h.articleRepository.FindSummaryTargetIDs(ctx, onlyMissing)
	if err != nil {
		return 0, err
	}
	if len(ids) == 0 {
		return 0, nil
	}
	batchCtx, err := h.batchSummary.begin(mode, len(ids))
	if err != nil {
		return 0, err
	}
	logrus.Infof("开始批量生成文章摘要: 模式 %s, 共 %d 篇", mode, len(ids))
	go h.runBatchSummary(batchCtx, ids)
	return len(ids), nil
}

func batchSummaryOnlyMissing(mode string) (bool, error) {
	switch mode {
	case BatchSummaryModeOverwrite:
		return false, nil
	case BatchSummaryModeFill:
		return true, nil
	default:
		return false, fmt.Errorf("%w: 未知的生成模式 %q", ErrInvalidParams, mode)
	}
}

func (h *Handler) runBatchSummary(ctx context.Context, ids []int) {
	start := time.Now()
	var wg sync.WaitGroup
	// Deferred LIFO: workers are awaited before the batch is marked complete.
	defer h.batchSummary.finish()
	defer wg.Wait()

	slots := make(chan struct{}, batchSummaryWorkers)
	for _, id := range ids {
		select {
		case <-ctx.Done():
			logrus.Warnf("摘要批量生成被中止，剩余文章保持原状 (已处理 %d/%d)",
				h.batchSummary.snapshot().Done, len(ids))
			return
		case slots <- struct{}{}:
		}
		wg.Add(1)
		go func(articleID int) {
			defer wg.Done()
			defer func() { <-slots }()
			err := h.generateSummaryForBatch(ctx, articleID)
			if err != nil {
				// 偶发上游慢响应（30s+ 排队）会造成一次失败，重试一次自愈；
				// 摘要失败不写回，重试是幂等的。
				logrus.Warnf("批量生成文章 %d 的摘要失败: %v, 将重试一次", articleID, err)
				err = h.generateSummaryForBatch(ctx, articleID)
			}
			if err != nil {
				logrus.Errorf("批量生成文章 %d 的摘要失败(重试后仍失败): %v", articleID, err)
			}
			h.batchSummary.record(err != nil)
		}(id)
	}
	wg.Wait()

	status := h.batchSummary.snapshot()
	logrus.Infof("摘要批量生成完成: 成功 %d 篇, 失败 %d 篇, 耗时 %v",
		status.Done-status.Failed, status.Failed, time.Since(start))
}

// generateSummaryForBatch reloads the article so a batch queued minutes ago
// summarises the current content rather than a stale snapshot.
func (h *Handler) generateSummaryForBatch(ctx context.Context, articleID int) error {
	articleCtx, cancel := context.WithTimeout(ctx, batchSummaryArticleTimeout)
	defer cancel()
	article, err := h.articleRepository.FindByID(articleCtx, articleID)
	if err != nil {
		return fmt.Errorf("读取文章 %d 失败: %w", articleID, err)
	}
	return h.ProcessSummaryGeneration(articleCtx, articleID, article.Content)
}

// StopBatchSummary is called on shutdown to drain an in-flight batch.
func (h *Handler) StopBatchSummary() { h.batchSummary.stop() }

func (h *Handler) StartBatchSummary(c *gin.Context) {
	var request struct {
		Mode string `json:"mode"`
	}
	if err := h.bindJSON(c, &request); err != nil {
		h.Error(c, err)
		return
	}
	total, err := h.StartBatchSummaryGeneration(c.Request.Context(), request.Mode)
	if err != nil {
		h.Error(c, err)
		return
	}
	h.SuccessWithData(c, struct {
		Started bool `json:"started"`
		Total   int  `json:"total"`
	}{Started: total > 0, Total: total})
}

func (h *Handler) GetBatchSummaryStatus(c *gin.Context) {
	h.SuccessWithData(c, h.batchSummary.snapshot())
}
