package geecache

import pb "geecache/geecachepb"

//抽象出2个接口
//为什么要这么做？抽象出这个2个接口有什么用

//根据key选择相应节点PeerGetter-HTTP的客户端
type PeerPicker interface {
	PickPeer(key string) (peer PeerGetter, ok bool)
}

//知道缓存空间Group.name和Key 查找相应的value
type PeerGetter interface {
	Get(in *pb.Request, out *pb.Response) error
	//Get(group string, key string) ([]byte, error)
}

/*
类型总结
key string
value []byte 缓存值
group Group 缓存空间
group.name string
httpGetter 客户端类 实现PeerGetter接口
节点 string




*/
