package database

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"dh-blog/internal/config"
	"dh-blog/internal/model"

	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"github.com/glebarez/sqlite"
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

	// 分类现有的系统设置
	if err := updateSystemSettingsType(db); err != nil {
		return nil, fmt.Errorf("更新系统设置类型失败: %w", err)
	}

	return db, nil
}

// updateSystemSettingsType 更新系统设置的配置类型
func updateSystemSettingsType(db *gorm.DB) error {
	// 博客基本配置相关键
	blogKeys := []string{
		model.SettingKeyBlogTitle,
		model.SettingKeySignature,
		model.SettingKeyAvatar,
		model.SettingKeyGithubLink,
		model.SettingKeyBilibiliLink,
		model.SettingKeyOpenBlog,
		model.SettingKeyOpenComment,
	}
	// 邮件配置相关键
	emailKeys := []string{
		model.SettingKeyCommentEmailNotify,
		model.SettingKeySmtpHost,
		model.SettingKeySmtpPort,
		model.SettingKeySmtpUser,
		model.SettingKeySmtpPass,
		model.SettingKeySmtpSender,
	}
	// AI配置相关键
	aiKeys := []string{
		model.SettingKeyAiApiURL,
		model.SettingKeyAiApiKey,
		model.SettingKeyAiModel,
		model.SettingKeyAiPromptGetTags,
		model.SettingKeyAiPromptGetAbstract,
	}
	// 存储配置相关键
	storageKeys := []string{
		model.SettingKeyFileStoragePath,
	}

	// 更新博客基本配置类型
	for _, key := range blogKeys {
		err := db.Exec("UPDATE system_settings SET config_type = ? WHERE setting_key = ?", model.ConfigTypeBlog, key).Error
		if err != nil {
			return fmt.Errorf("更新博客配置类型失败: %w", err)
		}
	}

	// 更新邮件配置类型
	for _, key := range emailKeys {
		err := db.Exec("UPDATE system_settings SET config_type = ? WHERE setting_key = ?", model.ConfigTypeEmail, key).Error
		if err != nil {
			return fmt.Errorf("更新邮件配置类型失败: %w", err)
		}
	}

	// 更新AI配置类型
	for _, key := range aiKeys {
		err := db.Exec("UPDATE system_settings SET config_type = ? WHERE setting_key = ?", model.ConfigTypeAI, key).Error
		if err != nil {
			return fmt.Errorf("更新AI配置类型失败: %w", err)
		}
	}

	// 更新存储配置类型
	for _, key := range storageKeys {
		err := db.Exec("UPDATE system_settings SET config_type = ? WHERE setting_key = ?", model.ConfigTypeStorage, key).Error
		if err != nil {
			return fmt.Errorf("更新存储配置类型失败: %w", err)
		}
	}

	// 记录更新完成
	fmt.Println("系统设置配置类型更新完成")
	return nil
}
