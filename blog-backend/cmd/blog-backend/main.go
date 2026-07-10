package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"dh-blog/internal/app"
	"dh-blog/internal/config"
	"dh-blog/internal/database"

	"github.com/sirupsen/logrus"
)

func main() {
	conf, err := config.Init()
	if err != nil {
		logrus.Fatalf("加载配置失败: %v", err)
	}

	// 初始化数据库连接和迁移
	db, err := database.Init(conf, app.SchemaModels()...)
	if err != nil {
		logrus.Fatalf("数据库初始化失败: %v", err)
	}

	if err := app.EnsureAdminUser(db); err != nil {
		logrus.Fatalf("初始化管理员用户失败: %v", err)
	}

	application, err := app.New(conf, db)
	if err != nil {
		logrus.Fatalf("初始化应用失败: %v", err)
	}
	application.Start()

	// 配置 HTTP 服务器
	server := &http.Server{
		Addr:    fmt.Sprintf("%s:%d", conf.Server.Address, conf.Server.HttpPort),
		Handler: application.Router,
	}

	// 设置日志级别为 Debug
	logrus.SetLevel(logrus.DebugLevel)

	// 启动 HTTP 服务器（在新的 goroutine 中）
	go func() {
		logrus.Infof("HTTP服务器启动，监听地址: %s", server.Addr)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logrus.Fatalf("无法监听 %s 端口: %v\n", server.Addr, err)
		}
	}()

	// 显示启动信息
	displayInfo(conf)

	// 优雅地关闭服务器
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM) // 监听中断和终止信号
	<-quit                                               // 阻塞直到接收到信号
	logrus.Info("服务器正在关闭...")

	// 设置关闭超时
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 优雅地关闭HTTP服务器
	if err := server.Shutdown(ctx); err != nil {
		logrus.Fatalf("服务器关闭失败: %v", err)
	}

	application.Shutdown()
	logrus.Info("服务器已成功关闭")
}

func displayInfo(conf *config.Config) {
	fmt.Println(`
███████╗ ██╗  ██╗    ██████╗ ██╗      ██████╗  ██████╗ 
██╔═══██╗██║  ██║    ██╔══██╗██║     ██╔═══██╗██╔════╝ 
██║   ██║███████║    ██████╔╝██║     ██║   ██║██║  ███╗
██║   ██║██╔══██║    ██╔══██╗██║     ██║   ██║██║   ██║
███████╔╝██║  ██║    ██████╔╝███████╗╚██████╔╝╚██████╔╝
╚══════╝ ╚═╝  ╚══════╝ ╚═════╝  ╚═════╝ ╚═════╝  ╚═════╝`)
	logrus.Info("[ DH-Blog ] 启动成功")
	logrus.Infof("[ DH-Blog ] 访问地址：%v", fmt.Sprintf("http://%s:%d",
		conf.Server.Address, conf.Server.HttpPort))
}
