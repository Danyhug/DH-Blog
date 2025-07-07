package config

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

type Server struct {
	Address   string `yaml:"address"`
	HttpPort  int    `yaml:"httpPort"`
	HttpsPort int    `yaml:"httpsPort"`
	CertFile  string `yaml:"certFile"`
	KeyFile   string `yaml:"keyFile"`
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

func DefaultConfig() *Config {
	return &Config{
		Server: Server{
			Address:   "0.0.0.0",
			HttpPort:  2233,
			HttpsPort: -1,
		},
		DataBase: DataBase{
			Type:   "sqlite3",
			DBFile: "../blog.db", // Default SQLite database file
			Dsn:    "",           // DSN will be empty for SQLite by default
		},
		JwtSecret: "test",
		Upload: Upload{
			Local: LocalUpload{
				Path: "uploads/article", // Default local upload path
			},
			Webdav: WebdavUpload{
				URL:      "", // Default empty WebDAV URL
				Username: "",
				Password: "",
				Path:     "uploads/webdav", // Default WebDAV path
			},
		},
	}
}

func Init() (*Config, error) {
	exePath, err := os.Executable()
	if err != nil {
		return nil, err
	}
	dataDir := filepath.Join(filepath.Dir(exePath), "data")
	if _, err := os.Stat(dataDir); os.IsNotExist(err) {
		if err := os.MkdirAll(dataDir, 0755); err != nil {
			return nil, err
		}
	}

	v := viper.New()
	v.SetConfigName("config")
	v.SetConfigType("yaml")
	v.AddConfigPath(dataDir)

	if err := v.ReadInConfig(); err != nil {
		var configFileNotFoundError viper.ConfigFileNotFoundError
		if errors.As(err, &configFileNotFoundError) {
			// Config file not found; create it with default values.
			defaultConfig := DefaultConfig()
			v.Set("server", defaultConfig.Server)
			v.Set("database", defaultConfig.DataBase)
			v.Set("jwtSecret", defaultConfig.JwtSecret)
			v.Set("upload", defaultConfig.Upload) // Set new upload config
			if err := v.WriteConfigAs(filepath.Join(dataDir, "config.yaml")); err != nil {
				return nil, err
			}
		} else {
			// Error reading config file
			return nil, err
		}
	}

	var config Config
	if err := v.Unmarshal(&config); err != nil {
		return nil, err
	}

	return &config, nil
}
