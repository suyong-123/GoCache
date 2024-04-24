package singleflight

import "sync"

//正在进行或已经结束的请求
type call struct {
	wg  sync.WaitGroup
	val interface{}
	err error
}

//管理不同key的请求call，在这个表里说明key正在被请求
type ManegeCall struct {
	lock sync.Mutex //保护哈希表 不被并发读写而加上的锁
	m    map[string]*call
}

//针对相同的key，无论Do被调用多少次，函数fn都只会被调用一次
func (mc *ManegeCall) Do(key string, fn func() (interface{}, error)) (interface{}, error) {
	mc.lock.Lock()

	//初始化
	if mc.m == nil {
		mc.m = make(map[string]*call)
	}
	//对哈希表进行读操作
	if c, ok := mc.m[key]; ok {
		mc.lock.Unlock()
		c.wg.Wait() //阻塞，如果请求正在进行中，则等待
		return c.val, c.err

	}
	//对哈希表写操作
	c := new(call)
	c.wg.Add(1)   //锁加1，发起请求前枷锁
	mc.m[key] = c //添加到哈希表里，表明 key 已经有对应的请求在处理
	mc.lock.Unlock()

	//调用fn，发起请求
	c.val, c.err = fn()
	c.wg.Done() //锁减1 请求结束

	//删除操作
	mc.lock.Lock()
	defer mc.lock.Unlock()
	delete(mc.m, key)

	return c.val, c.err
}
