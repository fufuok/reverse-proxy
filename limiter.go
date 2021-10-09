package rproxy

import (
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/phuslu/log"
	"golang.org/x/time/rate"
)

type RateLimiter interface {
	Allow(key string) bool
	Name() string
}

type tGlobalRateLimiter struct {
	lim *rate.Limiter
}

type tIPRateLimiter struct {
	m map[string]*tLimiter
	r rate.Limit
	b int

	sync.Mutex
}

// 带最后访问时间
type tLimiter struct {
	lim      *rate.Limiter
	lastSeen time.Time
}

var (
	// 限流器: 全局限流或基于 IP 限流
	limiter RateLimiter

	// IP 限流键的最大空闲时间, 用于清除
	limitKeyLife = 5 * time.Minute
)

func newGlobalRateLimiter(r rate.Limit, b int) *tGlobalRateLimiter {
	return &tGlobalRateLimiter{
		lim: rate.NewLimiter(r, b),
	}
}

func newIPRateLimiter(r rate.Limit, b int) *tIPRateLimiter {
	i := &tIPRateLimiter{
		m: make(map[string]*tLimiter),
		r: r,
		b: b,
	}

	go i.cleanup()

	return i
}

func NewRateLimiter() RateLimiter {
	if conf.Limit <= 0 || conf.Burst <= 0 {
		return nil
	}

	limit := rate.Limit(conf.Limit)

	// 全局限流器
	if conf.LimitMode == 1 {
		return newGlobalRateLimiter(limit, conf.Burst)
	}

	// IP 限流器
	return newIPRateLimiter(limit, conf.Burst)
}

func (g *tGlobalRateLimiter) Allow(_ string) bool {
	return g.lim.Allow()
}

func (g *tGlobalRateLimiter) Name() string {
	return "GlobalRateLimiter"
}

func (i *tIPRateLimiter) Allow(key string) bool {
	return i.getLimiter(key).lim.Allow()
}

func (i *tIPRateLimiter) getLimiter(key string) *tLimiter {
	i.Lock()
	defer i.Unlock()

	if limiter, ok := i.m[key]; ok {
		limiter.lastSeen = time.Now()
		return limiter
	}

	limiter := &tLimiter{
		lim:      rate.NewLimiter(i.r, i.b),
		lastSeen: time.Now(),
	}
	i.m[key] = limiter
	return limiter
}

func (i *tIPRateLimiter) Name() string {
	return "IPRateLimiter"
}

// 清除旧数据项
func (i *tIPRateLimiter) cleanup() {
	for {
		time.Sleep(limitKeyLife / 3)
		now := time.Now().Add(-limitKeyLife)
		i.Lock()
		for k, v := range i.m {
			if v.lastSeen.Before(now) {
				delete(i.m, k)
			}
		}
		i.Unlock()
	}
}

func LimitMiddleware(next http.Handler) http.Handler {
	if limiter == nil {
		return next
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip, _, _ := net.SplitHostPort(r.RemoteAddr)
		if !limiter.Allow(ip) {
			log.Warn().
				Str("client_ip", r.RemoteAddr).
				Str("method", r.Method).
				Str("host", r.Host).
				Str("uri", r.RequestURI).
				Msg(http.StatusText(http.StatusTooManyRequests))
			http.Error(w, http.StatusText(http.StatusTooManyRequests), http.StatusTooManyRequests)
			return
		}

		next.ServeHTTP(w, r)
	})
}
