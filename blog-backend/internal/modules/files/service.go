package files

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

type fileService struct {
	repo        fileRepository
	pathMu      sync.RWMutex
	filePath    string // 实际存储文件的基础路径
	chunkSizeKB int

	// events reports disk syncs to the admin feed; nil when nothing listens.
	events EventReporter

	// SyncFilesFromDisk 防抖相关
	syncMu     sync.Mutex
	syncTimer  *time.Timer
	syncExecMu sync.Mutex // 保护实际同步操作，防止并发执行

	// temp janitor 清理过期上传会话。stop/done 只在 Start 里创建，
	// started 由 tempJanitorMu 保护，避免 Stop 无锁读 Start 写入的字段。
	tempJanitorMu       sync.Mutex
	tempJanitorStarted  bool
	tempJanitorStop     chan struct{}
	tempJanitorDone     chan struct{}
	tempJanitorStopOnce sync.Once
}

// 过期上传会话的清理参数。complete 接口会删除成功会话，
// janitor 负责兜底清理被放弃的会话与服务重启残留。
const (
	tempSessionMaxAge   = 24 * time.Hour
	tempCleanupInterval = 6 * time.Hour
)

// StartTempJanitor 启动过期上传会话清理循环，可安全重复调用。
func (s *fileService) StartTempJanitor() {
	s.tempJanitorMu.Lock()
	defer s.tempJanitorMu.Unlock()
	if s.tempJanitorStarted {
		return
	}
	s.tempJanitorStarted = true
	s.tempJanitorStop = make(chan struct{})
	s.tempJanitorDone = make(chan struct{})
	go s.tempJanitorLoop()
}

// StopTempJanitor 停止清理循环。未启动时直接返回，重复调用安全。
func (s *fileService) StopTempJanitor() {
	s.tempJanitorMu.Lock()
	started := s.tempJanitorStarted
	stop := s.tempJanitorStop
	done := s.tempJanitorDone
	s.tempJanitorMu.Unlock()

	if !started {
		return
	}
	s.tempJanitorStopOnce.Do(func() { close(stop) })
	<-done
}

func (s *fileService) tempJanitorLoop() {
	defer close(s.tempJanitorDone)
	s.cleanupTempSessions()
	ticker := time.NewTicker(tempCleanupInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			s.cleanupTempSessions()
		case <-s.tempJanitorStop:
			return
		}
	}
}

// cleanupTempSessions 删除超过 tempSessionMaxAge 未改动的上传会话目录。
func (s *fileService) cleanupTempSessions() {
	baseDir := s.GetStoragePath()
	if baseDir == "" {
		return
	}
	tempRoot := filepath.Join(baseDir, tempDirName)
	entries, err := os.ReadDir(tempRoot)
	if err != nil {
		if !os.IsNotExist(err) {
			logrus.Warnf("读取上传临时目录失败: %v", err)
		}
		return
	}
	cutoff := time.Now().Add(-tempSessionMaxAge)
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		if info.ModTime().Before(cutoff) {
			if err := os.RemoveAll(filepath.Join(tempRoot, entry.Name())); err != nil {
				logrus.Warnf("清理过期上传会话失败: %s, 错误: %v", entry.Name(), err)
			} else {
				logrus.Infof("已清理过期上传会话: %s", entry.Name())
			}
		}
	}
}

var protectedDirectories = [...]string{"博客"}

// Service 定义文件模块对外开放的业务能力。
type Service interface {
	// UploadFile 保存其他模块提交的文件内容。
	UploadFile(ctx context.Context, userID uint64, parentID string, fileName string, fileSize int64, fileContent io.Reader) (*File, error)

	// GetDownloadInfo 返回经过访问校验的下载元信息。
	GetDownloadInfo(ctx context.Context, userID uint64, fileID string) (*File, error)

	// GetDownloadInfoForShare 返回经过磁盘存在性校验的下载元信息，不校验文件属主。
	// share 模块在令牌校验通过后调用：公开分享的受众并不是文件属主。
	GetDownloadInfoForShare(ctx context.Context, fileID string) (*File, error)

	// GetStoragePath 获取当前存储路径。
	GetStoragePath() string

	// EnsureProtectedDirectories 确保模块固定目录存在。
	EnsureProtectedDirectories(ctx context.Context) error

	// GetProtectedDirectoryID 获取固定目录的数据库 ID。
	GetProtectedDirectoryID(ctx context.Context, dirName string) (string, error)

	// SyncFilesFromDiskDebounced 请求一次防抖后的磁盘同步。
	SyncFilesFromDiskDebounced()
}

// newService 创建文件服务。
// 参数:
//   - repo: 文件存储库接口
//   - settingRepo: 系统设置仓库接口
//
// 返回:
//   - Service: 文件服务接口
func newService(repo fileRepository, initialPath string, initialChunkSizeKB int) *fileService {
	// 默认路径应该是 可执行文件/data/webdav
	executable, err := os.Executable()
	if err != nil {
		logrus.Errorf("无法获取可执行文件的路径: %v", err)
		return nil
	}
	defaultPath := filepath.Join(filepath.Dir(executable), "data", "webdav")

	if initialPath == "" {
		initialPath = defaultPath
	}
	if initialChunkSizeKB <= 0 {
		initialChunkSizeKB = 5120
	}
	service := &fileService{repo: repo, filePath: initialPath, chunkSizeKB: initialChunkSizeKB}
	if err := os.MkdirAll(initialPath, os.ModePerm); err != nil {
		logrus.Warnf("创建文件存储路径失败: %v", err)
	}

	return service
}
