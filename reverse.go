package rproxy

import (
	"bytes"
	"context"
	"crypto/tls"
	"io"
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
	LoadBalancer map[string]balancer.Balancer

	// 后端服务代理实例
	BackendProxy map[string]*httputil.ReverseProxy
}

func NewReverseProxy() *tReverseProxy {
	lb := make(map[string]balancer.Balancer)
	for h, v := range conf.Backend {
		lb[h] = balancer.New(balancer.Mode(conf.LBMode), v.LBMap, v.LBList)
	}
	return &tReverseProxy{
		LoadBalancer: lb,
		BackendProxy: newBackendProxy(),
	}
}

// 初始化后端代理实例
func newBackendProxy() map[string]*httputil.ReverseProxy {
	// 使用字节数据缓冲池
	bsPool := bytespool.NewBufPool(32 * 1024)
	backendProxy := make(map[string]*httputil.ReverseProxy)
	for _, v := range conf.Backend {
		for f, backend := range v.UrlHost {
			proxy := httputil.NewSingleHostReverseProxy(backend.ProxyPass)
			proxy.ErrorLog = rproxyLogger
			// 解决反代 HTTPS 时 x509: cannot validate certificate
			proxy.Transport = &http.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: true,
				},
			}
			// 指定请求主机头
			specifyHost := backend.SpecifyHost
			urlHost := backend.ProxyPass.Host
			target := backend.ProxyPass.String()
			// 替换请求主机头
			isPortForwarding := CheckPortForwarding(urlHost)
			director := proxy.Director
			proxy.Director = func(r *http.Request) {
				r.Header.Set(OriginalHostHeader, r.Host)
				r.Header.Set(ProxyPassHeader, target)
				director(r)
				if isPortForwarding {
					// 处理端口转发, 多域名访问时替换为正确 HostDomain
					r.Host = utils.ReplaceHost(urlHost, r.Host)
				} else if specifyHost != "" {
					r.Host = specifyHost
				} else {
					r.Host = urlHost
				}
				if conf.Debug {
					var body []byte
					if r.ContentLength > 0 {
						body, _ = io.ReadAll(r.Body)
						r.Body = io.NopCloser(bytes.NewReader(body))
					}
					log.Debug().Str("url", r.URL.String()).Bytes("body", body).Msg("Request")
				}
			}
			proxy.BufferPool = bsPool
			proxy.ErrorHandler = defaultErrorHandler
			proxy.ModifyResponse = defaultModifyResponse
			backendProxy[f] = proxy
		}
	}
	return backendProxy
}

func (p *tReverseProxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	rp := p.lb(r)
	if rp == nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
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
	host, _ := utils.SplitHostPort(r.Host)
	lb, ok := p.LoadBalancer[host]
	if !ok {
		lb, ok = p.LoadBalancer[DefaultServer]
		if !ok {
			return nil
		}
	}
	f := lb.Select(r.RemoteAddr)
	svr := p.BackendProxy[f]
	return svr
}

func defaultModifyResponse(resp *http.Response) error {
	originalHost := resp.Request.Header.Get(OriginalHostHeader)
	target := resp.Request.Header.Get(ProxyPassHeader)
	if conf.Debug {
		resp.Header.Set(OriginalHostHeader, originalHost)
		resp.Header.Set(ProxyPassHeader, target)
		var body []byte
		if resp.ContentLength > 0 {
			body, _ = io.ReadAll(resp.Body)
			resp.Body = io.NopCloser(bytes.NewReader(body))
		}
		log.Debug().Str("url", resp.Request.URL.String()).Bytes("body", body).Msg("Response")
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

func CheckPortForwarding(host string) bool {
	return strings.HasPrefix(host, "0.0.0.0:")
}
