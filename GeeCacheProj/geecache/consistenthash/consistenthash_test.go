package consistenthash

import (
	"fmt"
	"strconv"
	"testing"
)

func TestHashing(t *testing.T) {
	//自定义hash函数
	//实例化Map结构体
	m := New(3, func(key []byte) uint32 {
		i, _ := strconv.Atoi(string(key))
		return uint32(i)

	})
	m.Add("6", "2", "4") //对应的虚拟节点为06 16 26\02 12 22\04 14 24
	fmt.Print(m.Nodes)   //[2,4,6,12,14,16,22,24,26]
	//缓存数据 2节点里缓存了2 、11、27
	testCases := map[string]string{
		"2":  "2",
		"11": "2",
		"23": "4",
		"27": "2",
	}
	for k, v := range testCases {
		if m.Get(k) != v {
			t.Errorf("Asking for %s,should have yieloaded %s", k, v)
		}

	}
	m.Add("8") // 08 18 28
	// 27 should now map to 8.
	testCases["27"] = "8"

	for k, v := range testCases {
		if m.Get(k) != v {
			t.Errorf("Asking for %s,should have yieloaded %s", k, v)
		}
	}

}
