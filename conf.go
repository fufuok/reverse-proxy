package rproxy

import (
	"path/filepath"

	"github.com/fufuok/utils"
)

const (
	// APPName 应用名称, 用于日志文件名
	APPName = "rproxy"

	// ProxyPassHeader 头信息记录请求的后端地址, 用于调试和日志
	ProxyPassHeader = "X-Proxy-Pass"
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
	Host         string
	Listen       []string
	Forward      []string
	Backend      []*utils.TChoice
}

func InitMain(c *TConfig) {
	conf = c
	initLogger()
}
