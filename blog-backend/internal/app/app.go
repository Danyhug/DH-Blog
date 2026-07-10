package app

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"dh-blog/internal/config"
	"dh-blog/internal/router"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// App 持有应用级依赖和生命周期组件。
type App struct {
	Config          *config.Config
	DB              *gorm.DB
	Router          *gin.Engine
	StaticFilesPath string
	startOnce       sync.Once
	shutdownOnce    sync.Once
	starts          []func()
	shutdowns       []func()
}

// New 初始化应用依赖、业务模块和路由。
func New(conf *config.Config, db *gorm.DB) (*App, error) {
	paths, err := resolvePaths(conf)
	if err != nil {
		return nil, err
	}

	build := newBuildContext(conf, db, paths)
	routeModules, err := build.buildModules()
	if err != nil {
		build.cleanupAfterBuildFailure()
		return nil, err
	}

	logrus.Info("应用程序核心组件初始化完成")

	engine := router.Init(router.Options{
		Config:    conf,
		IPService: build.logging().IPService(),
		JWT:       build.jwtService,
	}, routeModules...)

	return &App{
		Config:          conf,
		DB:              db,
		Router:          engine,
		StaticFilesPath: paths.StaticFilesPath,
		starts:          build.starts(),
		shutdowns:       build.shutdowns(),
	}, nil
}

// Start 启动应用拥有的后台组件。
func (a *App) Start() {
	a.startOnce.Do(func() {
		for _, start := range a.starts {
			start()
		}
	})
}

// Shutdown 停止应用持有的后台组件。
func (a *App) Shutdown() {
	a.shutdownOnce.Do(func() {
		for _, shutdown := range a.shutdowns {
			shutdown()
		}
	})
}

type applicationPaths struct {
	DataDir            string
	DatabasePath       string
	StaticFilesPath    string
	DefaultStoragePath string
}

func resolvePaths(conf *config.Config) (applicationPaths, error) {
	exePath, err := os.Executable()
	if err != nil {
		return applicationPaths{}, fmt.Errorf("获取可执行文件路径失败: %w", err)
	}

	exeDir := filepath.Dir(exePath)
	dataDir := filepath.Join(exeDir, "data")
	databasePath := conf.DataBase.DBFile
	if !filepath.IsAbs(databasePath) {
		databasePath = filepath.Join(exeDir, databasePath)
	}
	return applicationPaths{
		DataDir:            dataDir,
		DatabasePath:       databasePath,
		StaticFilesPath:    filepath.Join(dataDir, "upload"),
		DefaultStoragePath: filepath.Join(dataDir, "webdav"),
	}, nil
}
