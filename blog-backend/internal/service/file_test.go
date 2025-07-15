package service

import (
	"bytes"
	"context"
	"dh-blog/internal/model"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gorm.io/gorm"
)

// 创建一个模拟的文件仓库
type mockFileRepository struct {
	mock.Mock
}

func (m *mockFileRepository) Create(ctx context.Context, file *model.File) error {
	args := m.Called(ctx, file)
	return args.Error(0)
}

func (m *mockFileRepository) Update(ctx context.Context, file *model.File) error {
	args := m.Called(ctx, file)
	return args.Error(0)
}

func (m *mockFileRepository) Delete(ctx context.Context, id uint) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *mockFileRepository) FindByID(ctx context.Context, id uint) (*model.File, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.File), args.Error(1)
}

func (m *mockFileRepository) ListByParentID(ctx context.Context, userID uint64, parentID string) ([]*model.File, error) {
	args := m.Called(ctx, userID, parentID)
	return args.Get(0).([]*model.File), args.Error(1)
}

func (m *mockFileRepository) FindByPath(ctx context.Context, userID uint64, path string) (*model.File, error) {
	args := m.Called(ctx, userID, path)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.File), args.Error(1)
}

func (m *mockFileRepository) FindByUserIDAndName(ctx context.Context, userID uint64, parentID string, name string) (*model.File, error) {
	args := m.Called(ctx, userID, parentID, name)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.File), args.Error(1)
}

func (m *mockFileRepository) BatchDelete(ctx context.Context, ids []uint) error {
	args := m.Called(ctx, ids)
	return args.Error(0)
}

func (m *mockFileRepository) CountByUserID(ctx context.Context, userID uint64) (int64, error) {
	args := m.Called(ctx, userID)
	return args.Get(0).(int64), args.Error(1)
}

func (m *mockFileRepository) CountByUserIDAndParentID(ctx context.Context, userID uint64, parentID string) (int64, error) {
	args := m.Called(ctx, userID, parentID)
	return args.Get(0).(int64), args.Error(1)
}

func TestFileService_ListFiles(t *testing.T) {
	// 创建模拟仓库
	mockRepo := new(mockFileRepository)

	// 设置测试路径
	testDir := t.TempDir()

	// 创建文件服务实例
	fileService := NewFileService(mockRepo, testDir)

	// 设置模拟行为
	mockFiles := []*model.File{
		{ID: 1, UserID: 1, ParentID: "", Name: "file1.txt", IsFolder: false, Size: 100},
		{ID: 2, UserID: 1, ParentID: "", Name: "folder1", IsFolder: true, Size: 0},
	}

	mockRepo.On("ListByParentID", mock.Anything, uint64(1), "").Return(mockFiles, nil)

	// 测试
	ctx := context.Background()
	files, err := fileService.ListFiles(ctx, 1, "")

	// 断言
	assert.NoError(t, err, "列出文件应该成功")
	assert.Equal(t, 2, len(files), "应该返回两个文件")
	assert.Equal(t, "file1.txt", files[0].Name, "文件名应匹配")

	// 验证所有期望都已满足
	mockRepo.AssertExpectations(t)
}

func TestFileService_CreateFolder(t *testing.T) {
	// 创建模拟仓库
	mockRepo := new(mockFileRepository)

	// 设置测试路径
	testDir := t.TempDir()

	// 创建文件服务实例
	fileService := NewFileService(mockRepo, testDir)

	// 设置模拟行为 - 确保文件夹不存在
	mockRepo.On("FindByUserIDAndName", mock.Anything, uint64(1), "parent", "newfolder").Return(nil, gorm.ErrRecordNotFound)

	// 设置创建文件夹的期望
	mockRepo.On("Create", mock.Anything, mock.AnythingOfType("*model.File")).Return(nil).Run(func(args mock.Arguments) {
		file := args.Get(1).(*model.File)
		file.ID = 3 // 模拟数据库自动生成ID
	})

	// 测试
	ctx := context.Background()
	folder, err := fileService.CreateFolder(ctx, 1, "parent", "newfolder")

	// 断言
	assert.NoError(t, err, "创建文件夹应该成功")
	assert.NotNil(t, folder, "应该返回文件夹信息")
	assert.Equal(t, "newfolder", folder.Name, "文件夹名应匹配")
	assert.Equal(t, uint64(3), folder.ID, "应该有正确的ID")

	// 验证所有期望都已满足
	mockRepo.AssertExpectations(t)

	// 验证物理文件夹已创建
	expectedPath := filepath.Join(testDir, "user_1", "parent", "newfolder")
	_, err = os.Stat(expectedPath)
	assert.NoError(t, err, "物理文件夹应该已创建")
}

func TestFileService_UploadFile(t *testing.T) {
	// 创建模拟仓库
	mockRepo := new(mockFileRepository)

	// 设置测试路径
	testDir := t.TempDir()

	// 创建文件服务实例
	fileService := NewFileService(mockRepo, testDir)

	// 设置模拟行为 - 确保文件不存在
	mockRepo.On("FindByUserIDAndName", mock.Anything, uint64(1), "parent", "testfile.txt").Return(nil, gorm.ErrRecordNotFound)

	// 设置创建文件的期望
	mockRepo.On("Create", mock.Anything, mock.AnythingOfType("*model.File")).Return(nil).Run(func(args mock.Arguments) {
		file := args.Get(1).(*model.File)
		file.ID = 4 // 模拟数据库自动生成ID
	})

	// 测试文件内容
	fileContent := []byte("这是测试文件内容")

	// 测试
	ctx := context.Background()
	file, err := fileService.UploadFile(ctx, 1, "parent", "testfile.txt", int64(len(fileContent)), bytes.NewReader(fileContent))

	// 断言
	assert.NoError(t, err, "上传文件应该成功")
	assert.NotNil(t, file, "应该返回文件信息")
	assert.Equal(t, "testfile.txt", file.Name, "文件名应匹配")
	assert.Equal(t, int64(len(fileContent)), file.Size, "文件大小应匹配")

	// 验证所有期望都已满足
	mockRepo.AssertExpectations(t)

	// 验证物理文件已创建且内容正确
	expectedPath := filepath.Join(testDir, "user_1", "parent", "testfile.txt")
	content, err := os.ReadFile(expectedPath)
	assert.NoError(t, err, "应该能够读取物理文件")
	assert.Equal(t, fileContent, content, "文件内容应该正确")
}

func TestFileService_GetDownloadInfo(t *testing.T) {
	// 创建模拟仓库
	mockRepo := new(mockFileRepository)

	// 设置测试路径
	testDir := t.TempDir()

	// 创建一个测试文件
	testFilePath := filepath.Join(testDir, "user_1", "test.txt")
	os.MkdirAll(filepath.Dir(testFilePath), os.ModePerm)
	err := os.WriteFile(testFilePath, []byte("测试内容"), 0644)
	assert.NoError(t, err, "创建测试文件应成功")

	// 创建文件服务实例
	fileService := NewFileService(mockRepo, testDir)

	// 设置模拟行为
	mockFile := &model.File{
		ID:          5,
		UserID:      1,
		ParentID:    "",
		Name:        "test.txt",
		IsFolder:    false,
		Size:        11, // "测试内容" 的字节大小
		StoragePath: "user_1/test.txt",
		MimeType:    "text/plain",
	}

	// 设置FindByID模拟行为
	mockRepo.On("FindByID", mock.Anything, uint(5)).Return(mockFile, nil)

	// 测试
	ctx := context.Background()
	fileInfo, err := fileService.GetDownloadInfo(ctx, 1, "5")

	// 断言
	assert.NoError(t, err, "获取下载信息应该成功")
	assert.NotNil(t, fileInfo, "应该返回文件信息")
	assert.Equal(t, "test.txt", fileInfo.Name, "文件名应匹配")
	assert.Equal(t, testFilePath, fileInfo.StoragePath, "应返回完整的物理文件路径")

	// 验证所有期望都已满足
	mockRepo.AssertExpectations(t)
}

// 生成随机字符串，避免测试冲突
func randomString(n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[i%len(letters)]
	}
	return string(b)
}
