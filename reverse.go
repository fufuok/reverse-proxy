package rproxy

import (
	"context"
	"crypto/tls"
	"net/http"
	"net/http/httputil"

	"github.com/phuslu/log"

	"github.com/fufuok/balancer"
)

type tReverseProxy struct {
	// 后端服务负载均衡器
	LB balancer.Balancer

	// 修改返回值方法
	ModifyResponse func(*http.Response) error

	// 错误处理方法
	ErrorHandler func(http.ResponseWriter, *http.Request, error)

	// 修改请求 Host
	Host string
}

func NewReverseProxy() (proxy *tReverseProxy) {
	return &tReverseProxy{
		LB:   balancer.New(balancer.Mode(conf.LBMode), conf.BackendMap, conf.BackendList),
		Host: conf.Host,
	}
}

func (p *tReverseProxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// 后端服务负载均衡
	backend := p.LB.Select(r.RemoteAddr)
	target := conf.Backend[backend]

	if p.ModifyResponse == nil {
		p.ModifyResponse = p.defaultModifyResponse
	}

	if p.ErrorHandler == nil {
		p.ErrorHandler = p.defaultErrorHandler
	}

	proxy := httputil.NewSingleHostReverseProxy(target)

	// 解决反代 HTTPS 时 x509: cannot validate certificate
	proxy.Transport = &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	}

	// 替换请求主机头
	director := proxy.Director
	proxy.Director = func(r *http.Request) {
		director(r)
		r.Host = p.Host
		if r.Host == "" {
			r.Host = target.Host
		}
		r.Header.Set(ProxyPassHeader, target.String())
	}

	proxy.ModifyResponse = p.ModifyResponse
	proxy.ErrorHandler = p.ErrorHandler
	proxy.ServeHTTP(w, r)
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

func (p *tReverseProxy) defaultModifyResponse(r *http.Response) error {
	target := r.Request.Header.Get(ProxyPassHeader)
	if conf.Debug {
		r.Header.Set(ProxyPassHeader, target)
	}
	log.Info().
		Str("client_ip", r.Request.RemoteAddr).
		Str("method", r.Request.Method).
		Str("host", r.Request.Host).
		Str("uri", r.Request.RequestURI).
		Str("proxy_pass", target).
		Msg(r.Status)
	return nil
}

func (p *tReverseProxy) defaultErrorHandler(w http.ResponseWriter, _ *http.Request, err error) {
	// connection unexpectedly closed by client
	if err == context.Canceled {
		return
	}

	log.Error().Err(err).Msg("502 Bad Gateway")
	w.WriteHeader(http.StatusBadGateway)
}
