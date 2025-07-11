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

// Dispatcher 任务调度器
type Dispatcher struct {
	// 任务处理函数,类型到处理器的映射
	taskHandlers map[string]Handler
	// 任务队列
	queue chan Task
	// 最大工作goroutine数
	maxWorkers int
	// 优雅关闭
	wg *sync.WaitGroup
	// 关闭信号
	quit chan struct{}
}

// NewDispatcher 创建一个新的任务调度器
func NewDispatcher(maxWorkers int, queueSize int) *Dispatcher {
	return &Dispatcher{
		taskHandlers: make(map[string]Handler),
		queue:        make(chan Task, queueSize),
		maxWorkers:   maxWorkers,
		wg:           &sync.WaitGroup{},
		quit:         make(chan struct{}),
	}
}

// Register 注册任务处理函数
func (d *Dispatcher) Register(taskType string, handler Handler) {
	d.taskHandlers[taskType] = handler
}

// Submit 用于提交任务
func (d *Dispatcher) Submit(task Task) {
	d.queue <- task
}

// Start 任务队列，启动！
func (d *Dispatcher) Start() {
	for i := 0; i < d.maxWorkers; i++ {
		d.wg.Add(1)
		go d.worker(i + 1)
	}
	logrus.Infof("启动了 %d 个工作协程", d.maxWorkers)
}

// worker 实际运行任务的协程
func (d *Dispatcher) worker(workID int) {
	defer d.wg.Done()

	logrus.Infof("启动工作协程 %d", workID)

	for {
		select {
		case task := <-d.queue:
			handler, ok := d.taskHandlers[task.Type()]
			if !ok {
				logrus.Errorf("任务类型 %s 没有对应的处理函数", task.Type())
				continue
			}

			logrus.Infof("WorkerID %d 处理任务 %s", workID, task.Type())
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			err := handler(ctx, task.Payload())
			cancel()

			if err != nil {
				logrus.Errorf("处理任务 %s 失败: %v", task.Type(), err)
				// TODO 重试逻辑
			} else {
				logrus.Infof("任务 %s 处理完成", task.Type())
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
	close(d.quit)
	d.wg.Wait()
	close(d.queue)
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
