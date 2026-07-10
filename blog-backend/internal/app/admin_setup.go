package app

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	usermodule "dh-blog/internal/modules/user"
	"dh-blog/internal/utils"

	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// EnsureAdminUser 创建首次启动所需的管理员账号。
func EnsureAdminUser(db *gorm.DB) error {
	userRepo := usermodule.NewRepository(db)
	if !userRepo.IsFirstStart() {
		return nil
	}

	logrus.Info("未检测到管理员用户，请创建管理员账户：")
	reader := bufio.NewReader(os.Stdin)

	fmt.Print("请输入管理员用户名: ")
	username, _ := reader.ReadString('\n')
	username = strings.TrimSpace(username)

	fmt.Print("请输入管理员密码: ")
	password, _ := reader.ReadString('\n')
	password = strings.TrimSpace(password)

	hashedPassword, err := utils.HashPassword(password)
	if err != nil {
		return fmt.Errorf("密码哈希失败: %w", err)
	}

	adminUser := &usermodule.User{
		Username: username,
		Password: hashedPassword,
	}

	if err := userRepo.Create(adminUser); err != nil {
		return fmt.Errorf("创建管理员用户失败: %w", err)
	}

	logrus.Info("管理员用户创建成功！")
	return nil
}
