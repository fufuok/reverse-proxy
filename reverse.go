package rproxy

import (
	"context"
	"crypto/tls"
	"errors"
	"net/http"
	"net/http/httputil"
	"net/url"
	"sync"

	"github.com/phuslu/log"

	"github.com/fufuok/utils"
)

type tReverseProxy struct {
	// 转发到后端服务地址列表
	backend []*utils.TChoice

	// 修改返回值方法
	ModifyResponse func(*http.Response) error

	// 错误处理方法
	ErrorHandler func(http.ResponseWriter, *http.Request, error)

	// 修改请求 Host
	Host string
}

func NewReverseProxy(backend []*utils.TChoice) (proxy *tReverseProxy, err error) {
	proxy = &tReverseProxy{}
	err = proxy.SetForward(backend)
	return
}

// SetForward 设置转发服务地址列表
func (p *tReverseProxy) SetForward(backend []*utils.TChoice) error {
	if len(backend) == 0 {
		return errors.New("后端服务地址有误")
	}

	p.backend = backend

	return nil
}

func (p *tReverseProxy) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	// 后端服务负载均衡
	target := p.balancer()

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
	proxy.ServeHTTP(rw, req)
}

// ListenAndServe 启动代理, 监听本地端口, HTTP
func (p *tReverseProxy) ListenAndServe(laddr string) error {
	return http.ListenAndServe(laddr, p)
}

// ListenAndServeTLS 启动代理, 监听本地端口, HTTPS
func (p *tReverseProxy) ListenAndServeTLS(laddr string, cf tls.Certificate) error {
	s := &http.Server{
		Addr:    laddr,
		Handler: p,
		TLSConfig: &tls.Config{
			Certificates: []tls.Certificate{cf},
		},
	}
	return s.ListenAndServeTLS("", "")
}

// 平滑加权轮询
func (p *tReverseProxy) balancer() (backend *url.URL) {
	return utils.SWRR(p.backend).Item.(*url.URL)
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

func (p *tReverseProxy) defaultErrorHandler(rw http.ResponseWriter, _ *http.Request, err error) {
	// connection unexpectedly closed by client
	if err == context.Canceled {
		return
	}

	log.Error().Err(err).Msg("502 Bad Gateway")
	rw.WriteHeader(http.StatusBadGateway)
}

// Start 开启代理服务
func Start() {
	var wg sync.WaitGroup

	rproxy, _ := NewReverseProxy(conf.Backend)
	rproxy.Host = conf.Host

	for _, l := range conf.Listen {
		wg.Add(1)
		go func(l *url.URL) {
			defer wg.Done()
			var err error
			if l.Scheme == "https" {
				err = rproxy.ListenAndServeTLS(l.Host, conf.Certificate)
			} else {
				err = rproxy.ListenAndServe(l.Host)
			}
			log.Fatal().Err(err).Msg("代理服务监听失败\nbye.")
		}(l)
	}

	log.Info().Strs("监听:", conf.LAddr).Msg("反向代理服务已启动")
	log.Info().Strs("后端:", conf.Forward).Msg("转发到后端服务地址")
	if conf.Host != "" {
		log.Info().Str("Host:", conf.Host).Msg("请求时替换主机头")
	}

	wg.Wait()
}
