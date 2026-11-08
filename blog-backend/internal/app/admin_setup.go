package app

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	usermodule "dh-blog/internal/modules/user"
	"dh-blog/internal/utils"

	"github.com/sirupsen/logrus"
	"golang.org/x/term"
	"gorm.io/gorm"
)

const (
	// Seeding the first account from the environment keeps container deployments
	// working: `docker run` without -it hands the process an empty or closed
	// stdin, where a prompt either blocks forever or returns an empty name.
	envAdminUsername = "DHBLOG_ADMIN_USERNAME"
	envAdminPassword = "DHBLOG_ADMIN_PASSWORD"

	minAdminPasswordLength = 6
)

type adminCredentials struct {
	username string
	password string
}

// EnsureAdminUser 创建首次启动所需的管理员账号。
func EnsureAdminUser(db *gorm.DB) error {
	userRepo := usermodule.NewRepository(db)
	if !userRepo.IsFirstStart() {
		return nil
	}

	credentials, err := firstStartCredentials()
	if err != nil {
		return err
	}

	hashedPassword, err := utils.HashPassword(credentials.password)
	if err != nil {
		return fmt.Errorf("密码哈希失败: %w", err)
	}

	adminUser := &usermodule.User{
		Username: credentials.username,
		Password: hashedPassword,
	}

	if err := userRepo.Create(adminUser); err != nil {
		return fmt.Errorf("创建管理员用户失败: %w", err)
	}

	logrus.Infof("管理员用户 %q 创建成功", credentials.username)
	return nil
}

// firstStartCredentials prefers the environment and only prompts on a real
// terminal. Anything else is a configuration error: reading a prompt from a
// non-interactive stdin is exactly what used to hang unattended starts.
func firstStartCredentials() (adminCredentials, error) {
	if credentials, ok, err := credentialsFromEnv(); ok || err != nil {
		return credentials, err
	}

	if !term.IsTerminal(int(os.Stdin.Fd())) {
		return adminCredentials{}, fmt.Errorf(
			"首次启动需要创建管理员账户，但标准输入不是终端：请设置环境变量 %s 与 %s，"+
				"或在终端 / docker run -it 中交互式启动",
			envAdminUsername, envAdminPassword)
	}

	return credentialsFromPrompt()
}

// credentialsFromEnv reports ok=false when the pair is not fully configured, so
// the caller can fall back to the prompt; a half-configured pair is an error
// rather than a silent fallback.
func credentialsFromEnv() (adminCredentials, bool, error) {
	username := strings.TrimSpace(os.Getenv(envAdminUsername))
	password := os.Getenv(envAdminPassword)
	if username == "" && password == "" {
		return adminCredentials{}, false, nil
	}
	if username == "" || password == "" {
		return adminCredentials{}, true, fmt.Errorf(
			"%s 与 %s 需要同时设置", envAdminUsername, envAdminPassword)
	}

	credentials := adminCredentials{username: username, password: password}
	if err := credentials.validate(); err != nil {
		return adminCredentials{}, true, err
	}

	logrus.Infof("使用环境变量创建管理员账户：%s", username)
	return credentials, true, nil
}

func credentialsFromPrompt() (adminCredentials, error) {
	logrus.Info("未检测到管理员用户，请创建管理员账户：")
	reader := bufio.NewReader(os.Stdin)

	fmt.Print("请输入管理员用户名: ")
	username, _ := reader.ReadString('\n')

	fmt.Print("请输入管理员密码: ")
	password, _ := reader.ReadString('\n')

	credentials := adminCredentials{
		username: strings.TrimSpace(username),
		// Only the trailing newline is trimmed: a password may legitimately
		// start or end with a space, and silently eating it would lock the
		// operator out on the next login.
		password: strings.TrimRight(password, "\r\n"),
	}
	if err := credentials.validate(); err != nil {
		return adminCredentials{}, err
	}
	return credentials, nil
}

func (c adminCredentials) validate() error {
	if c.username == "" {
		return fmt.Errorf("管理员用户名不能为空")
	}
	if len([]rune(c.password)) < minAdminPasswordLength {
		return fmt.Errorf("管理员密码至少需要 %d 个字符", minAdminPasswordLength)
	}
	return nil
}
