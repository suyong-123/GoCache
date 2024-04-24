package geecache

//抽象了一个只读数据结构ByteView用来表示缓存值
type ByteView struct {
	b []byte //存储真实的缓存值
}

//我们在 lru.Cache 的实现中，要求被缓存对象必须实现 Value 接口，即 Len() int 方法，返回其所占的内存大小。
func (v ByteView) Len() int { //作用域为ByteView的拷贝对象，修改不会反射到原对象，防止缓存值被外部程序修改
	return len(v.b)
}

func (v ByteView) ByteSlice() []byte { //获得切片副本
	return cloneBytes(v.b)
}

func cloneBytes(b []byte) []byte {
	c := make([]byte, len(b))
	copy(c, b)
	return c
}

func (v ByteView) String() string {
	return string(v.b)
}
