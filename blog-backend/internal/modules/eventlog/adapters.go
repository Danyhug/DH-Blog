package eventlog

import (
	"errors"
	"fmt"
	"strings"
)

// taskKindLabels turn the queue's internal type strings into what the admin
// page reads. The mapping lives here rather than in internal/task so that
// package keeps knowing nothing about presentation.
var taskKindLabels = map[string]string{
	"AI_Gen_Tags":    "AI 标签生成",
	"AI_Gen_Summary": "AI 摘要生成",
}

func taskLabel(taskType string) string {
	if label, ok := taskKindLabels[taskType]; ok {
		return label
	}
	return taskType
}

// taskSubject names the work in a way that reads on its own line, e.g.
// "文章 #12 的 AI 摘要生成".
func taskSubject(taskType string, targetID int) string {
	label := taskLabel(taskType)
	if targetID > 0 {
		return fmt.Sprintf("文章 #%d 的 %s", targetID, label)
	}
	return label
}

// TaskObserver adapts the service to the task package's Observer port. It is
// what finally gives the AI tag and summary jobs a voice: until now a job that
// exhausted its ten retries only wrote a line to the server log.
type TaskObserver struct{ service *Service }

func (o *TaskObserver) TaskQueued(taskType string, targetID int) {
	o.service.Publish(Event{
		Source:   SourceTask,
		Kind:     taskType,
		Status:   StatusQueued,
		TargetID: targetID,
		Title:    taskSubject(taskType, targetID) + " 已入队",
	})
}

func (o *TaskObserver) TaskSucceeded(taskType string, targetID, attempt int) {
	title := taskSubject(taskType, targetID) + " 完成"
	if attempt > 0 {
		title = fmt.Sprintf("%s（第 %d 次重试后成功）", title, attempt)
	}
	o.service.Publish(Event{
		Source:   SourceTask,
		Kind:     taskType,
		Status:   StatusSuccess,
		TargetID: targetID,
		Attempt:  attempt,
		Title:    title,
	})
}

func (o *TaskObserver) TaskRetrying(taskType string, targetID, attempt int, err error) {
	o.service.Publish(Event{
		Source:   SourceTask,
		Kind:     taskType,
		Status:   StatusRetrying,
		TargetID: targetID,
		Attempt:  attempt,
		Title:    fmt.Sprintf("%s 失败，即将进行第 %d 次重试", taskSubject(taskType, targetID), attempt),
		Detail:   errorDetail(err),
	})
}

func (o *TaskObserver) TaskFailed(taskType string, targetID, attempt int, err error) {
	title := taskSubject(taskType, targetID) + " 最终失败"
	if attempt > 0 {
		title = fmt.Sprintf("%s（已重试 %d 次）", title, attempt)
	}
	o.service.Publish(Event{
		Source:   SourceTask,
		Kind:     taskType,
		Status:   StatusFailed,
		TargetID: targetID,
		Attempt:  attempt,
		Title:    title,
		Detail:   errorDetail(err),
	})
}

// SyncReporter adapts the service to the files module's reporter port. A
// WebDAV write triggers a debounced rebuild that truncates the whole file
// table before rescanning, which is worth announcing to anyone who has the
// drive page open.
type SyncReporter struct{ service *Service }

const kindDiskSync = "disk_sync"

func (r *SyncReporter) SyncStarted() {
	r.service.Publish(Event{
		Source: SourceWebDAV,
		Kind:   kindDiskSync,
		Status: StatusRunning,
		Title:  "开始重建网盘文件索引",
		Detail: "WebDAV 写入后触发，将清空文件表并重新扫描磁盘",
	})
}

func (r *SyncReporter) SyncFinished(err error) {
	if err != nil {
		r.service.Publish(Event{
			Source: SourceWebDAV,
			Kind:   kindDiskSync,
			Status: StatusFailed,
			Title:  "网盘文件索引重建失败",
			Detail: errorDetail(err),
		})
		return
	}
	r.service.Publish(Event{
		Source: SourceWebDAV,
		Kind:   kindDiskSync,
		Status: StatusSuccess,
		Title:  "网盘文件索引重建完成",
	})
}

// GatewayReporter adapts the service to the AI gateway's reporter port. The
// hourly usage sync can pull a credential out of rotation on its own; that
// silently shrinks the gateway's capacity, so it belongs in the feed.
type GatewayReporter struct{ service *Service }

const kindUsageSync = "usage_sync"

func (r *GatewayReporter) UsageSyncFinished(failed int, parked, revived []string) {
	for _, key := range parked {
		r.service.Publish(Event{
			Source: SourceGateway,
			Kind:   kindUsageSync,
			Status: StatusFailed,
			Title:  fmt.Sprintf("网关密钥 %s 已自动停用", key),
			Detail: "上游用量同步判定该密钥已用尽或被拒绝，已移出轮换",
		})
	}
	for _, key := range revived {
		r.service.Publish(Event{
			Source: SourceGateway,
			Kind:   kindUsageSync,
			Status: StatusSuccess,
			Title:  fmt.Sprintf("网关密钥 %s 已恢复轮换", key),
			Detail: "上游用量同步显示配额已重置",
		})
	}
	if failed > 0 {
		r.service.Publish(Event{
			Source: SourceGateway,
			Kind:   kindUsageSync,
			Status: StatusFailed,
			Title:  fmt.Sprintf("上游用量同步有 %d 个密钥刷新失败", failed),
			Detail: "详见 AI 网关页面的密钥列表",
		})
	}
}

// errorDetail keeps a stored error short enough to render in a table cell.
func errorDetail(err error) string {
	if err == nil {
		return ""
	}
	detail := strings.TrimSpace(err.Error())
	const maxDetail = 500
	if runes := []rune(detail); len(runes) > maxDetail {
		return string(runes[:maxDetail]) + "…"
	}
	return detail
}

// ContentReporter adapts the service to the agent module's reporter port.
// Agent write actions have the same "nobody was watching" character as the
// rest of the feed, and a denied edit is the only signal that an agent tried
// to touch something it should not have.
type ContentReporter struct{ service *Service }

const (
	kindAgentWrite = "agent_write"
	kindEditGrant  = "edit_grant"
)

func (r *ContentReporter) ArticleCreated(agent, title string, articleID int) {
	r.service.Publish(Event{
		Source: SourceArticle, Kind: kindAgentWrite, Status: StatusSuccess,
		TargetID: articleID,
		Title:    fmt.Sprintf("%s 发布了《%s》", agent, title),
	})
}

// ArticleUpdated distinguishes the two flavours in the title, because a grant
// bypass is who-authorized-this information, and titles are frozen at publish
// time so the distinction survives later rewording of the code.
func (r *ContentReporter) ArticleUpdated(agent, title string, articleID int, viaGrant bool) {
	if viaGrant {
		title = fmt.Sprintf("%s 持临时授权修改了《%s》", agent, title)
	} else {
		title = fmt.Sprintf("%s 修改了《%s》", agent, title)
	}
	r.service.Publish(Event{
		Source: SourceArticle, Kind: kindAgentWrite, Status: StatusSuccess,
		TargetID: articleID,
		Title:    title,
	})
}

func (r *ContentReporter) ArticleUpdateDenied(agent, title string, articleID int, reason string) {
	r.service.Publish(Event{
		Source: SourceArticle, Kind: kindAgentWrite, Status: StatusFailed,
		TargetID: articleID,
		Title:    fmt.Sprintf("%s 尝试修改《%s》被拒绝", agent, title),
		Detail:   errorDetail(errors.New(reason)),
	})
}

func (r *ContentReporter) GrantIssued(articleID int, note string) {
	r.service.Publish(Event{
		Source: SourceArticle, Kind: kindEditGrant, Status: StatusRunning,
		TargetID: articleID,
		Title:    "已签发 AI 修改授权，1 小时后失效",
		Detail:   strings.TrimSpace(note),
	})
}
