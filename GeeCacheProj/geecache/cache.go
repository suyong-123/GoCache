package geecache

import (
	"sync"

	"geecache/lru"
)

//添加并发特性

type cache struct {
	lock       sync.Mutex
	lru        *lru.Cache
	cacheBytes int64 //最大缓存
}

// 实例化 lru，封装 get 和 add 方法，并添加互斥锁
func (c *cache) add(key string, value ByteView) {
	c.lock.Lock()
	defer c.lock.Unlock()
	if c.lru == nil {
		c.lru = lru.New(c.cacheBytes) //创建实例
	}
	c.lru.Add(key, value)
}

func (c *cache) get(key string) (value ByteView, ok bool) {
	c.lock.Lock()
	defer c.lock.Unlock()
	if c.lru == nil {
		return
	}
	if v, ok := c.lru.Get(key); ok {
		return v.(ByteView), ok
	}
	return
}
