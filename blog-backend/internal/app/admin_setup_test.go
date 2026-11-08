package app

import (
	"os"
	"testing"

	usermodule "dh-blog/internal/modules/user"
	"dh-blog/internal/utils"

	"github.com/glebarez/sqlite"
	"golang.org/x/term"
	"gorm.io/gorm"
)

func newAdminSetupDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	if err := db.AutoMigrate(&usermodule.User{}); err != nil {
		t.Fatalf("migrate users: %v", err)
	}
	return db
}

func TestCredentialsFromEnv(t *testing.T) {
	tests := []struct {
		name         string
		username     string
		password     string
		wantOK       bool
		wantUsername string
		wantPassword string
		wantErr      bool
	}{
		{
			name:         "complete pair",
			username:     "admin",
			password:     "s3cret!",
			wantOK:       true,
			wantUsername: "admin",
			wantPassword: "s3cret!",
		},
		{
			// 用户名两侧的空白通常是复制粘贴带进来的，清掉；密码原样保留
			name:         "trims username but keeps password verbatim",
			username:     "  admin  ",
			password:     " s3cret! ",
			wantOK:       true,
			wantUsername: "admin",
			wantPassword: " s3cret! ",
		},
		{
			name:    "nothing configured falls back to prompt",
			wantOK:  false,
			wantErr: false,
		},
		{
			// 只配一半时不能悄悄回落去问终端，那正是无人值守启动出问题的地方
			name:     "half configured pair is an error",
			username: "admin",
			wantOK:   true,
			wantErr:  true,
		},
		{
			name:     "short password is rejected",
			username: "admin",
			password: "123",
			wantOK:   true,
			wantErr:  true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Setenv(envAdminUsername, test.username)
			t.Setenv(envAdminPassword, test.password)

			credentials, ok, err := credentialsFromEnv()
			if (err != nil) != test.wantErr {
				t.Fatalf("error = %v, wantErr %v", err, test.wantErr)
			}
			if ok != test.wantOK {
				t.Fatalf("ok = %v, want %v", ok, test.wantOK)
			}
			if test.wantErr || !test.wantOK {
				return
			}
			if credentials.username != test.wantUsername {
				t.Errorf("username = %q, want %q", credentials.username, test.wantUsername)
			}
			if credentials.password != test.wantPassword {
				t.Errorf("password = %q, want %q", credentials.password, test.wantPassword)
			}
		})
	}
}

func TestEnsureAdminUserSeedsAccountFromEnv(t *testing.T) {
	if term.IsTerminal(int(os.Stdin.Fd())) {
		t.Skip("stdin is a terminal; this case only exercises the headless path")
	}

	t.Setenv(envAdminUsername, "admin")
	t.Setenv(envAdminPassword, "s3cret!")

	db := newAdminSetupDB(t)
	if err := EnsureAdminUser(db); err != nil {
		t.Fatalf("seed admin: %v", err)
	}

	var users []usermodule.User
	if err := db.Find(&users).Error; err != nil {
		t.Fatalf("load users: %v", err)
	}
	if len(users) != 1 {
		t.Fatalf("created %d users, want 1", len(users))
	}
	if users[0].Username != "admin" {
		t.Errorf("username = %q, want %q", users[0].Username, "admin")
	}
	// 落库的必须是哈希，不是明文
	if users[0].Password == "s3cret!" {
		t.Fatal("password was stored in plain text")
	}
	if !utils.CheckPasswordHash("s3cret!", users[0].Password) {
		t.Error("stored hash does not match the configured password")
	}
}

func TestEnsureAdminUserFailsWithoutTerminalOrEnv(t *testing.T) {
	if term.IsTerminal(int(os.Stdin.Fd())) {
		t.Skip("stdin is a terminal; would block on the prompt")
	}

	t.Setenv(envAdminUsername, "")
	t.Setenv(envAdminPassword, "")

	db := newAdminSetupDB(t)
	err := EnsureAdminUser(db)
	if err == nil {
		t.Fatal("expected an error: no env and no terminal to prompt on")
	}

	// 报错退出而不是建个空账号，也绝不能挂住
	var count int64
	if countErr := db.Model(&usermodule.User{}).Count(&count).Error; countErr != nil {
		t.Fatalf("count users: %v", countErr)
	}
	if count != 0 {
		t.Errorf("user count = %d, want 0", count)
	}
}

func TestEnsureAdminUserKeepsExistingUsers(t *testing.T) {
	db := newAdminSetupDB(t)

	hashedPassword, err := utils.HashPassword("existing-user")
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}
	if err := db.Create(&usermodule.User{Username: "someone", Password: hashedPassword}).Error; err != nil {
		t.Fatalf("seed existing user: %v", err)
	}

	// 已经有用户时不该再读环境变量，更不该再建一个
	t.Setenv(envAdminUsername, "admin")
	t.Setenv(envAdminPassword, "s3cret!")
	if err := EnsureAdminUser(db); err != nil {
		t.Fatalf("ensure admin: %v", err)
	}

	var count int64
	if err := db.Model(&usermodule.User{}).Count(&count).Error; err != nil {
		t.Fatalf("count users: %v", err)
	}
	if count != 1 {
		t.Errorf("user count = %d, want 1", count)
	}
}
