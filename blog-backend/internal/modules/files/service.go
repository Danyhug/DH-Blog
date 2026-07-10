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

	// SyncFilesFromDisk 防抖相关
	syncMu     sync.Mutex
	syncTimer  *time.Timer
	syncExecMu sync.Mutex // 保护实际同步操作，防止并发执行
}

var protectedDirectories = [...]string{"博客"}

// Service 定义文件模块对外开放的业务能力。
type Service interface {
	// UploadFile 保存其他模块提交的文件内容。
	UploadFile(ctx context.Context, userID uint64, parentID string, fileName string, fileSize int64, fileContent io.Reader) (*File, error)

	// GetDownloadInfo 返回经过访问校验的下载元信息。
	GetDownloadInfo(ctx context.Context, userID uint64, fileID string) (*File, error)

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
