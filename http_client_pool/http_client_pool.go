package http_client_pool

import (
	"crypto/tls"
	"fmt"
	"golang.org/x/net/proxy"
	"log"
	"net"
	"net/http"
	"net/url"
	"sync/atomic"
	"time"
)

type Proxy struct {
	Server string
	Port   int
	User   string
	Pass   string
}

type ClientPool struct {
	count      uint64
	index      uint64
	oneclients []*http.Client
}

//创建client池
//参数：代理、是否重定向、超时时间（s）、会在多少并发协程下使用
func NewClientPool(pxs []Proxy, isRedirect bool, timeout, threads uint) *ClientPool {
	length := 5 //默认5个client
	if threads/100 > 5 {
		length = int(threads) / 100
		if int(threads)%100 > 0 {
			length++
		}
	}
	count := len(pxs)
	var httpTransports = make([]http.RoundTripper, 0, length)
	if count == 0 {
		count = length //默认client池长度为5
		for i := 0; i < count; i++ {
			var httpTransport = &http.Transport{
				TLSClientConfig:     &tls.Config{InsecureSkipVerify: true}, //跳过HTTPS证书检查
				MaxIdleConns:        100,                                   //MaxIdleConns限制了最大keep-alive的连接数，超出的连接会被关闭掉
				MaxIdleConnsPerHost: 1000,                                  //最大空闲连接的主机数
				IdleConnTimeout:     45 * time.Second,                      //空闲连接超时时间
				Dial: (&net.Dialer{
					Timeout:   10 * time.Second, //TCP连接建立超时时间
					KeepAlive: 30 * time.Second,
				}).Dial,
				TLSHandshakeTimeout:   10 * time.Second, //TLS握手时间
				ResponseHeaderTimeout: 10 * time.Second, //头部返回超时时间
				// DisableCompression: true,
			}
			httpTransports = append(httpTransports, httpTransport)
		}
	} else {
		if count < length {
			times := length / count
			if length%count > 0 {
				times++
			}
			original_proxy := pxs
			for i := 0; i < times; i++ {
				pxs = append(pxs, original_proxy...)
			}
			count = len(pxs)
		}
		for i := range pxs {
			var p func(*http.Request) (*url.URL, error)
			var d = (&net.Dialer{
				Timeout:   10 * time.Second, //TCP连接建立超时时间
				KeepAlive: 30 * time.Second,
			}).Dial
			if pxs[i].User == "" {
				p = func(_ *http.Request) (*url.URL, error) {
					return url.Parse(fmt.Sprintf("http://%s:%d", pxs[i].Server, pxs[i].Port))
				}
			} else {
				var auth proxy.Auth
				auth.User = pxs[i].User
				auth.Password = pxs[i].Pass
				dialer, err := proxy.SOCKS5("tcp", fmt.Sprintf("%s:%d", pxs[i].Server, pxs[i].Port), &auth, proxy.Direct)
				if err != nil {
					log.Printf("启用代理服务器失败: %s:%d\n", pxs[i].Server, pxs[i].Port)
				} else {
					log.Printf("启用代理服务器成功: %s:%d\n", pxs[i].Server, pxs[i].Port)
					d = dialer.Dial
				}
			}

			var httpTransport = &http.Transport{
				TLSClientConfig:       &tls.Config{InsecureSkipVerify: true}, //跳过HTTPS证书检查
				MaxIdleConns:          100,                                   //MaxIdleConns限制了最大keep-alive的连接数，超出的连接会被关闭掉
				MaxIdleConnsPerHost:   1000,                                  //最大空闲连接的主机数
				IdleConnTimeout:       45 * time.Second,                      //空闲连接超时时间
				Dial:                  d,
				TLSHandshakeTimeout:   10 * time.Second, //TLS握手时间
				ResponseHeaderTimeout: 10 * time.Second, //头部返回超时时间
				Proxy:                 p,
				// DisableCompression: true,
			}
			httpTransports = append(httpTransports, httpTransport)
		}
	}

	var redirectPolicyFunc func(req *http.Request, via []*http.Request) error
	if isRedirect {
		redirectPolicyFunc = nil //使用默认，最多重定向10次
	} else {
		redirectPolicyFunc = func(_ *http.Request, _ []*http.Request) error {
			return http.ErrUseLastResponse
		}
	}

	pool := &ClientPool{
		count:      uint64(count),
		oneclients: make([]*http.Client, count),
	}
	for i := 0; i < count; i++ {
		oneclient := &http.Client{
			CheckRedirect: redirectPolicyFunc,
			Transport:     httpTransports[i],
			//设置请求绝对超时时间
			Timeout: time.Duration(timeout) * time.Second,
		}
		pool.oneclients[i] = oneclient
	}
	return pool
}

//获取client
func (p *ClientPool) Get() *http.Client {
	i := atomic.AddUint64(&p.index, 1)
	picked := int(i % p.count)
	return p.oneclients[picked]
}

//关闭client池
func (p *ClientPool) Close() {
	for _, c := range p.oneclients {
		c.CloseIdleConnections()
	}
	p.oneclients = nil
}
