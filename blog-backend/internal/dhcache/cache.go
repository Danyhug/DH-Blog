package dhcache

import (
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

const (
	// 默认缓存过期时间为两分钟
	defaultExpire = time.Second * 60 * 2
	gcTime        = time.Second * 60
)

type Cache interface {
	Set(key string, value interface{}, duration ...time.Duration) error
	Get(key string) (interface{}, bool)
	Delete(key string) bool
	SetNx(key string, value interface{}, duration ...time.Duration) bool
	Shutdown()
}

// item 缓存中存储的数据
type item struct {
	Value      interface{}
	ExpireTime time.Time
}

type DHCache struct {
	mu     sync.RWMutex
	items  map[string]item
	gcStop chan struct{}
}

func (d *DHCache) Set(key string, value interface{}, duration ...time.Duration) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	expireTime := defaultExpire
	if len(duration) == 1 {
		expireTime = duration[0]
	}

	d.items[key] = item{value, time.Now().Add(expireTime)}
	return nil
}

func (d *DHCache) Get(key string) (interface{}, bool) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	v, ok := d.items[key]
	if !ok {
		return nil, false
	}

	// 判断当前时间是否晚于过期时间
	if time.Now().After(v.ExpireTime) {
		return nil, false
	}

	return v.Value, true
}

func (d *DHCache) SetNx(key string, value interface{}, duration ...time.Duration) bool {
	d.mu.Lock()
	defer d.mu.Unlock()

	_, ok := d.items[key]
	if ok {
		return false
	}

	expireTime := defaultExpire
	if len(duration) == 1 {
		expireTime = duration[0]
	}

	d.items[key] = item{value, time.Now().Add(expireTime)}
	return true
}

// janitor 定时清理过期数据
func (d *DHCache) janitor() {
	logrus.Infof("DHCache 清洁工 [启动]")
	// 每15秒扫描一次对象，看是否需要被回收
	tick := time.NewTicker(gcTime)
	defer tick.Stop()

	for {
		select {
		case <-tick.C:
			d.mu.Lock()
			logrus.Info("DHCache 清洁工 [正在工作中]")
			now := time.Now()
			for index, v := range d.items {
				if now.After(v.ExpireTime) {
					// 已过期，清理
					delete(d.items, index)
				}
			}
			d.mu.Unlock()
			logrus.Info("DHCache 清洁工 [工作完成]")
		case <-d.gcStop:
			logrus.Info("DHCache 清洁工 [停止]")
			return
		}
	}
}

func (d *DHCache) Delete(key string) bool {
	d.mu.Lock()
	defer d.mu.Unlock()

	_, ok := d.items[key]
	if ok {
		delete(d.items, key)
		return true
	}
	return false
}

func (d *DHCache) Shutdown() {
	close(d.gcStop)
}

func NewCache() Cache {
	cache := DHCache{
		items:  make(map[string]item),
		gcStop: make(chan struct{}),
	}
	go cache.janitor()
	return &cache
}
