package lru

import (
	"testing"
)

type String string

func (d String) Len() int {
	return len(d)
}

// 测试查找/访问
// 一是当键存在于缓存中时，能够正确返回对应的值；二是当键不存在于缓存中时，能够正确返回不存在的信息。
func TestGet(t *testing.T) {
	lru := New(int64(0))            //创建实例函数，缓存最大容量为0
	lru.Add("key1", String("1234")) //向缓存中添加键值对
	if v, ok := lru.Get("key1"); !ok || string(v.(String)) != "1234" {
		t.Fatalf("cache hit key1=1234 fails")
	}
	if _, ok := lru.Get("key2"); ok {
		t.Fatalf("cache miss key2 failes")
	}

}

// 测试淘汰
func TestRemoveoldest(t *testing.T) {
	k1, k2, k3 := "key1", "key2", "key3"
	v1, v2, v3 := "value1", "value2", "value3"
	cap := len(k1 + k2 + v1 + v2) //字符串长度20
	//fmt.Println(cap)
	lru := New(int64(cap))
	lru.Add(k1, String(v1))
	lru.Add(k2, String(v2))
	lru.Add(k3, String(v3)) //k3-v3加入后，超过最大内存，就应该把队首k1-v1淘汰了
	lru.Get(k2)             //访问了k2,k2到队尾，此时k3在队首
	lru.Add(k1, String(v3)) //应该把k3淘汰

	if _, ok := lru.Get("key3"); ok || lru.Len() != 2 {
		t.Fatalf("Removeoldest key1 failed")
	}
}
