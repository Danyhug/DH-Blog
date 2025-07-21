package config

import (
	"errors"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"reflect"
	"time"

	"github.com/spf13/viper"
)

type Server struct {
	Address    string `yaml:"address"`
	HttpPort   int    `yaml:"httpPort"`
	HttpsPort  int    `yaml:"httpsPort"`
	CertFile   string `yaml:"certFile"`
	KeyFile    string `yaml:"keyFile"`
	StaticPath string `yaml:"staticPath"` // 新增：静态文件服务路径
}

type DataBase struct {
	Type   string `yaml:"type"`
	DBFile string `yaml:"dbFile"` // For SQLite, this will be the file path
	Dsn    string `yaml:"dsn"`    // For other databases like MySQL, this will be the DSN
}

type LocalUpload struct {
	Path string `yaml:"path"` // Local upload base path, e.g., "uploads/article"
}

type WebdavUpload struct {
	URL      string `yaml:"url"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
	Path     string `yaml:"path"` // WebDAV base path, e.g., "uploads/webdav"
}

type Upload struct {
	Local  LocalUpload  `yaml:"local"`
	Webdav WebdavUpload `yaml:"webdav"`
}

type Config struct {
	Server    Server   `yaml:"server"`
	DataBase  DataBase `yaml:"database"`
	JwtSecret string   `yaml:"jwtSecret"`
	Upload    Upload   `yaml:"upload"` // New upload configuration
}

func getRandomString(length int) string {
	// 获取一个随机字符串，用于生成 JWT 密钥
	var chars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789!@#$%^&*()_+"
	cLen := len(chars)
	jwtChars := make([]byte, length)
	for ; length > 0; length-- {
		jwtChars = append(jwtChars, chars[rand.Intn(cLen)])
	}
	return string(jwtChars)
}

func DefaultConfig() *Config {
	return &Config{
		Server: Server{
			Address:    "0.0.0.0",
			HttpPort:   2233,
			HttpsPort:  -1,
			StaticPath: "data/upload", // 默认静态文件服务路径
		},
		DataBase: DataBase{
			Type:   "sqlite3",
			DBFile: "data/dhblog.db", // Default SQLite database file
			Dsn:    "",               // DSN will be empty for SQLite by default
		},
		JwtSecret: getRandomString(16),
		Upload: Upload{
			Local: LocalUpload{
				Path: "article", // Default local upload path
			},
			Webdav: WebdavUpload{
				URL:      "", // Default empty WebDAV URL
				Username: "",
				Password: "",
				Path:     "webdav", // Default WebDAV path
			},
		},
	}
}

func Init() (*Config, error) {
	exePath, err := os.Executable()
	if err != nil {
		return nil, fmt.Errorf("获取可执行文件路径失败: %w", err)
	}
	// 获取项目根目录 (假设可执行文件在 cmd/blog-backend/)
	projectRoot := filepath.Dir(exePath)
	dataDir := filepath.Join(projectRoot, "data")
	configFilePath := filepath.Join(dataDir, "config.yaml")

	// 确保 data 目录存在
	if _, err := os.Stat(dataDir); os.IsNotExist(err) {
		if err := os.MkdirAll(dataDir, 0755); err != nil {
			return nil, fmt.Errorf("创建数据目录失败: %w", err)
		}
	}

	// 1. 创建一个 Viper 实例，用于加载默认配置并合并现有文件配置
	v := viper.New()
	v.SetConfigName("config")
	v.SetConfigType("yaml")
	v.AddConfigPath(dataDir)

	// 设置默认值
	defaultCfg := DefaultConfig()
	v.SetDefault("server", defaultCfg.Server)
	v.SetDefault("database", defaultCfg.DataBase)
	v.SetDefault("jwtSecret", defaultCfg.JwtSecret)
	v.SetDefault("upload", defaultCfg.Upload)

	// 2. 尝试读取现有配置文件
	readErr := v.ReadInConfig()
	configExists := true
	if readErr != nil {
		var configFileNotFoundError viper.ConfigFileNotFoundError
		if errors.As(readErr, &configFileNotFoundError) {
			configExists = false // 配置文件不存在
		} else {
			return nil, fmt.Errorf("读取配置文件失败: %w", readErr)
		}
	}

	// 3. 判断是否需要更新配置文件
	needsUpdate := false
	if !configExists {
		needsUpdate = true // 如果配置文件不存在，则需要写入默认配置
	} else {
		// 创建一个临时 Viper 实例，只读取文件内容，不加载默认值
		tempFileViper := viper.New()
		tempFileViper.SetConfigName("config")
		tempFileViper.SetConfigType("yaml")
		tempFileViper.AddConfigPath(dataDir)

		// 确保能读取到文件，如果不能，说明文件有问题，也需要更新
		if err := tempFileViper.ReadInConfig(); err != nil {
			needsUpdate = true
		} else {
			// 比较合并了默认值和文件值的 Viper (v) 与只读取文件值的 Viper (tempFileViper)
			// 如果它们不相等，说明 v 中包含了 tempFileViper 没有的默认值，即有新配置项
			if !reflect.DeepEqual(v.AllSettings(), tempFileViper.AllSettings()) {
				needsUpdate = true
			}
		}
	}

	// 4. 执行更新操作（如果需要）
	if needsUpdate {
		// 备份现有配置文件（如果存在）
		if configExists {
			backupFileName := fmt.Sprintf("config_backup_%s.yaml", time.Now().Format("20060102150405"))
			backupFilePath := filepath.Join(dataDir, backupFileName)
			if err := os.Rename(configFilePath, backupFilePath); err != nil {
				return nil, fmt.Errorf("备份配置文件失败: %w", err)
			}
			fmt.Printf("已备份旧配置文件至: %s\n", backupFilePath)
		}

		// 将合并后的配置（默认值 + 文件值）写入新的 config.yaml
		if err := v.WriteConfigAs(configFilePath); err != nil {
			return nil, fmt.Errorf("写入新配置文件失败: %w", err)
		}
		fmt.Printf("已更新配置文件至: %s\n", configFilePath)
	}

	// 5. Unmarshal 最终配置
	var finalConfig Config
	if err := v.Unmarshal(&finalConfig); err != nil {
		return nil, fmt.Errorf("解析最终配置文件失败: %w", err)
	}

	return &finalConfig, nil
}
