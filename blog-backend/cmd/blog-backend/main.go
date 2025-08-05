package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"dh-blog/internal/config"
	"dh-blog/internal/database"
	"dh-blog/internal/model"
	"dh-blog/internal/repository"
	"dh-blog/internal/service"
	"dh-blog/internal/utils"
	"dh-blog/internal/wire"

	"github.com/sirupsen/logrus"
)

func main() {
	conf, err := config.Init()
	if err != nil {
		logrus.Fatalf("加载配置失败: %v", err)
	}

	// 初始化 JWT 工具
	utils.InitJwtUtils(conf.JwtSecret)

	// 初始化数据库连接和迁移
	db, err := database.Init(conf)
	if err != nil {
		logrus.Fatalf("数据库初始化失败: %v", err)
	}

	// 检查用户是否存在，如果不存在则引导创建管理员用户
	userRepo := repository.NewUserRepository(db)
	first := userRepo.IsFirstStart() // 尝试获取默认管理员用户
	if first {
		logrus.Info("未检测到管理员用户，请创建管理员账户：")
		reader := bufio.NewReader(os.Stdin)

		fmt.Print("请输入管理员用户名: ")
		username, _ := reader.ReadString('\n')
		username = strings.TrimSpace(username)

		fmt.Print("请输入管理员密码: ")
		password, _ := reader.ReadString('\n')
		password = strings.TrimSpace(password)

		hashedPassword, hashErr := utils.HashPassword(password)
		if hashErr != nil {
			logrus.Fatalf("密码哈希失败: %v", hashErr)
		}

		adminUser := &model.User{
			Username: username,
			Password: hashedPassword,
		}

		if createErr := userRepo.CreateUser(adminUser); createErr != nil {
			logrus.Fatalf("创建管理员用户失败: %v", createErr)
		}
		logrus.Info("管理员用户创建成功！")
	} else if err != nil {
		logrus.Fatalf("查询管理员用户失败: %v", err)
	}

	// 初始化缓存服务
	cacheService := service.NewCacheService()

	// 初始化整个应用程序的依赖并获取 Gin 路由器
	router := wire.InitApp(conf, db)

	// 配置 HTTP 服务器
	server := &http.Server{
		Addr:    fmt.Sprintf("%s:%d", conf.Server.Address, conf.Server.HttpPort),
		Handler: router,
	}

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

	// 关闭缓存服务
	cacheService.Shutdown()
	logrus.Info("缓存服务已关闭")

	// 设置关闭超时
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 优雅地关闭HTTP服务器
	if err := server.Shutdown(ctx); err != nil {
		logrus.Fatalf("服务器关闭失败: %v", err)
	}

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
