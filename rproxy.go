package rproxy

import (
	"net/url"
	"sync"

	"github.com/phuslu/log"
)

const (
	// APPName 应用名称, 用于日志文件名
	APPName = "rproxy"

	// ProxyPassHeader 头信息记录请求的后端地址, 用于调试和日志
	ProxyPassHeader = "X-Proxy-Pass"
)

// Start 开启代理服务
func Start() {
	var wg sync.WaitGroup

	limiter = NewRateLimiter()
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

	log.Info().Strs("反向代理已启动:", conf.LAddr).Msg("")
	log.Info().Strs("转发到后端地址:", conf.BackendList).Str("负载均衡:", rproxy.LB.Name()).Msg("")
	if conf.Host != "" {
		log.Info().Str("替换请求主机头:", conf.Host).Msg("")
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
