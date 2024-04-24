package geecache

/*
实现客户端输入http://example.com/_geecache/groupname/key 实现查询缓存的请求。main函数使用curl进行测试
服务端回复请求响应 将缓存值作为响应的body回应

*/

import (
	"fmt"
	"geecache/consistenthash"
	pb "geecache/geecachepb"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"sync"

	"google.golang.org/protobuf/proto"
)

//服务端功能

const defaultBasePath = "/_geecache/"

// 作为承载节点间 HTTP 通信的核心数据结构
type HTTPPool struct {
	self        string //用来记录自己的地址，包括主机名/IP 和端口。
	basePath    string //http://example.com/_geecache/ 开头的请求
	lock        sync.Mutex
	peers       *consistenthash.Map    //一致性哈希 根据key选择节点
	httpGetters map[string]*httpGetter //每个远程节点对应一个httpGetter; key： http://10.0.0.2:8008
}

// 实例化
func NewHTTPPool(self string) *HTTPPool {
	return &HTTPPool{
		self:     self,
		basePath: defaultBasePath,
	}
}

// HTTPPool结构体实现http.Handler接口里的ServeHTTP方法
// 实现HTTP响应
func (p *HTTPPool) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	//首先判断访问路径的前缀是否是 basePath，不是返回错误。
	//r.URL.Path是/_geecache
	log.Println(r.URL.Path)
	if !strings.HasPrefix(r.URL.Path, p.basePath) {
		panic("HTTPPool serving unexpected path: " + r.URL.Path)
	}
	//约定访问路径格式为 /<basepath>/<groupname>/<key>
	parts := strings.SplitN(r.URL.Path[len(p.basePath):], "/", 2) //strings.SplitN将一个给定的字符串按给定的分隔符分割成n个子串
	//parts=[groupname key]
	if len(parts) != 2 {
		http.Error(w, "bas request", http.StatusBadRequest)
		return
	}
	groupname := parts[0]
	key := parts[1]

	//通过 groupname 得到 group 实例
	group := GetGroup(groupname)
	if group == nil {
		http.Error(w, "no such group: "+groupname, http.StatusNotFound)
		return

	}

	//知道缓存名字后获得缓存空间，然后从缓存空间中通过key获得缓存值value
	value, _ := group.Get(key)
	// Write the value to the response body as a proto message.
	body, err := proto.Marshal(&pb.Response{Value: value.ByteSlice()})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	//最终使用 w.Write() 将缓存值作为 httpResponse 的 body 返回。
	w.Header().Set("Content-Type", "application/octet-stream") //设置 HTTP 响应头部的 Content-Type 字段为 "application/octet-stream"，表示响应的内容类型为二进制流。
	//w.Write(value.ByteSlice())
	w.Write(body)

}

//客户端功能

type httpGetter struct {
	baseURL string //baseURL 表示将要访问的远程节点的地址，例如 http://example.com/_geecache/。
}

// 客户端类要实现PeerGetter接口，就必须实现接口下的方法Get,从Group和key得到缓存值
// func (h *httpGetter) Get(group string, key string) ([]byte, error) {
func (h *httpGetter) Get(in *pb.Request, out *pb.Response) error {
	u := fmt.Sprintf( //格式化
		"%v%v/%v", //按值的本来值除数
		h.baseURL,
		//url.QueryEscape(group), //对参数惊醒转码使之可以安全用在URL查询里
		//url.QueryEscape(key),
		url.QueryEscape(in.GetGroup()),
		url.QueryEscape(in.GetKey()),
	) //输出：http://example.com/_geecache/groupname/key
	//当我们请求服务器时，服务器发送的响应包体被保存在Body中。可以使用它提供的Read方法来获取数据内容。结束的时候，需要调用Body中的Close()方法关闭io。
	res, err := http.Get(u) //向指定的URL发起Get请求，返回响应

	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK { //响应的状态码不等于http成功状态码200
		return fmt.Errorf("server returned: %v", res.StatusCode)
	}

	bytes, err := io.ReadAll(res.Body) //读取响应body ,上述服务端返回将缓存值作为body
	if err != nil {
		return fmt.Errorf("reading response body: %v", res.Body)
	}
	//使用 proto.Unmarshal() 解码 HTTP 响应。
	if err = proto.Unmarshal(bytes, out); err != nil {
		return fmt.Errorf("decoding response body: %v", err)
	}
	//return bytes, nil
	return nil

}

// 确保*httpGetter类型实现了PeerGetter接口,编译时检查。如果 *httpGetter 类型没有实现 PeerGetter 接口，这一行代码将在编译时引发错误。
var _ PeerGetter = (*httpGetter)(nil)

//实现PeerPick接口，这个接口里的方法PickPeer实现：从key选择节点，

// Set() 方法实例化了一致性哈希算法，并将其他节点的地址添加到 HTTP 池中。这样，HTTP 池知道了其他节点的地址，并可以与它们进行通信。
const defaultReplicas = 50

func (p *HTTPPool) Set(peers ...string) {
	p.lock.Lock()
	defer p.lock.Unlock()

	p.peers = consistenthash.New(defaultReplicas, nil) //复习New参数：每个真实节点有多少个虚拟节点，如果没有自定义哈希函数（nil），就使用默认的
	p.peers.Add(peers...)                              //添加节点
	p.httpGetters = make(map[string]*httpGetter, len(peers))
	//为每一个节点创建了一个 HTTP 客户端 httpGetter。
	for _, peer := range peers {
		p.httpGetters[peer] = &httpGetter{baseURL: peer + p.basePath} //使用 peer + p.basePath 构建一个 baseURL。
	}

}

// PickerPeer() 包装了一致性哈希算法的 Get() 方法，根据具体的 key，选择节点，返回节点对应的 HTTP 客户端。
func (p *HTTPPool) PickPeer(key string) (PeerGetter, bool) {
	p.lock.Lock()
	defer p.lock.Unlock()

	if peer := p.peers.Get(key); peer != "" && peer != p.self { //peer是key对应的节点
		log.Printf("Pick peer %s", peer)
		return p.httpGetters[peer], true

	}
	return nil, false

}

var _ PeerPicker = (*HTTPPool)(nil) //HTTPPOOL类型实现PeerPicker 接口
