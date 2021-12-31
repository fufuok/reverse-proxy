package rproxy

import (
	"net/url"
	"sync"

	"github.com/fufuok/balancer"
	"github.com/phuslu/log"
)

const (
	// APPName 应用名称, 用于日志文件名
	APPName = "rproxy"

	// OriginalHostHeader 用户端请求的主机地址, 用于调试和日志
	OriginalHostHeader = "X-Original-Host"

	// ProxyPassHeader 头信息记录负载均衡选中的转发地址, 用于调试和日志
	ProxyPassHeader = "X-Proxy-Pass"

	// DefaultServer 默认接收指定域名外的所有请求, 类似 nginx server_name _;
	DefaultServer = ""
)

// Start 开启代理服务
func Start() {
	var wg sync.WaitGroup

	// 设置限流器
	limiter = NewRateLimiter()
	// 实例化反向代理
	rproxy := NewReverseProxy()

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

	log.Info().Strs("反向代理已启动:", conf.LAddr).Str("负载均衡:", balancer.Mode(conf.LBMode).String()).Msg("")

	for host, v := range conf.Backend {
		domain := conf.HostDomain
		if host != "" {
			domain = host
		}
		log.Info().Str("绑定域名:", host).Str("替换请求主机域名:", domain).Strs("转发到后端地址:", v.UrlList).Msg("")
	}

	if limiter != nil {
		log.Info().
			Int("限制每秒请求数:", conf.Limit).
			Int("最大突发请求数:", conf.Burst).
			Str("限流器:", limiter.Name()).
			Msg("")
	}

	wg.Wait()
}
