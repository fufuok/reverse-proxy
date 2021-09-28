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
	Host         string
	Listen       []*url.URL
	LAddr        []string
	BackendList  []string
	BackendMap   map[string]int
	Backend      map[string]*url.URL
	LBMode       int
	Certificate  tls.Certificate
}

func InitMain(c *TConfig) {
	conf = c
	initLogger()
}
