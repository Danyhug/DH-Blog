package logging

import (
	"fmt"
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

func TestResolveCityCachesLookupResult(t *testing.T) {
	module, db, cache := newTestModule(t)
	repository := module.repository
	ip := "203.0.113.9"
	lookupCalls := 0
	lookup := func(string) (string, error) {
		lookupCalls++
		return "中国移动/测试省/测试市", nil
	}

	city, err := repository.ResolveCity(ip, lookup)
	if err != nil {
		t.Fatalf("first resolve: %v", err)
	}
	if city != "中国移动/测试省/测试市" || lookupCalls != 1 {
		t.Fatalf("first resolve = %q, calls = %d; want resolved city, 1 call", city, lookupCalls)
	}

	// 内存缓存命中，不再调外部
	city, err = repository.ResolveCity(ip, lookup)
	if err != nil || city != "中国移动/测试省/测试市" || lookupCalls != 1 {
		t.Fatalf("memory-cached resolve = %q, calls = %d; want cached city, 1 call", city, lookupCalls)
	}

	// 清掉内存缓存，仍应命中数据库
	cache.Delete(getIPCityCacheKey(ip))
	city, err = repository.ResolveCity(ip, lookup)
	if err != nil || city != "中国移动/测试省/测试市" || lookupCalls != 1 {
		t.Fatalf("db-cached resolve = %q, calls = %d; want cached city, 1 call", city, lookupCalls)
	}

	var entry IPCityCache
	if err := db.Where("ip = ?", ip).First(&entry).Error; err != nil {
		t.Fatalf("read ip_city_cache: %v", err)
	}
	if entry.City != "中国移动/测试省/测试市" {
		t.Fatalf("persisted city = %q", entry.City)
	}
}

func TestResolveCitySkipsLookupForLocalNetwork(t *testing.T) {
	module, db, _ := newTestModule(t)
	repository := module.repository
	lookupCalls := 0
	lookup := func(string) (string, error) {
		lookupCalls++
		return "本地网络", nil
	}

	city, err := repository.ResolveCity("127.0.0.1", lookup)
	if err != nil {
		t.Fatalf("resolve local ip: %v", err)
	}
	if city != "本地网络" {
		t.Fatalf("local resolve = %q, want 本地网络", city)
	}
	var count int64
	if err := db.Model(&IPCityCache{}).Where("ip = ?", "127.0.0.1").Count(&count).Error; err != nil {
		t.Fatalf("count local city cache: %v", err)
	}
	if count != 0 {
		t.Fatalf("local ip persisted %d rows, want 0", count)
	}
}

func TestResolveCityRefreshesStaleDatabaseEntry(t *testing.T) {
	module, db, cache := newTestModule(t)
	repository := module.repository
	ip := "203.0.113.10"
	if err := db.Create(&IPCityCache{IP: ip, City: "旧省/旧市", UpdatedAt: time.Now().Add(-60 * 24 * time.Hour)}).Error; err != nil {
		t.Fatalf("seed stale city: %v", err)
	}
	lookupCalls := 0
	lookup := func(string) (string, error) {
		lookupCalls++
		return "新省/新市", nil
	}

	city, err := repository.ResolveCity(ip, lookup)
	if err != nil {
		t.Fatalf("resolve stale ip: %v", err)
	}
	if city != "新省/新市" || lookupCalls != 1 {
		t.Fatalf("stale resolve = %q, calls = %d; want refreshed city, 1 call", city, lookupCalls)
	}
	if cached, ok := cache.Get(getIPCityCacheKey(ip)); !ok || cached != "新省/新市" {
		t.Fatalf("refreshed cache = %#v, %v; want 新省/新市, true", cached, ok)
	}
	var entry IPCityCache
	if err := db.Where("ip = ?", ip).First(&entry).Error; err != nil {
		t.Fatalf("read refreshed city: %v", err)
	}
	if entry.City != "新省/新市" {
		t.Fatalf("refreshed db city = %q", entry.City)
	}
}

func TestResolveCityPropagatesLookupError(t *testing.T) {
	module, db, _ := newTestModule(t)
	repository := module.repository
	lookup := func(string) (string, error) {
		return "", fmt.Errorf("外部IP库不可达")
	}

	if _, err := repository.ResolveCity("203.0.113.11", lookup); err == nil {
		t.Fatal("ResolveCity() error = nil, want lookup error")
	}
	var count int64
	if err := db.Model(&IPCityCache{}).Where("ip = ?", "203.0.113.11").Count(&count).Error; err != nil {
		t.Fatalf("count failed city cache: %v", err)
	}
	if count != 0 {
		t.Fatalf("failed lookup persisted %d rows, want 0", count)
	}
}
