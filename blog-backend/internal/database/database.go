package database

import (
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"dh-blog/internal/config"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// ErrMigrationModelsRequired 表示初始化数据库时未提供权威迁移模型清单。
var ErrMigrationModelsRequired = errors.New("数据库迁移模型不能为空")

// Init 初始化数据库连接并执行自动迁移。
func Init(conf *config.Config, migrationModels ...any) (*gorm.DB, error) {
	if len(migrationModels) == 0 {
		return nil, ErrMigrationModelsRequired
	}

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
	dbPath := conf.DataBase.DBFile
	if !filepath.IsAbs(dbPath) {
		dbPath = filepath.Join(filepath.Dir(exePath), dbPath)
	}
	fmt.Printf("可执行文件路径: %s\n", exePath)
	fmt.Printf("数据库文件路径: %s\n", dbPath)

	// 初始化数据库连接
	db, err := gorm.Open(sqlite.Open(dbPath+"?_pragma=journal_mode(WAL)&_pragma=busy_timeout(10000)&_pragma=synchronous=NORMAL&_pragma=cache_size=10000&_pragma=temp_store=memory"), &gorm.Config{
		Logger: newLogger,
	})
	// 使用 SQLite 驱动并从配置中读取数据库文件路径
	if err != nil {
		return nil, fmt.Errorf("连接数据库失败: %w", err)
	}

	err = db.AutoMigrate(migrationModels...)
	if err != nil {
		return nil, fmt.Errorf("数据库自动迁移失败: %w", err)
	}

	return db, nil
}
