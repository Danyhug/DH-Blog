package task

import (
	"context"
	"sync"
	"time"

	"dh-blog/internal/repository"
	"dh-blog/internal/service"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

const (
	// MaxRetries 最大重试次数
	MaxRetries = 10
	// RetryInterval 重试间隔（秒）
	RetryInterval = 5
)

// RetryableTask 包装一个任务，添加重试信息
type RetryableTask struct {
	Task         Task
	RetryCount   int
	LastAttempt  time.Time
	NextAttempt  time.Time
	IsRetry      bool
	OriginalTime time.Time // 原始任务提交时间
}

// Dispatcher 任务调度器
type Dispatcher struct {
	// 任务处理函数,类型到处理器的映射
	taskHandlers map[string]Handler
	// 任务队列
	queue chan *RetryableTask
	// 最大工作goroutine数
	maxWorkers int
	// 优雅关闭
	wg *sync.WaitGroup
	// 关闭信号
	quit chan struct{}
	// 重试队列
	retryQueue chan *RetryableTask
	// 是否正在关闭
	isShutdown bool
	// 互斥锁，保护isShutdown
	mu sync.RWMutex
}

// NewDispatcher 创建一个新的任务调度器
func NewDispatcher(maxWorkers int, queueSize int) *Dispatcher {
	return &Dispatcher{
		taskHandlers: make(map[string]Handler),
		queue:        make(chan *RetryableTask, queueSize),
		retryQueue:   make(chan *RetryableTask, queueSize),
		maxWorkers:   maxWorkers,
		wg:           &sync.WaitGroup{},
		quit:         make(chan struct{}),
		isShutdown:   false,
	}
}

// Register 注册任务处理函数
func (d *Dispatcher) Register(taskType string, handler Handler) {
	d.taskHandlers[taskType] = handler
}

// Submit 用于提交任务
func (d *Dispatcher) Submit(task Task) {
	// 检查是否正在关闭
	d.mu.RLock()
	if d.isShutdown {
		d.mu.RUnlock()
		logrus.Warnf("任务队列正在关闭，拒绝新任务: %s", task.Type())
		return
	}
	d.mu.RUnlock()

	// 包装成可重试任务
	retryableTask := &RetryableTask{
		Task:         task,
		RetryCount:   0,
		LastAttempt:  time.Time{}, // 零值，表示从未尝试
		NextAttempt:  time.Time{}, // 零值，表示立即执行
		IsRetry:      false,
		OriginalTime: time.Now(),
	}

	d.queue <- retryableTask
	logrus.Debugf("提交任务: %s", task.Type())
}

// submitRetry 提交重试任务
func (d *Dispatcher) submitRetry(task *RetryableTask) {
	// 检查是否正在关闭
	d.mu.RLock()
	if d.isShutdown {
		d.mu.RUnlock()
		logrus.Warnf("任务队列正在关闭，拒绝重试任务: %s", task.Task.Type())
		return
	}
	d.mu.RUnlock()

	// 更新重试信息
	task.RetryCount++
	task.LastAttempt = time.Now()
	task.NextAttempt = time.Now().Add(RetryInterval * time.Second)
	task.IsRetry = true

	// 放入重试队列
	d.retryQueue <- task
	logrus.Infof("任务 %s 将在 %v 后进行第 %d 次重试",
		task.Task.Type(), RetryInterval*time.Second, task.RetryCount)
}

// Start 任务队列，启动！
func (d *Dispatcher) Start() {
	// 启动工作协程
	for i := 0; i < d.maxWorkers; i++ {
		d.wg.Add(1)
		go d.worker(i + 1)
	}
	logrus.Infof("启动了 %d 个工作协程", d.maxWorkers)

	// 启动重试管理协程
	d.wg.Add(1)
	go d.retryManager()
}

// retryManager 管理重试任务的协程
func (d *Dispatcher) retryManager() {
	defer d.wg.Done()

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	pendingRetries := make([]*RetryableTask, 0)

	for {
		select {
		case <-d.quit:
			logrus.Info("重试管理器停止")
			return

		case task := <-d.retryQueue:
			// 添加到待重试列表
			pendingRetries = append(pendingRetries, task)

		case <-ticker.C:
			now := time.Now()
			remaining := make([]*RetryableTask, 0, len(pendingRetries))

			// 检查是否有任务需要重试
			for _, task := range pendingRetries {
				if now.After(task.NextAttempt) {
					// 时间到了，重新提交到主队列
					d.queue <- task
					logrus.Infof("重新提交任务 %s 进行第 %d 次重试",
						task.Task.Type(), task.RetryCount)
				} else {
					// 还没到时间，保留在待重试列表
					remaining = append(remaining, task)
				}
			}

			// 更新待重试列表
			pendingRetries = remaining
		}
	}
}

// worker 实际运行任务的协程
func (d *Dispatcher) worker(workID int) {
	defer d.wg.Done()

	logrus.Infof("启动工作协程 %d", workID)

	for {
		select {
		case retryableTask := <-d.queue:
			task := retryableTask.Task
			handler, ok := d.taskHandlers[task.Type()]
			if !ok {
				logrus.Errorf("任务类型 %s 没有对应的处理函数", task.Type())
				continue
			}

			// 记录日志，区分是否为重试
			if retryableTask.IsRetry {
				logrus.Infof("WorkerID %d 处理重试任务 %s (第 %d 次尝试)",
					workID, task.Type(), retryableTask.RetryCount)
			} else {
				logrus.Infof("WorkerID %d 处理任务 %s", workID, task.Type())
			}

			// 执行任务
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			err := handler(ctx, task.Payload())
			cancel()

			if err != nil {
				// 处理失败，判断是否需要重试
				if retryableTask.RetryCount < MaxRetries {
					logrus.Warnf("处理任务 %s 失败: %v, 将进行重试 (%d/%d)",
						task.Type(), err, retryableTask.RetryCount+1, MaxRetries)
					d.submitRetry(retryableTask)
				} else {
					// 超过最大重试次数
					logrus.Errorf("任务 %s 失败，已达到最大重试次数 %d: %v",
						task.Type(), MaxRetries, err)

					// 这里可以添加失败后的处理逻辑，如通知管理员等
					elapsed := time.Since(retryableTask.OriginalTime)
					logrus.Errorf("任务 %s 最终失败，总耗时 %v", task.Type(), elapsed)
				}
			} else {
				// 任务成功
				if retryableTask.IsRetry {
					logrus.Infof("重试任务 %s 处理完成 (第 %d 次尝试成功)",
						task.Type(), retryableTask.RetryCount)
				} else {
					logrus.Infof("任务 %s 处理完成", task.Type())
				}
			}
		case <-d.quit:
			logrus.Infof("工作协程 %d 停止", workID)
			return
		}
	}
}

// Stop 关闭
func (d *Dispatcher) Stop() {
	logrus.Infof("正在关闭任务队列...")

	// 标记为正在关闭
	d.mu.Lock()
	d.isShutdown = true
	d.mu.Unlock()

	close(d.quit)
	d.wg.Wait()
	close(d.queue)
	close(d.retryQueue)
	logrus.Infof("任务队列已关闭")
}

// TaskManager 任务管理器，负责初始化和管理所有任务
type TaskManager struct {
	dispatcher *Dispatcher
}

// NewTaskManager 创建一个新的任务管理器
func NewTaskManager(
	db *gorm.DB,
	aiService service.AIService,
	tagRepo *repository.TagRepository,
) *TaskManager {
	// 创建任务调度器
	dispatcher := NewDispatcher(5, 100) // 5个工作协程，队列大小100

	// 注册AI相关任务
	RegisterAITaskHandlers(dispatcher, db, aiService, tagRepo)

	logrus.Info("任务管理器初始化完成")
	return &TaskManager{
		dispatcher: dispatcher,
	}
}

// Start 启动任务管理器
func (m *TaskManager) Start() {
	m.dispatcher.Start()
	logrus.Info("任务管理器已启动")
}

// Stop 停止任务管理器
func (m *TaskManager) Stop() {
	m.dispatcher.Stop()
	logrus.Info("任务管理器已停止")
}

// SubmitTask 提交任务
func (m *TaskManager) SubmitTask(task Task) {
	m.dispatcher.Submit(task)
	logrus.Infof("任务 %s 已提交", task.Type())
}
