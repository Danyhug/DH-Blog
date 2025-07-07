package wire

import (
	"dh-blog/internal/config"
	"dh-blog/internal/handler"
	"dh-blog/internal/repository"
	"dh-blog/internal/router"
	"dh-blog/internal/service"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// InitApp 初始化整个应用程序的依赖
func InitApp(conf *config.Config, db *gorm.DB) *gin.Engine {
	// 2. 初始化 Repository 层
	userRepo := repository.NewUserRepository(db)
	tagRepo := repository.NewTagRepository(db)
	categoryRepo := repository.NewCategoryRepository(db)
	commentRepo := repository.NewCommentRepository(db)
	logRepo := repository.NewLogRepository(db)
	dailyStatsRepo := repository.NewDailyStatsRepository(db)
	articleRepo := repository.NewArticleRepository(db, categoryRepo, tagRepo)

	// 初始化 Uploader
	localUploader := service.NewLocalUploader(conf)
	webdavUploader := service.NewWebdavUploader(conf)

	// 初始化 UploadService
	uploadService := service.NewUploadService(localUploader, webdavUploader)

	// 3. 初始化 Handler 层，直接注入 Repository 和缓存
	articleHandler := handler.NewArticleHandler(articleRepo, tagRepo, categoryRepo, dailyStatsRepo)
	userHandler := handler.NewUserHandler(userRepo)
	commentHandler := handler.NewCommentHandler(commentRepo)
	logHandler := handler.NewLogHandler(logRepo, dailyStatsRepo)
	adminHandler := handler.NewAdminHandler(uploadService)

	// 4. 初始化路由器并注册路由
	appRouter := router.Init(articleHandler, userHandler, commentHandler, logHandler, adminHandler)

	return appRouter
}
