package database

import (
	"dh-blog/internal/config"
	"dh-blog/internal/model"
	"github.com/sirupsen/logrus"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// Init 初始化数据库连接并执行自动迁移
func Init(conf *config.Config) *gorm.DB {
	// 初始化数据库连接
	db, err := gorm.Open(sqlite.Open(conf.DataBase.DBFile), &gorm.Config{}) // 使用 SQLite 驱动并从配置中读取数据库文件路径
	if err != nil {
		logrus.Fatalf("连接数据库失败: %v", err)
	}

	// 自动迁移模型到数据库表
	// GORM 会根据 model 包中定义的结构体自动创建或更新数据库表
	err = db.AutoMigrate(
		&model.AccessLog{},
		&model.Article{},
		&model.Category{},
		&model.Comment{},
		&model.DailyStat{},
		&model.IPStat{},
		&model.Tag{},
		&model.User{},
	)
	if err != nil {
		logrus.Fatalf("数据库自动迁移失败: %v", err)
	}

	logrus.Info("数据库连接和自动迁移成功")
	return db
}
