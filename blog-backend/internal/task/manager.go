package task

import (
	"dh-blog/internal/repository"
	"dh-blog/internal/service"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

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

// GetDispatcher 获取任务调度器
func (m *TaskManager) GetDispatcher() *Dispatcher {
	return m.dispatcher
}

// SubmitTask 提交任务
func (m *TaskManager) SubmitTask(task Task) {
	m.dispatcher.Submit(task)
	logrus.Infof("任务 %s 已提交", task.Type())
}
