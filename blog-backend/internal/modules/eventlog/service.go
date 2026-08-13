package eventlog

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"dh-blog/internal/model"

	"github.com/sirupsen/logrus"
)

const (
	// publishBuffer decouples publishers from the database. Callers are
	// background workers and HTTP handlers whose real job is something else,
	// so publishing must never block on a write.
	publishBuffer = 256
	// retainedEvents bounds the table. The feed is a diagnostic aid, not an
	// audit trail, and this is roughly a year of a personal blog's activity.
	retainedEvents = 2000
	pruneInterval  = 6 * time.Hour
	// replayLimit caps how much history one reconnecting client may pull.
	replayLimit = 200
)

// Service persists events and pushes them to connected admin pages.
type Service struct {
	repo *repository
	hub  *hub

	queue    chan *Event
	stop     chan struct{}
	wg       sync.WaitGroup
	stopOnce sync.Once
	// logs is the server's own output, kept as scrollback and streamed live.
	// It is the other half of this page: the event feed says what jobs did,
	// the log says what the process was doing while they did it.
	logs       *logRing
	logQueue   chan LogLine
	attachOnce sync.Once
	// mu guards closed against publishers still running while the process
	// shuts down. Task workers outlive some of their callers, so a publish
	// racing the queue's close is a real ordering, not a theoretical one.
	mu     sync.RWMutex
	closed bool
	// lastID lets a fresh socket tell the client where the feed currently is,
	// without a second round trip.
	lastID atomic.Int64
}

func newService(repo *repository) *Service {
	s := &Service{
		repo:     repo,
		hub:      newHub(),
		queue:    make(chan *Event, publishBuffer),
		stop:     make(chan struct{}),
		logs:     newLogRing(logRingSize),
		logQueue: make(chan LogLine, logQueueSize),
	}

	// Seed the cursor so the first hello after a restart is not "0", which a
	// client would read as "I missed everything".
	var maxID int64
	if err := repo.db.Model(&Event{}).Select("COALESCE(MAX(id), 0)").Scan(&maxID).Error; err != nil {
		logrus.Warnf("读取事件日志游标失败: %v", err)
	}
	s.lastID.Store(maxID)

	s.wg.Add(3)
	go s.writeLoop()
	go s.pruneLoop()
	go s.startLogStream()
	return s
}

// Publish records one background event. It returns immediately; persistence
// and fan-out happen on the writer goroutine so a slow disk cannot stall the
// job that is reporting.
func (s *Service) Publish(event Event) {
	if event.CreatedAt.IsZero() {
		event.CreatedAt = model.JSONTime{Time: time.Now()}
	}
	record := event
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.closed {
		return
	}
	select {
	case s.queue <- &record:
	default:
		// Dropping is the right failure here: the feed is diagnostic, and
		// blocking a task worker to record that a task ran would be worse
		// than losing the line.
		logrus.Warnf("事件日志队列已满，丢弃一条事件: %s/%s", event.Source, event.Kind)
	}
}

// writeLoop persists queued events and only then broadcasts them, so every
// client sees the same ids and ordering the history endpoint would give it.
func (s *Service) writeLoop() {
	defer s.wg.Done()
	for event := range s.queue {
		if err := s.repo.create(context.Background(), event); err != nil {
			logrus.Warnf("写入事件日志失败: %v", err)
			continue
		}
		s.lastID.Store(event.ID)
		s.hub.broadcast(FrameEvent, event)
	}
}

func (s *Service) pruneLoop() {
	defer s.wg.Done()
	ticker := time.NewTicker(pruneInterval)
	defer ticker.Stop()
	for {
		select {
		case <-s.stop:
			return
		case <-ticker.C:
			removed, err := s.repo.prune(context.Background(), retainedEvents)
			if err != nil {
				logrus.Warnf("清理事件日志失败: %v", err)
				continue
			}
			if removed > 0 {
				logrus.Infof("已清理 %d 条过期的后台事件", removed)
			}
		}
	}
}

// Cursor reports the newest persisted event id.
func (s *Service) Cursor() int64 { return s.lastID.Load() }

// Clients reports how many admin pages are currently attached.
func (s *Service) Clients() int { return s.hub.clientCount() }

// HandleCommand registers a client→server frame handler. This is the seam for
// future two-way features; the socket itself needs no changes to gain one.
func (s *Service) HandleCommand(frameType string, handler CommandHandler) {
	s.hub.handle(frameType, handler)
}

// Shutdown drains the writer and closes every socket.
func (s *Service) Shutdown() {
	s.stopOnce.Do(func() {
		// Block new publishes before closing the queue, so a worker that is
		// mid-Publish cannot send on a closed channel.
		s.mu.Lock()
		s.closed = true
		s.mu.Unlock()

		close(s.stop)
		close(s.queue)
		s.hub.shutdown()
		s.wg.Wait()
	})
}
