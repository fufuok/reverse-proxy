package main

import (
	"log"
	"net/url"
	"os"
	"strings"

	"github.com/fufuok/utils"
	"github.com/fufuok/utils/xdaemon"
	"github.com/urfave/cli/v2"

	"github.com/fufuok/reverse-proxy"
)

var conf = &rproxy.TConfig{}

func main() {
	app := cli.NewApp()
	app.Name = "HTTP Reverse Proxy"
	app.Usage = "HTTP 反向代理服务"
	app.UsageText = "- 支持后端服务平滑加权轮询\n   - 示例: " +
		"./rproxy -debug -L=:555 -F=http://1.1.1.1:12345,5 -F=https://ff.cn"
	app.Version = "v0.0.1.21090616"
	app.Copyright = "https://github.com/fufuok/reverse-proxy"
	app.Authors = []*cli.Author{
		{
			Name:  "Fufu",
			Email: "fufuok.com",
		},
	}
	app.Flags = []cli.Flag{
		&cli.BoolFlag{
			Name:        "debug",
			Usage:       "调试模式, 控制台输出日志",
			Destination: &conf.Debug,
		},
		&cli.StringFlag{
			Name:        "loglevel",
			Value:       "info",
			Usage:       "文件日志级别: debug, info, warn, error, fatal, panic",
			Destination: &conf.LogLevel,
		},
		&cli.StringFlag{
			Name:        "logfile",
			Value:       rproxy.LogFile,
			Usage:       "日志文件位置",
			Destination: &conf.LogFile,
		},
		&cli.StringFlag{
			Name:        "errorlogfile",
			Value:       rproxy.ErrorLogFile,
			Usage:       "错误级别的日志文件位置",
			Destination: &conf.ErrorLogFile,
		},
		&cli.StringFlag{
			Name:        "host",
			Usage:       "指定请求主机头, 非80/443时带上端口, -host=fufuok.com:999",
			Destination: &conf.Host,
		},
		&cli.StringSliceFlag{
			Name:     "L",
			Value:    cli.NewStringSlice(":7777"),
			Usage:    "本地监听端口号, 可多个, -L=127.0.0.1:123 -L=:555",
			Required: true,
		},
		&cli.StringSliceFlag{
			Name:     "F",
			Usage:    "后端服务地址, 可多个, -F=协议://地址:端口,权重值, -F=http://fufu.cn:666,8",
			Required: true,
		},
	}
	app.Action = func(c *cli.Context) error {
		// 日志目录
		_ = os.Mkdir(rproxy.LogDir, os.ModePerm)

		conf.Listen = parseListenAddr(c.StringSlice("L"))
		if len(conf.Listen) == 0 {
			_ = cli.ShowAppHelp(c)
			log.Fatalln("本地监听端口号有误\nbye.")
		}

		conf.Backend, conf.Forward = parseSWRR(c.StringSlice("F"))
		if len(conf.Backend) == 0 {
			_ = cli.ShowAppHelp(c)
			log.Fatalln("转发到后端服务地址列表有误\nbye.")
		}

		// 守护自身
		if !conf.Debug {
			xdaemon.NewDaemon(rproxy.DaemonLogFile).Run()
		}

		// 初始化服务配置
		rproxy.InitMain(conf)

		// 启动服务
		rproxy.Start()

		return nil
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatalln(err, "\nbye.")
	}
}

// 解析监听地址, 去重
func parseListenAddr(ss []string) (laddr []string) {
	for _, s := range ss {
		s = strings.TrimSpace(s)
		if s != "" && strings.Contains(s, ":") && !utils.InStrings(laddr, s) {
			laddr = append(laddr, s)
		}
	}
	return
}

// 解析平滑加权轮询结构体
func parseSWRR(ss []string) (swrr []*utils.TChoice, res []string) {
	for _, s := range ss {
		x := strings.SplitN(s, ",", 2)
		svr := strings.TrimSpace(x[0])
		if svr == "" {
			continue
		}

		w := 1
		if len(x) == 2 {
			w = utils.GetInt(x[1], 1)
		}

		if u, err := url.Parse(svr); err == nil && u.Host != "" {
			swrr = append(swrr, &utils.TChoice{
				Item:   u,
				Weight: w,
			})
			res = append(res, u.String())
		}
	}
	return
}
