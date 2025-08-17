package service

import (
	"sync"

	"dh-blog/internal/dhcache"

	"github.com/sirupsen/logrus"
)

// CacheService 缓存服务，用于管理全局缓存实例
type CacheService struct {
	cache dhcache.Cache
}

var (
	cacheInstance *CacheService
	cacheOnce     sync.Once
)

// NewCacheService 创建或获取缓存服务实例（单例模式）
func NewCacheService() *CacheService {
	cacheOnce.Do(func() {
		logrus.Info("初始化缓存服务")
		cacheInstance = &CacheService{
			cache: dhcache.NewCache(),
		}
	})
	return cacheInstance
}

// GetCache 获取缓存实例
func (s *CacheService) GetCache() dhcache.Cache {
	return s.cache
}

// Shutdown 关闭缓存服务
func (s *CacheService) Shutdown() {
	if s.cache != nil {
		logrus.Info("关闭缓存服务")
		s.cache.Shutdown()
	}
}
