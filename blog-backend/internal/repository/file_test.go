package repository

import (
	"context"
	"dh-blog/internal/model"
	"testing"

	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// setupTestDB 设置测试数据库
// 创建一个内存 SQLite 数据库并迁移文件表结构
// 参数:
//   - t: 测试对象
//
// 返回:
//   - *gorm.DB: 数据库连接
func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("打开 SQLite 数据库失败: %v", err)
	}

	// 创建文件表
	err = db.AutoMigrate(&model.File{})
	if err != nil {
		t.Fatalf("迁移 File 模型失败: %v", err)
	}

	return db
}

// TestFileRepository_CRUD 测试文件仓库的基本CRUD操作
// 测试创建、读取、更新和删除文件记录的功能
func TestFileRepository_CRUD(t *testing.T) {
	db := setupTestDB(t)
	repo := NewFileRepository(db)
	ctx := context.Background()

	// 创建测试文件对象
	file := &model.File{
		UserID:   1,
		ParentID: "",
		Name:     "test.txt",
		IsFolder: false,
		Size:     100,
	}

	// 测试创建操作
	err := repo.Create(ctx, file)
	assert.NoError(t, err, "创建文件记录应该成功")
	assert.NotEqual(t, uint(0), file.ID, "创建后ID应该已生成")

	// 测试查找操作
	found, err := repo.FindByID(ctx, uint(file.ID))
	assert.NoError(t, err, "查找文件记录应该成功")
	assert.Equal(t, file.Name, found.Name, "文件名应该匹配")
	assert.Equal(t, file.UserID, found.UserID, "用户ID应该匹配")

	// 测试更新操作
	file.Name = "updated.txt"
	err = repo.Update(ctx, file)
	assert.NoError(t, err, "更新文件记录应该成功")

	// 验证更新结果
	updated, err := repo.FindByID(ctx, uint(file.ID))
	assert.NoError(t, err, "查找更新后的文件记录应该成功")
	assert.Equal(t, "updated.txt", updated.Name, "文件名应该已更新")

	// 测试删除操作
	err = repo.Delete(ctx, uint(file.ID))
	assert.NoError(t, err, "删除文件记录应该成功")

	// 验证删除结果
	_, err = repo.FindByID(ctx, uint(file.ID))
	assert.Error(t, err, "删除后应该无法找到文件记录")
}

// TestFileRepository_ListByParentID 测试按父目录ID列出文件
// 测试列出指定目录下所有文件的功能
func TestFileRepository_ListByParentID(t *testing.T) {
	db := setupTestDB(t)
	repo := NewFileRepository(db)
	ctx := context.Background()

	// 创建一个根目录文件夹
	rootFolder := &model.File{
		UserID:   1,
		ParentID: "",
		Name:     "root",
		IsFolder: true,
		Size:     0,
	}
	err := repo.Create(ctx, rootFolder)
	assert.NoError(t, err, "创建根文件夹应该成功")

	// 在根目录中创建3个测试文件
	for i := 1; i <= 3; i++ {
		file := &model.File{
			UserID:   1,
			ParentID: "root",
			Name:     "file-in-root.txt",
			IsFolder: false,
			Size:     100,
		}
		err := repo.Create(ctx, file)
		assert.NoError(t, err, "创建测试文件应该成功")
	}

	// 测试列出根目录下的所有文件
	files, err := repo.ListByParentID(ctx, 1, "root")
	assert.NoError(t, err, "列出文件应该成功")
	assert.Len(t, files, 3, "根目录下应该有3个文件")

	// 测试列出不存在目录下的文件
	files, err = repo.ListByParentID(ctx, 1, "non-existent")
	assert.NoError(t, err, "列出不存在目录的文件应该成功，但返回空列表")
	assert.Len(t, files, 0, "不存在的目录下应该没有文件")
}

// TestFileRepository_FindByUserIDAndName 测试按用户ID、父目录和文件名查找文件
// 测试根据用户ID、父目录和文件名精确定位文件的功能
func TestFileRepository_FindByUserIDAndName(t *testing.T) {
	db := setupTestDB(t)
	repo := NewFileRepository(db)
	ctx := context.Background()

	// 创建一个测试文件
	file := &model.File{
		UserID:   1,
		ParentID: "parent",
		Name:     "unique-name.txt",
		IsFolder: false,
		Size:     100,
	}
	err := repo.Create(ctx, file)
	assert.NoError(t, err, "创建测试文件应该成功")

	// 测试按用户ID、父目录和文件名查找文件
	found, err := repo.FindByUserIDAndName(ctx, 1, "parent", "unique-name.txt")
	assert.NoError(t, err, "查找存在的文件应该成功")
	assert.Equal(t, file.ID, found.ID, "找到的文件ID应该匹配")

	// 测试查找不存在的文件
	_, err = repo.FindByUserIDAndName(ctx, 1, "parent", "not-exists.txt")
	assert.Error(t, err, "查找不存在的文件应该返回错误")
}
