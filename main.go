//package main
//
//import (
//"crypto/tls"
//"flag"
//"fmt"
//"io"
//"log"
//"net"
//"net/http"
//	"sync"
//	"time"
//)
//func handleTunneling(w http.ResponseWriter, r *http.Request) {
//	fmt.Println("host is : ", r.Host)
//	//设置超时防止大量超时导致服务器资源不大量占用
//	dest_conn, err := net.DialTimeout("tcp", "localhost:5000", 10*time.Second)
//	if err != nil {
//		http.Error(w, err.Error(), http.StatusServiceUnavailable)
//		return
//	}
//	w.WriteHeader(http.StatusOK)
//	//类型转换
//	hijacker, ok := w.(http.Hijacker)
//	if !ok {
//		http.Error(w, "Hijacking not supported", http.StatusInternalServerError)
//		return
//	}
//	//接管连接
//	client_conn, _, err := hijacker.Hijack()
//	if err != nil {
//		http.Error(w, err.Error(), http.StatusServiceUnavailable)
//	}
//	go transfer(dest_conn, client_conn)
//	go transfer(client_conn, dest_conn)
//}
////转发连接的数据
//func transfer(destination io.WriteCloser, source io.ReadCloser) {
//	defer destination.Close()
//	defer source.Close()
//	io.Copy(destination, source)
//}
//func handleHTTP(w http.ResponseWriter, req *http.Request) {
//	//roudtrip 传递发送的请求返回响应的结果
//	fmt.Println("http host is : ", req.Host)
//	resp, err := http.DefaultTransport.RoundTrip(req)
//	if err != nil {
//		http.Error(w, err.Error(), http.StatusServiceUnavailable)
//		return
//	}
//	defer resp.Body.Close()
//	//把目标服务器的响应header复制
//	copyHeader(w.Header(), resp.Header)
//	w.WriteHeader(resp.StatusCode)
//	io.Copy(w, resp.Body)
//}
////复制响应头
//func copyHeader(dst, src http.Header) {
//	for k, vv := range src {
//		for _, v := range vv {
//			dst.Add(k, v)
//		}
//	}
//}
//func main() {
//	//证书路径
//	var pemPath string
//	flag.StringVar(&pemPath, "pem", "miproxy.crt", "path to pem file")
//	//私钥路径
//	var keyPath string
//	flag.StringVar(&keyPath, "key", "miproxy.key", "path to key file")
//	//协议
//	var proto string
//	flag.StringVar(&proto, "proto", "https", "Proxy protocol (http or https)")
//	flag.Parse()
//	//只支持http和https协议
//	if proto != "http" && proto != "https" {
//		log.Fatal("Protocol must be either http or https")
//	}
//	fmt.Println("l and s")
//	server := &http.Server{
//		Addr: ":8888",
//		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
//			fmt.Println("URL",r.URL.String(),r.Method)
//			if r.Method == http.MethodConnect {
//				//支持https websocket deng ... tcp
//				handleTunneling(w, r)
//			} else {
//				//直接http代理
//				handleHTTP(w, r)
//			}
//		}),
//		// 关闭http2
//		TLSNextProto: make(map[string]func(*http.Server, *tls.Conn, http.Handler)),
//	}
//
//	server2 := &http.Server{
//		Addr: ":8889",
//		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
//			fmt.Println("URL",r.URL.String(),r.Method)
//			if r.Method == http.MethodConnect {
//				//支持https websocket deng ... tcp
//				handleTunneling(w, r)
//			} else {
//				//直接http代理
//				handleHTTP(w, r)
//			}
//		}),
//		// 关闭http2
//		TLSNextProto: make(map[string]func(*http.Server, *tls.Conn, http.Handler)),
//	}
//
//	//log.Fatal(server.Serve())
//	//if proto == "http" {
//	var wg sync.WaitGroup
//	wg.Add(1)
//	go func() {
//		log.Fatal(server2.ListenAndServeTLS(pemPath, keyPath))
//	}()
//	go func(){log.Fatal(server.ListenAndServe())}()
//	//} else {
//	//}
//	wg.Wait()
//}


package main

import (
	"crypto/tls"
	"fmt"
	"github.com/ouqiang/goproxy"
	"io"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"sync"
	"time"
)


type handle struct {
	addrs []string
}

func copyHeader(dst, src http.Header) {
	for k, vv := range src {
		for _, v := range vv {
			dst.Add(k, v)
		}
	}
}

func (this *handle) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var remote *url.URL
	var err error
	if strings.Contains(r.Host, "reg.docker.alibaba-inc.com") {
			//roudtrip 传递发送的请求返回响应的结果
		//remote = r.URL

			resp, err := http.DefaultTransport.RoundTrip(r)
			if err != nil {
				http.Error(w, err.Error(), http.StatusServiceUnavailable)
				return
			}
			defer resp.Body.Close()
			//把目标服务器的响应header复制
			copyHeader(w.Header(), resp.Header)
			w.WriteHeader(resp.StatusCode)
			io.Copy(w, resp.Body)
			return
	} else {
		remote, err = url.Parse("http://" + "0.0.0.0:5000")
	}

	fmt.Printf("%+v", r)
	if err != nil {
		panic(err)
	}

	proxy := httputil.NewSingleHostReverseProxy(remote)
	proxy.ServeHTTP(w, r)
}

func startServer() {
	//被代理的服务器host和port
	h := &handle{}

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		srv := http.Server{
			Addr:    fmt.Sprintf("%s:%d", "0.0.0.0", 8888),
			Handler: h,
		}
		err := srv.ListenAndServe()
		log.Fatal(fmt.Sprintf("http error: %s", err))
	}()

	go func() {
		srv := http.Server {
			Addr:    fmt.Sprintf("%s:%d", "0.0.0.0", 8889),
			Handler: h,
			TLSConfig: &tls.Config{
				GetCertificate: func(info *tls.ClientHelloInfo) (*tls.Certificate, error) {
					//if cert, err := server.routeTable.certificates[info.ServerName]; !err {
					//	return nil, errors.New("cert not found")
					//} else {
					//	return cert, nil
					//}
					return nil, nil
				},
			},
		}
		err := srv.ListenAndServeTLS("", "")
		log.Fatal(fmt.Sprintf("https error: %s", err))
	}()

	wg.Wait()
}

func main() {


	var wg sync.WaitGroup
	proxy := goproxy.New()
	wg.Add(1)
	go func() {
		server := &http.Server{
			Addr:         ":8889",
			Handler:       proxy,
			ReadTimeout:  1 * time.Minute,
			WriteTimeout: 1 * time.Minute,
		}
		err := server.ListenAndServe()
		if err != nil {
			panic(err)
		}
	}()
	go func() {
		server := &http.Server{
			Addr:         ":8888",
			Handler:       proxy,
			ReadTimeout:  1 * time.Minute,
			WriteTimeout: 1 * time.Minute,
		}
		err := server.ListenAndServe()
		if err != nil {
			panic(err)
		}
	}()
	wg.Wait()
	//cnn, err := net.DialTimeout("tcp", "reg.docker.alibaba-inc.com:80", 5 * time.Second)
	//if err != nil {
	//
	//}
	//fmt.Println(cnn)
	startServer()
}