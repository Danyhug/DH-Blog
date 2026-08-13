package task

// Observer is notified as a task moves through the queue. It exists because
// the queue is where background work goes to fail quietly: a handler that
// exhausts MaxRetries used to leave nothing behind but a log line.
//
// The methods are separate rather than one call with a status string so the
// contract is checked by the compiler, and the task package stays free of any
// vocabulary it would otherwise have to share with its observer.
type Observer interface {
	// TaskQueued fires once when a task is accepted into the queue.
	TaskQueued(taskType string, targetID int)
	// TaskSucceeded fires when a handler returns nil. attempt is 0 for work
	// that succeeded on its first run.
	TaskSucceeded(taskType string, targetID, attempt int)
	// TaskRetrying fires when a handler failed but retries remain; attempt is
	// the retry that is about to be scheduled.
	TaskRetrying(taskType string, targetID, attempt int, err error)
	// TaskFailed fires when a task is given up on for good.
	TaskFailed(taskType string, targetID, attempt int, err error)
}

// Targeted lets a task name the business record it acts on, so an observer can
// link its report back to that record. The task package treats it as an opaque
// id; tasks that do not act on a single row simply do not implement it.
type Targeted interface {
	Target() int
}

// targetOf reads a task's record id, or 0 when it does not have one.
func targetOf(task Task) int {
	if targeted, ok := task.(Targeted); ok {
		return targeted.Target()
	}
	return 0
}
