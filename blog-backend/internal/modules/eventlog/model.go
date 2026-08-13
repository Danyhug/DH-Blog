package eventlog

import (
	"dh-blog/internal/model"
)

// Event statuses. A background job normally walks queued → success, or
// queued → retrying* → failed. Sources that do not queue (a WebDAV sync, a
// gateway key being parked) publish running/success/failed directly.
const (
	StatusQueued   = "queued"
	StatusRunning  = "running"
	StatusSuccess  = "success"
	StatusRetrying = "retrying"
	StatusFailed   = "failed"
)

// Event sources. These name the subsystem the work belongs to, not the code
// that published it, so the admin page can group by something meaningful.
const (
	SourceTask    = "task"
	SourceArticle = "article"
	SourceWebDAV  = "webdav"
	SourceGateway = "gateway"
)

// Event is one thing that happened in the background where nobody was
// watching. It is append-only and deliberately denormalised: the admin page
// reads it as a flat feed, and a job's target row may be gone by the time
// anyone looks.
type Event struct {
	ID        int64          `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	CreatedAt model.JSONTime `gorm:"column:created_at;index" json:"createdAt"`
	Source    string         `gorm:"column:source;index" json:"source"`
	// Kind is the job identity within a source, e.g. the task queue's type
	// string ("AI_Gen_Tags") or "disk_sync".
	Kind   string `gorm:"column:kind;index" json:"kind"`
	Status string `gorm:"column:status;index" json:"status"`
	// TargetID links back to the business record (an article id, ...). Zero
	// means the job is not about a single row.
	TargetID int `gorm:"column:target_id" json:"targetId"`
	// Title is the one line the admin page shows. It is written at publish
	// time rather than derived on read, so the feed stays readable even after
	// the wording in the code changes.
	Title string `gorm:"column:title" json:"title"`
	// Detail carries the error text or extra context. Empty on success.
	Detail string `gorm:"column:detail" json:"detail"`
	// Attempt is the 1-based try number for retried work, 0 when not retried.
	Attempt int `gorm:"column:attempt" json:"attempt"`
}

func (Event) TableName() string { return "task_events" }

// Failed reports whether this event represents work that did not complete.
func (e *Event) Failed() bool { return e.Status == StatusFailed }
