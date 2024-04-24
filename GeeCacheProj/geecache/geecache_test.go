package geecache

import (
	"fmt"
	"log"
	"reflect"
	"testing"
)

// 测试回调函数
func TestGetter(t *testing.T) {
	var g Getter = GetterFunc(func(key string) ([]byte, error) {
		return []byte(key), nil //这个函数的功能就是将字符串转换为字节数组
	})
	expect := []byte("key")
	if v, _ := g.Get("key"); !reflect.DeepEqual(v, expect) { //reflect.DeepEqual检查两个接口是否指向相同的底层值
		t.Errorf("callback failed")
	}
}

var db = map[string]string{
	"Tom":  "630",
	"Jack": "589",
	"Sam":  "567",
	" ":    "000",
}

// 测试Get方法
func TestGet(t *testing.T) {
	t.Helper()

	loadCounts := make(map[string]int, len(db)) //使用loadCounts统计某个键调用回调函数的次数
	//创建Group实例
	gee := NewGroup("scores", 2<<10, GetterFunc(
		func(key string) ([]byte, error) {
			log.Println("[SlowDB] search key", key)
			if v, ok := db[key]; ok {
				if _, ok := loadCounts[key]; ok {
					loadCounts[key] = 0

				}
				loadCounts[key] += 1
				return []byte(v), nil
			}
			return nil, fmt.Errorf("%s not exist", key)
		}))

	for k, v := range db {
		//缓存为空的情况下，能够通过回调函数获取源数据
		if view, err := gee.Get(k); err != nil || view.String() != v {
			t.Fatalf("failed to get value of Tom")

		}
		////缓存已经存在的情况下，调用回调函数次数大于1，说明没有缓存
		if _, err := gee.Get(k); err != nil || loadCounts[k] > 1 {
			t.Fatalf("cache %s miss", k)
		}
	}
	if view, err := gee.Get("unknown"); err == nil {
		t.Fatalf("the value of unknown should be empty, but %s got", view)
	}

}
