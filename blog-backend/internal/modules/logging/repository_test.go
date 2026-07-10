package logging

import (
	"sync"
	"testing"
	"time"
)

type cacheItem struct {
	value any
}

type memoryCache struct {
	mu    sync.RWMutex
	items map[string]cacheItem
}

func newMemoryCache() *memoryCache {
	return &memoryCache{items: make(map[string]cacheItem)}
}

func (c *memoryCache) Set(key string, value interface{}, _ ...time.Duration) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.items[key] = cacheItem{value: value}
	return nil
}

func (c *memoryCache) Get(key string) (interface{}, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	item, ok := c.items[key]
	return item.value, ok
}

func (c *memoryCache) Delete(key string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	if _, ok := c.items[key]; !ok {
		return false
	}
	delete(c.items, key)
	return true
}

func (c *memoryCache) SetNx(key string, value interface{}, _ ...time.Duration) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	if _, exists := c.items[key]; exists {
		return false
	}
	c.items[key] = cacheItem{value: value}
	return true
}

func (c *memoryCache) Shutdown() {}

func TestRepositoryBatchesAccessLogs(t *testing.T) {
	module, db, _ := newTestModule(t)
	repository := module.repository
	repository.batchSize = 2

	if err := repository.SaveAccessLog(nil); err != nil {
		t.Fatalf("save nil access log: %v", err)
	}
	if err := repository.SaveAccessLog(&AccessLog{IPAddress: "192.0.2.1", AccessDate: time.Now()}); err != nil {
		t.Fatalf("buffer first access log: %v", err)
	}
	var count int64
	if err := db.Model(&AccessLog{}).Count(&count).Error; err != nil {
		t.Fatalf("count buffered access logs: %v", err)
	}
	if count != 0 {
		t.Fatalf("stored logs before batch threshold = %d, want 0", count)
	}

	if err := repository.SaveAccessLog(&AccessLog{IPAddress: "192.0.2.2", AccessDate: time.Now()}); err != nil {
		t.Fatalf("flush access log batch: %v", err)
	}
	if err := db.Model(&AccessLog{}).Count(&count).Error; err != nil {
		t.Fatalf("count flushed access logs: %v", err)
	}
	if count != 2 {
		t.Fatalf("stored logs after batch threshold = %d, want 2", count)
	}
}

func TestRepositoryBanAndUnbanIPUpdatesCache(t *testing.T) {
	module, db, cache := newTestModule(t)
	repository := module.repository
	ip := "198.51.100.4"

	if err := repository.BanIP(ip, "test", time.Now().Add(time.Hour)); err != nil {
		t.Fatalf("ban IP: %v", err)
	}
	banned, err := repository.IsIPBanned(ip)
	if err != nil {
		t.Fatalf("check banned IP: %v", err)
	}
	if !banned {
		t.Fatal("IsIPBanned() = false after ban")
	}
	if cached, ok := cache.Get(getIPBlacklistCacheKey(ip)); !ok || cached != true {
		t.Fatalf("ban cache = %#v, %v; want true, true", cached, ok)
	}

	if err := repository.UnbanIP(ip); err != nil {
		t.Fatalf("unban IP: %v", err)
	}
	banned, err = repository.IsIPBanned(ip)
	if err != nil {
		t.Fatalf("check unbanned IP: %v", err)
	}
	if banned {
		t.Fatal("IsIPBanned() = true after unban")
	}
	if cached, ok := cache.Get(getIPBlacklistCacheKey(ip)); !ok || cached != false {
		t.Fatalf("unban cache = %#v, %v; want false, true", cached, ok)
	}
	var active int64
	if err := db.Model(&IPBlacklist{}).Where("ip_address = ?", ip).Count(&active).Error; err != nil {
		t.Fatalf("count active bans: %v", err)
	}
	if active != 0 {
		t.Fatalf("active bans after unban = %d, want 0", active)
	}
	var history int64
	if err := db.Unscoped().Model(&IPBlacklist{}).Where("ip_address = ?", ip).Count(&history).Error; err != nil {
		t.Fatalf("count ban history: %v", err)
	}
	if history != 1 {
		t.Fatalf("ban history after unban = %d, want 1", history)
	}
}
