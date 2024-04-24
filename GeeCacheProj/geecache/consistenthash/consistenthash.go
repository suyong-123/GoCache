package consistenthash

import (
	"hash/crc32"
	"sort"
	"strconv"
)

// 函数类型 允许用于替换成自定义的 Hash 函数，也方便测试时替换，默认为 crc32.ChecksumIEEE 算法。
type Hash func(key []byte) uint32

type Map struct {
	hash     Hash           //哈希函数
	replicas int            //虚拟节点倍数
	Nodes    []int          //哈希环
	NodeMap  map[int]string //虚拟节点和真实节点映射表 虚拟节点：真实节点
}

// Map 类型实例化 允许自定义哈希环函数和虚拟节点倍数
func New(replicas int, fn Hash) *Map {
	m := &Map{
		hash:     fn,
		replicas: replicas,
		NodeMap:  make(map[int]string),
	}
	if m.hash == nil { //如果没有自定义哈希函数，就使用默认的
		m.hash = crc32.ChecksumIEEE

	}
	return m
}

// 添加真实节点（缓存服务器）
func (m *Map) Add(Nodes ...string) { //允许传入多个string类型的参数
	for _, node := range Nodes {
		//对每一个真实节点 key，对应创建 m.replicas 个虚拟节点
		for i := 0; i < m.replicas; i++ {
			xunihash := int(m.hash([]byte(strconv.Itoa(i) + node))) //使用m.hash()计算虚拟节点的哈希值
			m.Nodes = append(m.Nodes, xunihash)
			m.NodeMap[xunihash] = node //在 hashMap 中增加虚拟节点和真实节点的映射关系。
		}

	}
	sort.Ints(m.Nodes) //排序

}

// 查询Peer，通过key，去查询key所在的节点
func (m *Map) Get(key string) string {
	if len(m.Nodes) == 0 { //如果没有缓存服务器
		return ""
	}
	//计算key的哈希值
	keyhash := int(m.hash([]byte(key))) //m.hash函数输入的参数是[]byte ，所以要将string类型转换为[]byte ;输入的参数是uint32,转换为int
	//顺时针找到第一个匹配的虚拟节点的下标 idx
	idx := sort.Search(len(m.Nodes), func(i int) bool {
		return m.Nodes[i] >= keyhash
	})

	v := m.Nodes[idx%len(m.Nodes)] //从 m.keys 中获取到对应的哈希值 因为 m.keys 是一个环状结构，所以用取余数的方式来处理这种情况。
	//在m.hashMap找到对应真实节点
	return m.NodeMap[v]

}
