package main

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"dh-blog/internal/config"
	"dh-blog/internal/wire"
	"github.com/sirupsen/logrus"
)

func main() {
	conf := config.DefaultConfig()

	// 初始化整个应用程序的依赖并获取 Gin 路由器
	router := wire.InitApp(conf)

	// 配置 HTTP 服务器
	server := &http.Server{
		Addr:    fmt.Sprintf("%s:%d", conf.Server.Address, conf.Server.HttpPort),
		Handler: router,
	}

	// 启动 HTTP 服务器（在新的 goroutine 中）
	go func() {
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logrus.Fatalf("无法监听 %s 端口: %v\n", server.Addr, err)
		}
	}()

	// 显示启动信息
	displayInfo()

	// 优雅地关闭服务器
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM) // 监听中断和终止信号
	<-quit                                               // 阻塞直到接收到信号
	logrus.Info("服务器正在关闭...")

}

func displayInfo() {
	fmt.Println(`
███████╗ ██╗  ██╗    ██████╗ ██╗      ██████╗  ██████╗ 
██╔═══██╗██║  ██║    ██╔══██╗██║     ██╔═══██╗██╔════╝ 
██║   ██║███████║    ██████╔╝██║     ██║   ██║██║  ███╗
██║   ██║██╔══██║    ██╔══██╗██║     ██║   ██║██║   ██║
███████╔╝██║  ██║    ██████╔╝███████╗╚██████╔╝╚██████╔╝
╚══════╝ ╚═╝  ╚══════╝ ╚═════╝ ╚══════╝ ╚═════╝  ╚═════╝`)
	logrus.Info("[ DH-Blog ] 启动成功")
	logrus.Infof("[ DH-Blog ] 访问地址：%v\n", fmt.Sprintf("http://%s:%d",
		config.DefaultConfig().Server.Address, config.DefaultConfig().Server.HttpPort))
	logrus.Info("[ DH-Blog ] 默认用户名：admin")
	logrus.Info("[ DH-Blog ] 默认密码：admin")
}
