package main

import (
	"flag"
	"fmt"
	"geecache"
	"log"
	"net/http"
)

var db = map[string]string{
	"Tom":  "630",
	"Jack": "589",
	"Sam":  "567",
}

// //Group实例化，缓存空间的名字groupname为“score”,自定义了一个回调函数，从数据库db中获取数据
func createGroup() *geecache.Group {
	return geecache.NewGroup("scores", 2<<10, geecache.GetterFunc(
		func(key string) ([]byte, error) {
			log.Println("[SlowDB] search key", key)
			if v, ok := db[key]; ok {
				return []byte(v), nil
			}
			return nil, fmt.Errorf("%s not exist", key)
		}))
}

// startCacheServer() 用来启动缓存服务器：创建 HTTPPool，添加节点信息，注册到 gee 中，启动 HTTP 服务（共3个端口，8001/8002/8003），用户不感知。
func startCacheServer(addr string, addrs []string, gee *geecache.Group) { //addr 服务器地址 addrs包含了其他节点地址的切片列表
	peers := geecache.NewHTTPPool(addr)
	peers.Set(addrs...)
	gee.RegisterPeers(peers)
	log.Println("geecache is running at", addr)
	log.Fatal(http.ListenAndServe(addr[7:], peers))

}

//startAPIServer() 用来启动一个 API 服务（端口 9999），与用户进行交互，用户感知。
//从给定的 geecache.Group 对象中检索缓存数据，并通过 HTTP 接口将数据返回给客户端。

func startAPIServer(apiAddr string, gee *geecache.Group) { //apiAddr 服务器地址
	http.Handle("/api", http.HandlerFunc( //注册了一个处理函数，用于处理 /api 路径的 HTTP 请求
		func(w http.ResponseWriter, r *http.Request) {
			key := r.URL.Query().Get("key") //通过 r.URL.Query().Get("key") 获取请求 URL 中的查询参数 key
			view, err := gee.Get(key)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/octet-stream")
			w.Write(view.ByteSlice())
		}))
	log.Println("fontend server is running at", apiAddr)
	log.Fatal(http.ListenAndServe(apiAddr[7:], nil)) //apiAddr[7:] 去掉 http:// 前缀，以获得适当的地址格式。服务将在该地址运行，接受并处理 HTTP 请求
}

func main() {
	//需要命令行传入 port 和 api 2 个参数，用来在指定端口启动 HTTP 服务。
	var port int
	var api bool
	flag.IntVar(&port, "port", 8001, "Geecache server port")
	flag.BoolVar(&api, "api", false, "Start a api server?")
	flag.Parse()

	apiAddr := "http://localhost:9999"
	addrMap := map[int]string{
		8001: "http://localhost:8001",
		8002: "http://localhost:8002",
		8003: "http://localhost:8003",
	}
	var addrs []string
	for _, v := range addrMap {
		addrs = append(addrs, v)
	}

	gee := createGroup()
	if api {
		go startAPIServer(apiAddr, gee)
	}
	startCacheServer(addrMap[port], []string(addrs), gee)
}
