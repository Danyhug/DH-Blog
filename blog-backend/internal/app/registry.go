package app

import (
	"context"
	"fmt"

	"dh-blog/internal/config"
	"dh-blog/internal/dhcache"
	adminmodule "dh-blog/internal/modules/admin"
	articlemodule "dh-blog/internal/modules/article"
	commentmodule "dh-blog/internal/modules/comment"
	filesmodule "dh-blog/internal/modules/files"
	loggingmodule "dh-blog/internal/modules/logging"
	sharemodule "dh-blog/internal/modules/share"
	systemmodule "dh-blog/internal/modules/system"
	usermodule "dh-blog/internal/modules/user"
	webdavmodule "dh-blog/internal/modules/webdav"
	"dh-blog/internal/platform/ai"
	"dh-blog/internal/router"
	"dh-blog/internal/task"
	"dh-blog/internal/utils"

	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// moduleRegistration is the single application-level declaration for a
// feature module. Its position controls route registration order.
type moduleRegistration struct {
	Name            string
	MigrationModels func() []any
	Build           func(*buildContext) (router.Module, error)
}

func noMigrationModels() []any { return nil }

// moduleRegistrations keeps the existing route order. Build functions use the
// context below to construct and retain dependencies that are shared between
// modules, so route order does not have to become dependency order.
var moduleRegistrations = []moduleRegistration{
	{
		Name:            "article",
		MigrationModels: articlemodule.MigrationModels,
		Build: func(ctx *buildContext) (router.Module, error) {
			return ctx.article()
		},
	},
	{
		Name:            "user",
		MigrationModels: usermodule.MigrationModels,
		Build: func(ctx *buildContext) (router.Module, error) {
			return ctx.user(), nil
		},
	},
	{
		Name:            "comment",
		MigrationModels: commentmodule.MigrationModels,
		Build: func(ctx *buildContext) (router.Module, error) {
			return ctx.comment(), nil
		},
	},
	{
		Name:            "admin",
		MigrationModels: noMigrationModels,
		Build: func(ctx *buildContext) (router.Module, error) {
			return adminmodule.New(ctx.files().Service()), nil
		},
	},
	{
		Name:            "logging",
		MigrationModels: loggingmodule.MigrationModels,
		Build: func(ctx *buildContext) (router.Module, error) {
			return ctx.logging(), nil
		},
	},
	{
		Name:            "system",
		MigrationModels: systemmodule.MigrationModels,
		Build: func(ctx *buildContext) (router.Module, error) {
			return ctx.system()
		},
	},
	{
		Name:            "files",
		MigrationModels: filesmodule.MigrationModels,
		Build: func(ctx *buildContext) (router.Module, error) {
			return ctx.files(), nil
		},
	},
	{
		Name:            "share",
		MigrationModels: sharemodule.MigrationModels,
		Build: func(ctx *buildContext) (router.Module, error) {
			return ctx.share(), nil
		},
	},
	{
		Name:            "webdav",
		MigrationModels: noMigrationModels,
		Build: func(ctx *buildContext) (router.Module, error) {
			return webdavmodule.New(webdavmodule.Dependencies{
				Enabled: ctx.conf.WebDAVServer.Enabled,
				Prefix:  ctx.conf.WebDAVServer.Prefix,
				Users:   ctx.user(),
				Files:   ctx.files().Service(),
			}), nil
		},
	},
}

// buildContext is the application composition container. Concrete modules are
// deliberately kept here (rather than in a service locator exposed to feature
// packages) so cross-module dependencies remain visible at the composition root.
type buildContext struct {
	conf  *config.Config
	db    *gorm.DB
	paths applicationPaths

	cache      dhcache.Cache
	jwtService *utils.JWTService
	tasks      *task.TaskManager
	aiService  articlemodule.AIService

	userModule    *usermodule.Module
	commentModule *commentmodule.Module
	loggingModule *loggingmodule.Module
	filesModule   *filesmodule.Module
	systemModule  *systemmodule.Module
	articleModule *articlemodule.Module
	shareModule   *sharemodule.Module
}

func newBuildContext(conf *config.Config, db *gorm.DB, paths applicationPaths) *buildContext {
	cache := dhcache.NewCache()
	logrus.Info("缓存服务初始化完成")
	return &buildContext{
		conf:       conf,
		db:         db,
		paths:      paths,
		cache:      cache,
		jwtService: utils.NewJWTService(conf.JwtSecret, conf.Server.JwtExpire),
	}
}

func (ctx *buildContext) user() *usermodule.Module {
	if ctx.userModule == nil {
		ctx.userModule = usermodule.New(ctx.db, ctx.jwtService)
	}
	return ctx.userModule
}

func (ctx *buildContext) comment() *commentmodule.Module {
	if ctx.commentModule == nil {
		ctx.commentModule = commentmodule.New(ctx.db)
	}
	return ctx.commentModule
}

func (ctx *buildContext) logging() *loggingmodule.Module {
	if ctx.loggingModule == nil {
		ctx.loggingModule = loggingmodule.New(ctx.db, ctx.cache)
	}
	return ctx.loggingModule
}

func (ctx *buildContext) files() *filesmodule.Module {
	if ctx.filesModule == nil {
		ctx.filesModule = filesmodule.New(filesmodule.Dependencies{
			DB:                 ctx.db,
			StaticFilesPath:    ctx.paths.StaticFilesPath,
			InitialStoragePath: ctx.paths.DefaultStoragePath,
			InitialChunkSizeKB: 5120,
		})
	}
	return ctx.filesModule
}

func (ctx *buildContext) system() (*systemmodule.Module, error) {
	if ctx.systemModule != nil {
		return ctx.systemModule, nil
	}
	module, err := systemmodule.New(systemmodule.Dependencies{
		DB:           ctx.db,
		Cache:        ctx.cache,
		DataDir:      ctx.paths.DataDir,
		DatabasePath: ctx.paths.DatabasePath,
		Storage:      ctx.files().StorageRuntime(),
	})
	if err != nil {
		return nil, fmt.Errorf("初始化系统模块失败: %w", err)
	}
	ctx.systemModule = module
	if err := ctx.files().Service().EnsureProtectedDirectories(context.Background()); err != nil {
		logrus.Warnf("创建固定目录失败: %v", err)
	}
	return module, nil
}

func (ctx *buildContext) article() (*articlemodule.Module, error) {
	if ctx.articleModule != nil {
		return ctx.articleModule, nil
	}
	system, err := ctx.system()
	if err != nil {
		return nil, err
	}
	if ctx.aiService == nil {
		ctx.aiService = ai.NewAIService(system.AIConfigSource(), ctx.cache)
	}
	if ctx.tasks == nil {
		ctx.tasks = task.NewTaskManager()
	}
	module, err := articlemodule.New(articlemodule.Dependencies{
		DB:             ctx.db,
		Cache:          ctx.cache,
		AI:             ctx.aiService,
		CommentCounter: ctx.comment(),
		Tasks:          ctx.tasks,
	})
	if err != nil {
		return nil, fmt.Errorf("初始化文章模块失败: %w", err)
	}
	ctx.articleModule = module
	return module, nil
}

func (ctx *buildContext) share() *sharemodule.Module {
	if ctx.shareModule == nil {
		ctx.shareModule = sharemodule.New(sharemodule.Dependencies{
			DB:          ctx.db,
			FileService: ctx.files().Service(),
		})
	}
	return ctx.shareModule
}

func (ctx *buildContext) buildModules() ([]router.Module, error) {
	modules := make([]router.Module, 0, len(moduleRegistrations))
	for _, registration := range moduleRegistrations {
		module, err := registration.Build(ctx)
		if err != nil {
			return nil, fmt.Errorf("构建模块 %s: %w", registration.Name, err)
		}
		modules = append(modules, module)
	}
	return modules, nil
}

func (ctx *buildContext) starts() []func() {
	if ctx.tasks == nil {
		return nil
	}
	return []func(){ctx.tasks.Start}
}

func (ctx *buildContext) shutdowns() []func() {
	shutdowns := make([]func(), 0, 3)
	if ctx.tasks != nil {
		shutdowns = append(shutdowns, ctx.tasks.Stop)
	}
	if ctx.shareModule != nil {
		shutdowns = append(shutdowns, ctx.shareModule.Shutdown)
	}
	return append(shutdowns, ctx.cache.Shutdown)
}

func (ctx *buildContext) cleanupAfterBuildFailure() {
	if ctx.shareModule != nil {
		ctx.shareModule.Shutdown()
	}
	ctx.cache.Shutdown()
}
