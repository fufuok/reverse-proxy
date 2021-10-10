package rproxy

import (
	"context"
	"crypto/tls"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"

	"github.com/fufuok/utils"
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

func NewReverseProxy() *tReverseProxy {
	return &tReverseProxy{
		LB:             balancer.New(balancer.Mode(conf.LBMode), conf.BackendMap, conf.BackendList),
		ModifyResponse: defaultModifyResponse,
		ErrorHandler:   defaultErrorHandler,
		Host:           conf.Host,
	}
}

func (p *tReverseProxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	target := p.lb(r)
	r.Header.Set(ProxyPassHeader, target.String())

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

// 后端服务负载均衡
func (p *tReverseProxy) lb(r *http.Request) *url.URL {
	backend := p.LB.Select(r.RemoteAddr)
	svr := conf.Backend[backend]

	r.Header.Set(OriginalHostHeader, r.Host)
	r.Header.Set(ProxyBackendHeader, backend)

	// 处理 HTTP 端口转发
	if strings.HasPrefix(svr.Host, "0.0.0.0:") {
		return &url.URL{
			Scheme:      svr.Scheme,
			Opaque:      svr.Opaque,
			User:        svr.User,
			Host:        utils.ReplaceHost(svr.Host, r.Host),
			Path:        svr.Path,
			RawPath:     svr.RawPath,
			ForceQuery:  svr.ForceQuery,
			RawQuery:    svr.RawQuery,
			Fragment:    svr.Fragment,
			RawFragment: svr.RawFragment,
		}
	}

	return svr
}

func defaultModifyResponse(resp *http.Response) error {
	originalHost := resp.Request.Header.Get(OriginalHostHeader)
	backend := resp.Request.Header.Get(ProxyBackendHeader)
	target := resp.Request.Header.Get(ProxyPassHeader)
	if conf.Debug {
		resp.Header.Set(OriginalHostHeader, originalHost)
		resp.Header.Set(ProxyBackendHeader, backend)
		resp.Header.Set(ProxyPassHeader, target)
	}
	log.Info().
		Str("client_ip", resp.Request.RemoteAddr).
		Str("method", resp.Request.Method).
		Str("original_host", originalHost).
		Str("host", resp.Request.Host).
		Str("uri", resp.Request.RequestURI).
		Str("proxy_backend", backend).
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
