package main

import (
	"crypto/tls"
	_ "embed"
	"log"
	"net/url"
	"os"
	"strings"

	"github.com/fufuok/utils"
	"github.com/fufuok/utils/xdaemon"
	"github.com/urfave/cli/v2"

	rproxy "github.com/fufuok/reverse-proxy"
)

var (
	version = "v0.2.1.22090101"

	// 全局配置项
	conf = &rproxy.TConfig{}

	//go:embed server.crt
	defaultCert []byte

	//go:embed server.key
	defaultKey []byte
)

func main() {
	if err := newApp().Run(os.Args); err != nil {
		log.Fatalln(err, "\nbye.")
	}
}

func newApp() *cli.App {
	app := cli.NewApp()
	app.Name = "HTTP(s) Reverse Proxy"
	app.Usage = "HTTP/HTTPS 反向代理"
	app.UsageText = `- 支持同时监听 HTTP/HTTPS (指定或使用默认证书)
   - 支持后端服务负载均衡 (6 种模式)
   - 支持 HTTP/HTTPS 端口转发 (-F=http://0.0.0.0:88 请求 http://f.cn:7777, 实际返回 http://f.cn:88 的请求结果)
   - 简单: ./rproxy -debug -F=https://www.baidu.com
   - 综合: ./rproxy -debug -L=:7777 -L=https://:555 -F=http://1.2.3.4:666,5 -F=https://ff.cn -lb=3 -limit=30 -burst=50
   - 复用: ./rproxy -debug -L=:80 -F=http://1.1.1.1 -F=http://2.2.2.2:666*f.cn,2 -F=http://3.3.3.3:777*f.cn -F=http://4.4.4.4*x.cn`
	app.Version = version
	app.Copyright = "https://github.com/fufuok/reverse-proxy"
	app.Authors = []*cli.Author{
		{
			Name:  "Fufu",
			Email: "fufuok.com",
		},
	}
	app.Flags = appFlags()
	app.Action = appAction()
	return app
}

func appFlags() []cli.Flag {
	return []cli.Flag{
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
			Usage:       "指定全局的请求主机头域名, 端口会自动补齐, -host=fufuok.com",
			Destination: &conf.HostDomain,
		},
		&cli.StringFlag{
			Name:  "cert",
			Usage: "指定 HTTPS 服务端证书文件, 为空时使用内置证书",
		},
		&cli.StringFlag{
			Name:  "key",
			Usage: "指定 HTTPS 服务端私钥文件, 为空时使用内置私钥",
		},
		&cli.IntFlag{
			Name:        "limitmode",
			Usage:       "请求速率限制模式: 0 按请求 IP 限制(默认), 1 全局限制, 不分 IP",
			Destination: &conf.LimitMode,
		},
		&cli.IntFlag{
			Name:        "limit",
			Usage:       "限制每秒允许的请求数, 0 表示不限制(默认)",
			Destination: &conf.Limit,
		},
		&cli.IntFlag{
			Name:        "burst",
			Usage:       "允许的突发请求数, 如: -limit=30 -burst=50 (每秒 30 请求, 允许突发 50 请求)",
			Destination: &conf.Burst,
		},
		&cli.IntFlag{
			Name:        "lb",
			Usage:       "负载均衡: 0 加权轮询(默认), 1 平滑加权轮询, 2 加权随机, 3 IP哈希, 4 轮询, 5 随机",
			Destination: &conf.LBMode,
		},
		&cli.StringSliceFlag{
			Name:     "F",
			Usage:    "后端服务地址, 可多个, -F=协议://地址:端口*主机头域名(可选),权重值(可选), -F=http://fufu.cn:666,8",
			Required: true,
		},
		&cli.StringSliceFlag{
			Name:  "L",
			Value: cli.NewStringSlice(":7777"),
			Usage: "本地监听端口号, 默认 HTTP, 可多个, -L=127.0.0.1:123 -L=https://:555",
		},
	}
}

func appAction() func(c *cli.Context) error {
	return func(c *cli.Context) error {
		// 日志目录
		_ = os.MkdirAll(rproxy.LogDir, os.ModePerm)

		conf.Listen, conf.LAddr = parseListenAddr(c.StringSlice("L"))
		if len(conf.Listen) == 0 {
			_ = cli.ShowAppHelp(c)
			log.Fatalln("本地监听端口号有误\nbye.")
		}

		conf.Backend = parseBackend(c.StringSlice("F"))
		if len(conf.Backend) == 0 {
			_ = cli.ShowAppHelp(c)
			log.Fatalln("转发到后端服务地址列表有误\nbye.")
		}

		// 守护自身
		if !conf.Debug {
			xdaemon.NewDaemon(rproxy.DaemonLogFile).Run()
		}

		// 初始化证书
		conf.Certificate, _ = tls.X509KeyPair(defaultCert, defaultKey)
		cert := c.String("cert")
		key := c.String("key")
		if cert != "" && key != "" {
			cf, err := tls.LoadX509KeyPair(cert, key)
			if err != nil {
				_ = cli.ShowAppHelp(c)
				log.Fatalln("证书错误:", err, "\nbye.")
			}
			conf.Certificate = cf
		}

		// 初始化服务配置
		rproxy.InitMain(conf)

		// 启动服务
		rproxy.Start()

		return nil
	}
}

// 解析监听地址, 保留协议和主机头, 默认 HTTP, 去重
func parseListenAddr(ss []string) (listen []*url.URL, laddr []string) {
	for _, s := range ss {
		s = strings.TrimSpace(s)
		if s == "" || !strings.Contains(s, ":") {
			continue
		}

		var l *url.URL
		u, err := url.Parse(s)
		if err != nil || u.Host == "" || u.Port() == "" {
			l = &url.URL{
				Scheme: "http",
				Host:   s,
			}
		} else {
			l = &url.URL{
				Scheme: u.Scheme,
				Host:   u.Host,
			}
		}

		addr := l.Scheme + "://" + l.Host
		if !utils.InStrings(laddr, addr) {
			laddr = append(laddr, addr)
			listen = append(listen, l)
		}
	}
	return
}

// 解析转发的后端服务地址
func parseBackend(ss []string) map[string]*rproxy.TBackend {
	backend := make(map[string]*rproxy.TBackend)
	host := ""
	for _, s := range ss {
		// 相同的端口, 访问不同的域名会转发到对应的地址和端口, 简化的 nginx 多网站配置
		// -F=转发后端地址(ProxyPass)*指定主机头(HostDomain, 优先使用, 空表示默认服务),权重(Weighted)
		// -L=:8080 -F=http://1.2.3.4:666*a.cn,5 -F=http://2.3.4.5:777*a.cn -F=https://b.cn,2
		// -F=http://8.8.8.8:888 -host=c.cn
		// > 绑定域名:="" 替换请求主机域名:="c.cn" 转发到后端地址:=["https://b.cn","http://8.8.8.8:888"]
		// > 绑定域名:="a.cn" 替换请求主机域名:="a.cn" 转发到后端地址:=["http://1.2.3.4:666","http://2.3.4.5:777"]
		// 访问: http://a.cn:8080 会在指定 HostDomain 为 a.cn 使用 666 777 端口的 2 个后端负载均衡
		// 访问: http://x.cn:8080 会使用指定 HostDomain 为 c.cn 访问 888 或 b.cn
		// 若不加全局主机头域名 `-host=c.cn` 参数, 则除访问 a.cn 外, 其他请求转发时不替换 HostDomain
		x := strings.SplitN(s, ",", 2)
		y := strings.SplitN(x[0], "*", 2)
		svr := strings.TrimSpace(y[0])
		if svr == "" {
			continue
		}

		u, err := url.Parse(svr)
		if err != nil || u.Host == "" {
			continue
		}

		specifyHost := ""
		host = ""
		if rproxy.CheckPortForwarding(u.Host) {
			// 端口转发时自适应域名, 指定主机头无效
			conf.HostDomain = ""
		} else {
			// 优先使用转发地址指定的主机头域名(*host), 默认为全局指定的主机头域名(-host), 或空
			specifyHost = conf.HostDomain
			if len(y) == 2 && y[1] != "" {
				specifyHost = y[1]
				host = y[1]
			}
			// 补齐主机头端口
			if specifyHost != "" {
				specifyHost = utils.ReplaceHost(svr, specifyHost)
			}
		}

		w := 1
		if len(x) == 2 {
			w = utils.GetInt(x[1], 1)
		}

		us := u.String()
		f := joinString(host, us)
		if backend[host] == nil {
			backend[host] = &rproxy.TBackend{
				LBMap:   make(map[string]int),
				UrlHost: make(map[string]*rproxy.TUrlHost),
			}
		}
		backend[host].LBMap[f] = w
		backend[host].LBList = append(backend[host].LBList, f)
		backend[host].UrlList = append(backend[host].UrlList, us)
		backend[host].UrlHost[f] = &rproxy.TUrlHost{
			ProxyPass:   u,
			SpecifyHost: specifyHost,
		}
	}

	// 默认配置
	if _, ok := backend[rproxy.DefaultServer]; !ok {
		if backend[host] == nil {
			return nil
		}
		backend[rproxy.DefaultServer] = backend[host]
	}

	return backend
}

func joinString(a string, b ...string) string {
	return utils.AddString(a, "*!*", utils.AddString(b...))
}
