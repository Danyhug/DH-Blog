package config

import (
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

type Config struct {
	Server    Server   `yaml:"server"`
	DataBase  DataBase `yaml:"database"`
	JwtSecret string   `yaml:"jwtSecret"`
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
			DBFile: "data/dhblog.db", // Default SQLite database file
			Dsn:    "",               // DSN will be empty for SQLite by default
		},
		JwtSecret: "test",
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
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			// Config file not found; create it with default values.
			defaultConfig := DefaultConfig()
			v.Set("server", defaultConfig.Server)
			v.Set("database", defaultConfig.DataBase)
			v.Set("jwtSecret", defaultConfig.JwtSecret)
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
