package config

import (
	"crypto/rand"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"time"

	"github.com/go-viper/mapstructure/v2"
	"github.com/spf13/viper"
)

type Server struct {
	Address   string        `yaml:"address"`
	HttpPort  int           `yaml:"httpPort"`
	JwtExpire time.Duration `yaml:"jwtExpire"`
}

type DataBase struct {
	// Only the SQLite file is configurable: the driver is linked in directly and
	// there is no code path that would read a DSN or a database type.
	DBFile string `yaml:"dbFile"`
}

type WebDAVServer struct {
	Enabled bool   `yaml:"enabled"` // 是否启用 WebDAV 服务端
	Prefix  string `yaml:"prefix"`  // WebDAV 路由前缀，默认 "/dav"
}

// AIGateway 配置 AI 网关（一期：搜索网关）。供应商密钥与配额存在数据库里，
// 这里只放需要重启才会变、且与部署强相关的运行参数。
type AIGateway struct {
	Enabled          bool          `yaml:"enabled"`          // 是否对外开放网关接口
	CacheTTL         time.Duration `yaml:"cacheTTL"`         // 搜索结果缓存时长，0 表示关闭
	UpstreamTimeout  time.Duration `yaml:"upstreamTimeout"`  // 单次上游调用超时
	QueueWait        time.Duration `yaml:"queueWait"`        // 出站限速最长排队时间
	LogRetentionDays int           `yaml:"logRetentionDays"` // 请求日志保留天数
}

type Config struct {
	Server       Server       `yaml:"server"`
	DataBase     DataBase     `yaml:"database"`
	JwtSecret    string       `yaml:"jwtSecret"`
	WebDAVServer WebDAVServer `yaml:"webdavServer"` // WebDAV 服务端配置
	AIGateway    AIGateway    `yaml:"aiGateway"`    // AI 网关配置
	LogLevel     string       `yaml:"logLevel"`     // 日志级别: debug/info/warn/error，默认 info
}

// getRandomString 生成 JWT 密钥。用 crypto/rand 而不是 math/rand：这个值直接决定
// 令牌可被伪造的难度，可预测的伪随机源不合适。
func getRandomString(length int) string {
	const chars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789!@#$%^&*()_+"
	jwtChars := make([]byte, length)
	// Rejection sampling keeps every character equally likely; the modulo
	// shortcut would favour the first 256%len(chars) characters.
	limit := byte(256 - 256%len(chars))
	for index := 0; index < length; {
		buffer := make([]byte, 1)
		if _, err := rand.Read(buffer); err != nil {
			// crypto/rand failing means the platform is broken; the caller cannot
			// recover either, so surfacing it beats silently minting a weak secret.
			panic(fmt.Sprintf("生成 JWT 密钥失败: %v", err))
		}
		if buffer[0] >= limit {
			continue
		}
		jwtChars[index] = chars[int(buffer[0])%len(chars)]
		index++
	}
	return string(jwtChars)
}

func DefaultConfig() *Config {
	return &Config{
		Server: Server{
			Address:   "0.0.0.0",
			HttpPort:  2233,
			JwtExpire: time.Hour * 24 * 30, // 默认一个月
		},
		DataBase: DataBase{
			DBFile: "data/dhblog.db", // Default SQLite database file
		},
		JwtSecret: getRandomString(16),
		WebDAVServer: WebDAVServer{
			Enabled: true,
			Prefix:  "/dav",
		},
		AIGateway: AIGateway{
			Enabled:          true,
			CacheTTL:         time.Minute * 15,
			UpstreamTimeout:  time.Second * 15,
			QueueWait:        time.Second * 2,
			LogRetentionDays: 90,
		},
		LogLevel: "info",
	}
}

// retiredSettingKeys 是曾经写入过 config.yaml、但现在没有任何代码读取的配置项。
// 留着它们比删掉更糟：改这些值完全没效果，看起来却像是生效的开关。
// 只删键、不动结构，老配置升级时由 Init 顺手清掉一次。
//
// 必须全小写：viper 的键大小写不敏感，AllSettings() 交出来的是小写形式
// （config.yaml 里也写成 httpsport 而不是 httpsPort）。
var retiredSettingKeys = [][]string{
	{"server", "httpsport"},  // HTTPS 从未实现
	{"server", "certfile"},   // 同上
	{"server", "keyfile"},    // 同上
	{"server", "staticpath"}, // 实际路径来自 system_settings.file_storage_path
	{"database", "type"},     // 只支持 SQLite，驱动写死在 database.Init
	{"database", "dsn"},      // 同上，没有其它数据库的连接逻辑
	{"upload"},               // 上传路径同样由 system_settings 管理
}

// dropRetiredSettings 就地删除退役键，返回是否真的删掉了东西。
func dropRetiredSettings(settings map[string]any) bool {
	dropped := false
	for _, path := range retiredSettingKeys {
		node := settings
		for _, key := range path[:len(path)-1] {
			child, ok := node[key].(map[string]any)
			if !ok {
				node = nil
				break
			}
			node = child
		}
		if node == nil {
			continue
		}
		leaf := path[len(path)-1]
		if _, ok := node[leaf]; ok {
			delete(node, leaf)
			dropped = true
		}
	}
	return dropped
}

// writeSettings 把整理过的配置落盘。不复用原来的 viper：它没有删除键的公开接口，
// 另起一个实例写入才能把退役键真正从文件里去掉。
func writeSettings(settings map[string]any, path string) error {
	out := viper.New()
	for key, value := range settings {
		out.Set(key, value)
	}
	return out.WriteConfigAs(path)
}

func Init() (*Config, error) {
	exePath, err := os.Executable()
	if err != nil {
		return nil, fmt.Errorf("获取可执行文件路径失败: %w", err)
	}
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
	v.SetDefault("server", map[string]any{
		"address":   defaultCfg.Server.Address,
		"httpPort":  defaultCfg.Server.HttpPort,
		"jwtExpire": defaultCfg.Server.JwtExpire,
	})
	v.SetDefault("database", map[string]any{
		"dbFile": defaultCfg.DataBase.DBFile,
	})
	v.SetDefault("jwtSecret", defaultCfg.JwtSecret)
	v.SetDefault("webdavServer", map[string]any{
		"enabled": defaultCfg.WebDAVServer.Enabled,
		"prefix":  defaultCfg.WebDAVServer.Prefix,
	})
	v.SetDefault("aiGateway", map[string]any{
		"enabled":          defaultCfg.AIGateway.Enabled,
		"cacheTTL":         defaultCfg.AIGateway.CacheTTL,
		"upstreamTimeout":  defaultCfg.AIGateway.UpstreamTimeout,
		"queueWait":        defaultCfg.AIGateway.QueueWait,
		"logRetentionDays": defaultCfg.AIGateway.LogRetentionDays,
	})
	v.SetDefault("logLevel", defaultCfg.LogLevel)

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

	// 4. Unmarshal 配置到结构体（使用自定义 DecodeHook 处理 time.Duration）
	var finalConfig Config
	if err := v.Unmarshal(&finalConfig, viper.DecodeHook(
		mapstructure.ComposeDecodeHookFunc(
			mapstructure.StringToTimeDurationHookFunc(),
			mapstructure.StringToSliceHookFunc(","),
		),
	)); err != nil {
		return nil, fmt.Errorf("解析最终配置文件失败: %w", err)
	}

	// 5. 执行更新操作（如果需要）
	settings := v.AllSettings()
	if dropRetiredSettings(settings) {
		needsUpdate = true
	}

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

		// 将合并后的配置写入新的 config.yaml
		if err := writeSettings(settings, configFilePath); err != nil {
			return nil, fmt.Errorf("写入新配置文件失败: %w", err)
		}
		fmt.Printf("已更新配置文件至: %s\n", configFilePath)
	}

	return &finalConfig, nil
}
