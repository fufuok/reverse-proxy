package rproxy

import (
	"context"
	"crypto/tls"
	"net/http"
	"net/http/httputil"
	"strings"

	"github.com/fufuok/bytespool"
	"github.com/fufuok/utils"
	"github.com/phuslu/log"

	"github.com/fufuok/balancer"
)

type tReverseProxy struct {
	// 后端服务负载均衡器
	LB balancer.Balancer

	// 后端服务代理实例
	BackendProxy map[string]*httputil.ReverseProxy

	// 修改请求 Host
	Host string
}

func NewReverseProxy() *tReverseProxy {
	p := &tReverseProxy{
		LB:   balancer.New(balancer.Mode(conf.LBMode), conf.BackendMap, conf.BackendList),
		Host: conf.Host,
	}

	// 使用字节数据缓冲池
	bufPool := bytespool.NewBufPool(32 * 1024)

	// 初始化后端代理实例
	p.BackendProxy = make(map[string]*httputil.ReverseProxy)
	for u, target := range conf.Backend {
		proxy := httputil.NewSingleHostReverseProxy(target)
		// 解决反代 HTTPS 时 x509: cannot validate certificate
		proxy.Transport = &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		}
		// 替换请求主机头
		isPortForwarding := strings.HasPrefix(target.Host, "0.0.0.0:")
		director := proxy.Director
		proxy.Director = func(r *http.Request) {
			director(r)
			if p.Host != "" {
				r.Host = p.Host
			} else if isPortForwarding {
				// 处理端口转发, 多域名访问时替换为正确 Host
				r.Host = utils.ReplaceHost(target.Host, r.Host)
			} else {
				r.Host = target.Host
			}
		}
		proxy.BufferPool = bufPool
		proxy.ErrorHandler = defaultErrorHandler
		proxy.ModifyResponse = defaultModifyResponse
		p.BackendProxy[u] = proxy
	}

	return p
}

func (p *tReverseProxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	rp := p.lb(r)
	rp.ServeHTTP(w, r)
}

// ListenAndServe 启动代理, 监听本地端口, HTTP
func (p *tReverseProxy) ListenAndServe(laddr string) error {
	return http.ListenAndServe(laddr, LimitMiddleware(p))
}

// ListenAndServeTLS 启动代理, 监听本地端口, HTTPS
func (p *tReverseProxy) ListenAndServeTLS(laddr string, cf tls.Certificate) error {
	s := &http.Server{
		Addr:    laddr,
		Handler: LimitMiddleware(p),
		TLSConfig: &tls.Config{
			Certificates: []tls.Certificate{cf},
		},
	}
	return s.ListenAndServeTLS("", "")
}

// 后端服务负载均衡
func (p *tReverseProxy) lb(r *http.Request) *httputil.ReverseProxy {
	u := p.LB.Select(r.RemoteAddr)
	svr := p.BackendProxy[u]

	r.Header.Set(OriginalHostHeader, r.Host)
	r.Header.Set(ProxyPassHeader, u)

	return svr
}

func defaultModifyResponse(resp *http.Response) error {
	originalHost := resp.Request.Header.Get(OriginalHostHeader)
	target := resp.Request.Header.Get(ProxyPassHeader)
	if conf.Debug {
		resp.Header.Set(OriginalHostHeader, originalHost)
		resp.Header.Set(ProxyPassHeader, target)
	}
	log.Info().
		Str("client_ip", resp.Request.RemoteAddr).
		Str("method", resp.Request.Method).
		Str("original_host", originalHost).
		Str("uri", resp.Request.RequestURI).
		Str("proxy_host", resp.Request.Host).
		Str("proxy_pass", target).
		Msg(resp.Status)
	return nil
}

func defaultErrorHandler(w http.ResponseWriter, _ *http.Request, err error) {
	// connection unexpectedly closed by client
	if err == context.Canceled {
		return
	}
	log.Error().Err(err).Msg("502 Bad Gateway")
	w.WriteHeader(http.StatusBadGateway)
}
