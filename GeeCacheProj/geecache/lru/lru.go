package lru

import "container/list"

type Cache struct {
	cache    map[string]*list.Element //列表里的指针
	l1       *list.List               //列表
	nbytes   int64                    //内存
	maxBytes int64                    //缓存最大值
}

func New(maxBytes int64) *Cache { //相当于初始化
	return &Cache{
		cache:    make(map[string]*list.Element),
		l1:       list.New(),
		maxBytes: maxBytes,
	}
}

type Value interface {
	Len() int
}

//Cache这个类型实现了Value接口定义的方法
func (c *Cache) Len() int {
	return c.l1.Len() //返回链表的长度
}

type entry struct {
	key   string
	value Value
}

//查找 访问记录
func (c *Cache) Get(key string) (val Value, ok bool) {
	if ele, ok := c.cache[key]; ok {
		c.l1.MoveToFront(ele) //移到队尾
		kv := ele.Value.(*entry)
		return kv.value, true //返回节点

	}
	return

}

//淘汰 移除最少访问的节点（队首）
func (c *Cache) RemoveOldest() {
	ele := c.l1.Back() //取队首节点
	if ele != nil {
		c.l1.Remove(ele) //从链表中删掉节点
		kv := ele.Value.(*entry)
		delete(c.cache, kv.key)                                //从map中删除映射关系
		c.nbytes -= int64(len(kv.key)) + int64(kv.value.Len()) //把key和value的长度从内存中减掉

	}

}

//新增
func (c *Cache) Add(key string, value Value) {
	//如果键存在，更新对应节点的值
	if ele, ok := c.cache[key]; ok {
		kv := ele.Value.(*entry)
		kv.value = value
		c.nbytes += int64(value.Len()) - int64(kv.value.Len()) //更新内存，加上现在的vlaue长度，减去原来的value长度
		c.l1.MoveToFront(ele)                                  //修改相当于访问了，把节点移动到队尾
	} else { //如果不存在
		ele := c.l1.PushFront(&entry{key, value})        //在队尾加入新的节点
		c.cache[key] = ele                               //在map中添加映射
		c.nbytes += int64(len(key)) + int64(value.Len()) //加内存
	}
	//保持内存不超过最大值,超过时，执行淘汰
	for c.maxBytes != 0 && c.nbytes > c.maxBytes {
		c.RemoveOldest()
	}

}
