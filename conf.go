package rproxy

import (
	"crypto/tls"
	"net/url"
	"path/filepath"

	"github.com/fufuok/utils"
)

var (
	// RootPath 运行绝对路径
	RootPath = utils.ExecutableDir(true)

	// LogDir 日志路径
	LogDir        = filepath.Join(RootPath, "..", "log")
	LogFile       = filepath.Join(LogDir, APPName+".log")
	ErrorLogFile  = filepath.Join(LogDir, APPName+".error.log")
	DaemonLogFile = filepath.Join(LogDir, APPName+".daemon.log")

	conf *TConfig
)

type TConfig struct {
	Debug        bool
	LogLevel     string
	LogFile      string
	ErrorLogFile string

	// 全局指定的请求主机头域名, 可为空, 优先级低于为每个转发地址单独的指定
	HostDomain string

	Listen []*url.URL
	LAddr  []string

	// 按指定的请求主机头域名分类
	Backend map[string]*TBackend
	LBMode  int

	Certificate tls.Certificate

	LimitMode int
	Limit     int
	Burst     int
}

type TBackend struct {
	LBList  []string
	LBMap   map[string]int
	UrlList []string
	UrlHost map[string]*TUrlHost
}

type TUrlHost struct {
	// 转发的后端服务地址
	ProxyPass *url.URL

	// 转发时指定的请求主机头
	SpecifyHost string
}

func InitMain(c *TConfig) {
	conf = c
	initLogger()
}
