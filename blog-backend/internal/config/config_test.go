package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/viper"
)

func TestGetRandomString(t *testing.T) {
	const length = 32
	first := getRandomString(length)

	if len(first) != length {
		t.Fatalf("len = %d, want %d", len(first), length)
	}
	// 早先的实现用 make([]byte, length) 再 append，结果前 length 个字节是 NUL，
	// 落盘后写成 "\0\0…wS4GDq"，看起来像密钥但其实只有一半是随机字符。
	if strings.ContainsRune(first, 0) {
		t.Fatalf("secret contains NUL bytes: %q", first)
	}
	if second := getRandomString(length); second == first {
		t.Fatalf("two calls returned the same secret: %q", first)
	}
}

func TestDropRetiredSettings(t *testing.T) {
	// 键必须全小写，和 viper.AllSettings() 交出来的形状一致
	settings := map[string]any{
		"server": map[string]any{
			"address":    "0.0.0.0",
			"httpport":   2233,
			"httpsport":  -1,
			"certfile":   "",
			"staticpath": "data/upload",
		},
		"database": map[string]any{
			"dbfile": "data/dhblog.db",
			"type":   "sqlite3",
			"dsn":    "",
		},
		"upload": map[string]any{
			"local": map[string]any{"path": "article"},
		},
		"loglevel": "info",
	}

	if !dropRetiredSettings(settings) {
		t.Fatal("reported nothing removed")
	}

	server := settings["server"].(map[string]any)
	for _, key := range []string{"httpsport", "certfile", "staticpath"} {
		if _, ok := server[key]; ok {
			t.Errorf("server.%s survived", key)
		}
	}
	// 仍在用的键不能跟着一起掉
	if server["address"] != "0.0.0.0" || server["httpport"] != 2233 {
		t.Errorf("server = %v, lost a live key", server)
	}

	database := settings["database"].(map[string]any)
	for _, key := range []string{"type", "dsn"} {
		if _, ok := database[key]; ok {
			t.Errorf("database.%s survived", key)
		}
	}
	if database["dbfile"] != "data/dhblog.db" {
		t.Errorf("database = %v, lost dbFile", database)
	}

	if _, ok := settings["upload"]; ok {
		t.Error("upload survived")
	}
	if settings["loglevel"] != "info" {
		t.Errorf("loglevel = %v", settings["loglevel"])
	}

	// 已经干净时不该再报「删过东西」，否则每次启动都会重写配置
	if dropRetiredSettings(settings) {
		t.Error("second run reported changes on an already clean map")
	}
}

func TestWriteSettingsRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")

	settings := map[string]any{
		"server": map[string]any{
			"address":   "0.0.0.0",
			"httpPort":  2233,
			"jwtExpire": "720h0m0s",
		},
		"jwtSecret": "abc",
		"logLevel":  "info",
	}
	if err := writeSettings(settings, path); err != nil {
		t.Fatalf("write: %v", err)
	}

	reloaded := viper.New()
	reloaded.SetConfigFile(path)
	if err := reloaded.ReadInConfig(); err != nil {
		t.Fatalf("reload: %v", err)
	}
	if got := reloaded.GetInt("server.httpPort"); got != 2233 {
		t.Errorf("server.httpPort = %d", got)
	}
	// Duration 必须仍是可解析的字符串，而不是纳秒整数
	if got := reloaded.GetString("server.jwtExpire"); got != "720h0m0s" {
		t.Errorf("server.jwtExpire = %q", got)
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read back: %v", err)
	}
	for _, retired := range []string{"httpsPort", "dsn", "upload:"} {
		if strings.Contains(string(raw), retired) {
			t.Errorf("written config still contains %q:\n%s", retired, raw)
		}
	}
}
