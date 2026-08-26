package eventlog

import (
	"fmt"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

const (
	// logRingSize is how much scrollback a page gets on first paint. Small
	// enough to hold in memory without thinking about it, long enough to cover
	// what just went wrong.
	logRingSize = 1000
	// logQueueSize decouples the logging goroutine from the fan-out. Overflow
	// drops lines, which is the only acceptable failure: logging must never
	// block the thing that is logging.
	logQueueSize = 512
	// logSnapshotLimit caps one scrollback request.
	logSnapshotLimit = 500
)

// LogLine is one line of server output as the admin console sees it.
type LogLine struct {
	// Seq is a monotonic id used by the client to deduplicate the scrollback
	// it fetched against the lines the socket pushes.
	Seq int64 `json:"seq"`
	// Time is the compact clock-only stamp shown in the console; Timestamp
	// carries the full date so a hover tooltip can show which day it was.
	Time      string `json:"time"`
	Timestamp string `json:"timestamp"`
	Level     string `json:"level"`
	Message   string `json:"message"`
	Fields    map[string]string `json:"fields,omitempty"`
}

// logRing is a fixed-size circular scrollback buffer.
type logRing struct {
	mu    sync.RWMutex
	lines []LogLine
	// next is the write position; the buffer is full once it has wrapped.
	next   int
	filled bool
	seq    int64
}

func newLogRing(size int) *logRing {
	return &logRing{lines: make([]LogLine, size)}
}

// append stamps the line with the next sequence number and stores it.
func (r *logRing) append(line LogLine) LogLine {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.seq++
	line.Seq = r.seq
	r.lines[r.next] = line
	r.next++
	if r.next == len(r.lines) {
		r.next = 0
		r.filled = true
	}
	return line
}

// snapshot returns up to limit of the most recent lines, oldest first.
func (r *logRing) snapshot(limit int) []LogLine {
	r.mu.RLock()
	defer r.mu.RUnlock()

	total := r.next
	if r.filled {
		total = len(r.lines)
	}
	if limit > total {
		limit = total
	}
	out := make([]LogLine, 0, limit)
	// Walk backwards from the newest, then reverse, so a partially filled
	// ring and a wrapped one are handled by the same arithmetic.
	for i := 0; i < limit; i++ {
		index := (r.next - 1 - i + len(r.lines)) % len(r.lines)
		out = append(out, r.lines[index])
	}
	for left, right := 0, len(out)-1; left < right; left, right = left+1, right-1 {
		out[left], out[right] = out[right], out[left]
	}
	return out
}

// cursor reports the newest sequence number issued.
func (r *logRing) cursor() int64 {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.seq
}

// logHook feeds logrus output into the stream.
//
// Fire runs on whatever goroutine just logged, which makes re-entrancy the
// hazard that shapes this whole type: if it took a lock the fan-out also
// takes, one warning emitted while broadcasting would deadlock the server. So
// Fire does exactly one thing — a non-blocking channel send — and never logs.
type logHook struct {
	queue chan LogLine
}

// consoleTimeStamp is the clock-only stamp shown in the console; fullTimestamp
// keeps the date for hover tooltips.
const (
	consoleTimeStamp = "15:04:05.000"
	fullTimestamp    = "2006-01-02 15:04:05.000"
)

func (h *logHook) Levels() []logrus.Level { return logrus.AllLevels }

func (h *logHook) Fire(entry *logrus.Entry) error {
	line := LogLine{
		Time:      entry.Time.Format(consoleTimeStamp),
		Timestamp: entry.Time.Format(fullTimestamp),
		Level:     entry.Level.String(),
		Message:   entry.Message,
	}
	if len(entry.Data) > 0 {
		// The entry is reused after Fire returns, so its fields are copied
		// rather than referenced.
		line.Fields = make(map[string]string, len(entry.Data))
		for key, value := range entry.Data {
			line.Fields[key] = fmt.Sprint(value)
		}
	}

	select {
	case h.queue <- line:
	default:
		// Dropped. Saying so here would recurse straight back into Fire.
	}
	return nil
}

// startLogStream drains the hook queue into the ring buffer and out to every
// connected console.
//
// It stops on the stop channel rather than on a closed queue, and the queue is
// never closed, because a logrus hook cannot be uninstalled: packages keep
// logging all the way through shutdown, and every one of those lines would be
// a send on a closed channel. Once this returns, the hook's non-blocking sends
// simply fill the buffer and start dropping.
func (s *Service) startLogStream() {
	defer s.wg.Done()
	for {
		select {
		case <-s.stop:
			return
		case line := <-s.logQueue:
			stored := s.logs.append(line)
			// Quiet on purpose: a warning raised while broadcasting a log
			// line would be a log line, and round it goes.
			s.hub.fanout(FrameLog, stored, true)
		}
	}
}

// AttachLogrus installs the hook on the standard logger. It is separate from
// construction because logrus is a process-wide singleton: tests build a
// service without hijacking everyone else's logging.
func (s *Service) AttachLogrus() {
	s.attachOnce.Do(func() {
		logrus.AddHook(&logHook{queue: s.logQueue})
		logrus.Infof("后台日志流已接入，保留最近 %d 行", logRingSize)
	})
}

// LogSnapshot returns recent scrollback, oldest first.
func (s *Service) LogSnapshot(limit int) []LogLine {
	if limit <= 0 || limit > logSnapshotLimit {
		limit = logSnapshotLimit
	}
	return s.logs.snapshot(limit)
}

// LogCursor reports the newest log sequence number.
func (s *Service) LogCursor() int64 { return s.logs.cursor() }

// publishLogLine feeds a line in directly, bypassing logrus. Tests use it; so
// could any future caller that wants something on the console without putting
// it in the server's own log.
func (s *Service) publishLogLine(level, message string) {
	now := time.Now()
	select {
	case s.logQueue <- LogLine{
		Time:      now.Format(consoleTimeStamp),
		Timestamp: now.Format(fullTimestamp),
		Level:     level,
		Message:   message,
	}:
	default:
	}
}
