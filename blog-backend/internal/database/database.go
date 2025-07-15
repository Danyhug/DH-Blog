package database

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"dh-blog/internal/config"
	"dh-blog/internal/model"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Init 初始化数据库连接并执行自动迁移
func Init(conf *config.Config) (*gorm.DB, error) {
	newLogger := logger.New(
		log.New(os.Stdout, "\r\n", log.LstdFlags), // io writer
		logger.Config{
			SlowThreshold:             time.Second, // Slow SQL threshold
			LogLevel:                  logger.Info, // Log level
			IgnoreRecordNotFoundError: true,        // Ignore ErrRecordNotFound error for logger
			Colorful:                  true,        // Disable color
		},
	)

	exePath, err := os.Executable()
	if err != nil {
		return nil, err
	}
	dbPath := filepath.Join(filepath.Dir(exePath), conf.DataBase.DBFile)
	fmt.Printf("可执行文件路径: %s\n", exePath)
	fmt.Printf("数据库文件路径: %s\n", dbPath)

	// 初始化数据库连接
	db, err := gorm.Open(sqlite.Open(dbPath+"?_pragma=journal_mode(WAL)&_pragma=busy_timeout(10000)"), &gorm.Config{
		Logger: newLogger,
	})
	// 使用 SQLite 驱动并从配置中读取数据库文件路径
	if err != nil {
		return nil, fmt.Errorf("连接数据库失败: %w", err)
	}

	// 处理IP黑名单表的迁移问题
	// 先删除表（如果存在），然后重新创建
	if db.Migrator().HasTable(&model.IPBlacklist{}) {
		if err := db.Migrator().DropTable(&model.IPBlacklist{}); err != nil {
			return nil, fmt.Errorf("删除IP黑名单表失败: %w", err)
		}
	}

	// 自动迁移模型到数据库表
	// GORM 会根据 model 包中定义的结构体自动创建或更新数据库表
	err = db.AutoMigrate(
		&model.AccessLog{},
		&model.Article{},
		&model.Category{},
		&model.Comment{},
		&model.Tag{},
		&model.User{},
		&model.SystemSetting{},
		&model.IPBlacklist{},
		&model.TagRelation{},
		&model.File{},
	)
	if err != nil {
		return nil, fmt.Errorf("数据库自动迁移失败: %w", err)
	}

	// 插入默认数据
	if err := insertDefaultData(db); err != nil {
		return nil, fmt.Errorf("插入默认数据失败: %w", err)
	}

	return db, nil
}
