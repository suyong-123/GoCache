package geecache

/*
将cache缓存结构体（key-value）封装在group缓存空间中
实现：通过group.name获取这个缓存空间
1、然后在这个缓存空间中再通过key去查询缓存值value  g.mainCache.get(key)
2、如果没有这个缓存，则调佣getter回调函数获得源数据，并进行缓存
3、如果本地服务器（本节点）没有缓存，则调佣getFromPeer()从远程节点获取


*/

/*
结构体嵌套总结：从而实现继承，大的结构体可以直接访问小的结构体的方法
Group缓存空间结构体包含cache , cache包含*lru.Cache
例如通过key查找value ： g.mainCache.get(key) 而实际上get(key)方法使用的是c.lru.Get(key)，这个Get（key）是(c *Cache) Get(key string) (val Value, ok bool)



*/

import (
	"fmt"
	pb "geecache/geecachepb"
	"geecache/singleflight"
	"log"
	"sync"
)

//设计一个回调函数，当缓存不存在时，调用这个函数，得到源数据

// 定义一个接口
type Getter interface {
	Get(key string) ([]byte, error)
}

// 定义一个函数类型
type GetterFunc func(key string) ([]byte, error)

// 函数类型实现接口（接口型函数）
func (g GetterFunc) Get(key string) ([]byte, error) {
	return g(key)
}

//定义一个函数类型 F，并且实现接口 A 的方法，然后在这个方法中调用自己。这是 Go 语言中将其他函数（参数返回值定义与 F 一致）转换为接口 A 的常用技巧。

// 一个 Group 可以认为是一个缓存空间
type Group struct {
	name      string
	getter    Getter
	mainCache cache
	peers     PeerPicker

	loader *singleflight.ManegeCall
}

var (
	rwlock sync.RWMutex //读写锁
	groups = make(map[string]*Group)
)

// 实例化Group
func NewGroup(name string, cacheBytes int64, getter Getter) *Group {
	if getter == nil {
		panic("nil Getter")
	}
	rwlock.Lock() //写锁
	defer rwlock.Unlock()
	g := &Group{
		name:      name,
		getter:    getter,
		mainCache: cache{cacheBytes: cacheBytes},
		loader:    &singleflight.ManegeCall{},
	}
	groups[name] = g //将 group 存储在全局变量 groups 中
	return g
}
func GetGroup(name string) *Group {
	rwlock.RLock()
	g := groups[name]
	rwlock.RUnlock()
	return g
}

// Get value for a key from cache

func (g *Group) Get(key string) (ByteView, error) {
	if key == "" {
		return ByteView{}, fmt.Errorf("key is required")
	}
	if v, ok := g.mainCache.get(key); ok { //从 mainCache 中查找缓存
		log.Println("[GeeCache]hit")
		return v, nil
	}
	//如果缓存中没有，调用load方法
	return g.load(key)
}

// 新增RegisterPeers()方法,实现了 PeerPicker 接口的 HTTPPool 注入到 Group 中。
// 将创建的 HTTP 池 peers 注册到缓存组 gee 中。这使得缓存组知道如何与其他节点通信，并在分布式系统中共享和管理缓存数据。
func (g *Group) RegisterPeers(peers PeerPicker) {
	if g.peers != nil {
		panic("RegisterPeerPicker called more than once")
	}
	g.peers = peers

}

// 修改 load 方法，使用 PickPeer() 方法选择节点，若非本机节点，则调用 getFromPeer() 从远程获取。若是本机节点或失败，则回退到 getLocally()。
// 修改 load 函数，将原来的 load 的逻辑，使用 g.loader.Do 包裹起来即可，这样确保了并发场景下针对相同的 key，load 过程只会调用一次。
func (g *Group) load(key string) (value ByteView, err error) {
	viewi, err := g.loader.Do(key, func() (interface{}, error) {
		if g.peers != nil {
			if peer, ok := g.peers.PickPeer(key); ok {
				if value, err = g.getFromPeer(peer, key); err == nil {
					return value, nil
				}
				log.Println("[GeeCache] Failed to get from peer", err)

			}
		}
		return g.getLocally(key)

	})
	if err == nil {
		return viewi.(ByteView), nil
	}
	return

}

// getLocally 调用用户回调函数 g.getter.Get() 获取源数据，并且将源数据添加到缓存 mainCache 中
func (g *Group) getLocally(key string) (ByteView, error) {
	bytes, err := g.getter.Get(key) //如果缓存中没有，就是用回调结构体中的Get方法获取指定键的源数据
	if err != nil {
		return ByteView{}, err
	}
	value := ByteView{b: cloneBytes(bytes)}
	g.populateCache(key, value)
	return value, nil

}
func (g *Group) populateCache(key string, value ByteView) {
	g.mainCache.add(key, value)
}

// 实现 PeerGetter 接口的 httpGetter 从访问远程节点，获取缓存值。
func (g *Group) getFromPeer(peer PeerGetter, key string) (ByteView, error) {
	//bytes, err := peer.Get(g.name, key)
	req := &pb.Request{
		Group: g.name,
		Key:   key,
	}
	res := &pb.Response{}
	err := peer.Get(req, res)
	if err != nil {
		return ByteView{}, nil
	}
	return ByteView{b: res.Value}, nil

}
