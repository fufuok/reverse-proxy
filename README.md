# HTTP(s) Reverse Proxy (HTTP/HTTPS 反向代理)

## 特征

- 支持多个本地监听(HTTP/HTTPS)和多个后端服务(HTTP/HTTPS)
- 支持指定负载均衡: 
  - `-lb=0` `WeightedRoundRobin` 加权轮询(默认) 
  - `-lb=1` `SmoothWeightedRoundRobin` 平滑加权轮询
  - `-lb=2` `WeightedRand` 加权随机
  - `-lb=3` `ConsistentHash` IP一致性哈希
  - `-lb=4` `RoundRobin` 轮询
  - `-lb=5` `Random` 随机
  - [github.com/fufuok/balancer: Goroutine-safe, High-performance general load balancing algorithm library.](https://github.com/fufuok/balancer)
- 支持指定请求主机头: `Host`
- 支持按 IP 限流或全局限流:
  - `-limitmode=1` 全局限流, 不分 IP, 默认为按 IP 分别限流
  - `-limit=30` 限制每秒 30 个请求
  - `-burst=50` 允许突发 50 个请求
- 支持 HTTP/HTTPS 端口转发, 自适应 `Host`, 示例: `-F=协议://0.0.0.0:端口`
- 需要 HTTPS 时可选指定证书和私钥, 不指定时使用内置证书

## 使用

目录:

```
.
├── bin
│   └── rproxy
├── log
│   ├── rproxy.2021-09-15T12-00-00.log
│   ├── rproxy.daemon.log
│   └── rproxy.log -> rproxy.2021-09-15T12-00-00.log
```

运行:

`bin\rproxy.exe -h`

`./bin/rproxy -h`

```shell
NAME:
   HTTP(s) Reverse Proxy - HTTP/HTTPS 反向代理

USAGE:
   - 支持同时监听 HTTP/HTTPS (指定或使用默认证书)
   - 支持后端服务负载均衡 (6 种模式)
   - 支持 HTTP/HTTPS 端口转发 (-F=http://0.0.0.0:88 请求 http://f.cn:7777, 实际返回 http://f.cn:88 的请求结果)
   - 简单: ./rproxy -debug -F=https://www.baidu.com
   - 综合: ./rproxy -debug -L=:7777 -L=https://:555 -F=http://1.2.3.4:666,5 -F=https://ff.cn -lb=3 -limit=30 -burst=50

VERSION:
   v0.1.0.21121111

AUTHOR:
   Fufu <fufuok.com>

COMMANDS:
   help, h  Shows a list of commands or help for one command

GLOBAL OPTIONS:
   --debug               调试模式, 控制台输出日志 (default: false)
   --loglevel value      文件日志级别: debug, info, warn, error, fatal, panic (default: "info")
   --logfile value       日志文件位置 (default: "程序位置/../log/rproxy.log")
   --errorlogfile value  错误级别的日志文件位置 (default: "程序位置/../log/rproxy.error.log")
   --host value          指定请求主机头, 非80/443时带上端口, -host=fufuok.com:999
   --cert value          指定 HTTPS 服务端证书文件, 为空时使用内置证书
   --key value           指定 HTTPS 服务端私钥文件, 为空时使用内置私钥
   --limitmode value     请求速率限制模式: 0 按请求 IP 限制(默认), 1 全局限制, 不分 IP (default: 0)
   --limit value         限制每秒允许的请求数, 0 表示不限制(默认) (default: 0)
   --burst value         允许的突发请求数, 如: -limit=30 -burst=50 (每秒 30 请求, 允许突发 50 请求) (default: 0)
   --lb value            负载均衡: 0 加权轮询(默认), 1 平滑加权轮询, 2 加权随机, 3 IP哈希, 4 轮询, 5 随机 (default: 0)
   -F value              后端服务地址, 可多个, -F=协议://地址:端口,权重值(可选), -F=http://fufu.cn:666,8
   -L value              本地监听端口号, 默认 HTTP, 可多个, -L=127.0.0.1:123 -L=https://:555 (default: ":7777")
   --help, -h            show help (default: false)
   --version, -v         print the version (default: false)

COPYRIGHT:
   https://github.com/fufuok/reverse-proxy
```

## 示例

1. 转发请求到百度:

   `./rproxy -debug -F=https://www.baidu.com`

   浏览器访问则显示百度结果: `http://127.0.0.1:7777/s?ie=utf-8&wd=xunyou`

   控制台日志:

   ```shell
   ./rproxy -debug -F=https://www.baidu.com
   1209 14:03:54 INF > 反向代理已启动:=["http://:7777"]
   1209 14:03:54 INF > 转发到后端地址:=["https://www.baidu.com"] 负载均衡:="WeightedRoundRobin"
   1209 14:04:14 INF > client_ip="127.0.0.1:63983" method="GET" original_host="127.0.0.1:7777" uri="/s?ie=utf-8&wd=xunyou" proxy_host="www.baidu.com" proxy_pass="https://www.baidu.com" 200 OK
   1209 14:04:16 INF > client_ip="127.0.0.1:63983" method="GET" original_host="127.0.0.1:7777" uri="/sugrec?prod=..." proxy_host="www.baidu.com" proxy_pass="https://www.baidu.com" 200 OK
   ```
   

**注意: 转发目标服务是域名时, 域名解析的结果不要是自身 IP, 可能造成循环转发**

2. 可以多级转发, 假如上面的机器 IP 是 `192.168.1.100`, 现在这台是 `192.168.1.13`

   `./rproxy -debug -L=:555 -F=http://192.168.1.100:7777`

   浏览器访问: `http://192.168.1.13:555/s?ie=utf-8&wd=golang`

   **1.13** 控制台日志:

   ```shell
   ./rproxy -debug -L=:555 -F=http://192.168.1.100:7777
   1209 14:06:49 INF > 反向代理已启动:=["http://:555"] 
   1209 14:06:49 INF > 转发到后端地址:=["http://192.168.1.100:7777"] 负载均衡:="WeightedRoundRobin" 
   1209 14:07:03 INF > client_ip="192.168.1.100:64044" method="GET" original_host="192.168.1.13:555" uri="/s?ie=utf-8&wd=golang" proxy_host="192.168.1.100:7777" proxy_pass="http://192.168.1.100:7777" 200 OK
   1209 14:07:04 INF > client_ip="192.168.1.100:64044" method="GET" original_host="192.168.1.13:555" uri="/sugrec?prod..." proxy_host="192.168.1.100:7777" proxy_pass="http://192.168.1.100:7777" 200 OK
   1209 14:07:04 INF > client_ip="192.168.1.100:64044" method="GET" original_host="192.168.1.13:555" uri="/favicon.ico" proxy_host="192.168.1.100:7777" proxy_pass="http://192.168.1.100:7777" 200 OK
   ```

   **1.100** 控制台日志:
   
   ```shell
   1209 14:07:03 INF > client_ip="192.168.1.13:60744" method="GET" original_host="192.168.1.100:7777" uri="/s?ie=utf-8&wd=golang" proxy_host="www.baidu.com" proxy_pass="https://www.baidu.com" 200 OK
   1209 14:07:04 INF > client_ip="192.168.1.13:60744" method="GET" original_host="192.168.1.100:7777" uri="/sugrec?prod=..." proxy_host="www.baidu.com" proxy_pass="https://www.baidu.com" 200 OK
   1209 14:07:04 INF > client_ip="192.168.1.13:60744" method="GET" original_host="192.168.1.100:7777" uri="/favicon.ico" proxy_host="www.baidu.com" proxy_pass="https://www.baidu.com" 200 OK
   ```
   
3. 可以转发到多个后端服务

   1. 支持设定每个后端服务的权重, `-F=协议://地址:端口,权重值`

   2. 默认权重为 1, 全部为 1 或未设置时, 使用轮询; 有权重时使用平滑加权轮询

      浏览器访问 `http://127.0.0.1:888/s?ie=utf-8&wd=fufuok` 就可以看到转发地址按权重均衡了
      
      ```shell
      ./rproxy -debug -L=:888 -F=http://192.168.1.13:555 -F=http://192.168.1.100:7777,2
      1209 14:13:08 INF > 反向代理已启动:=["http://:888"]
      1209 14:13:08 INF > 转发到后端地址:=["http://192.168.1.13:555","http://192.168.1.100:7777"] 负载均衡:="WeightedRoundRobin"
      1209 14:13:17 INF > client_ip="[::1]:64158" method="GET" original_host="127.0.0.1:888" uri="/s?ie=utf-8&wd=fufuok" proxy_host="192.168.1.100:7777" proxy_pass="http://192.168.1.100:7777" 200 OK
      1209 14:13:17 INF > client_ip="[::1]:64158" method="GET" original_host="127.0.0.1:888" uri="/sugrec?prod=..." proxy_host="192.168.1.100:7777" proxy_pass="http://192.168.1.13:555" 200 OK
      1209 14:13:40 INF > client_ip="[::1]:64158" method="GET" original_host="127.0.0.1:888" uri="/sugrec?pre=..." proxy_host="192.168.1.100:7777" proxy_pass="http://192.168.1.100:7777" 200 OK
      ```
      
   
4. 可以指定请求主机头, 一般转发到 IP 时都最好指定 Host 参数

   **注意: 非 80/443 端口时, 主机头需要加上端口:**

   `./rproxy -debug -F=http://1.1.1.1 -F=http://2.2.2.2 -host=orign.fufuok.com:9001`

   指定 Host 反代 HTTPS 示例:

   ```shell
   ./rproxy -debug -F=https://14.215.177.39,3 -F=https://14.215.177.38,2 -F=https://220.181.38.150 -host=www.baidu.com
   1209 13:35:35 INF > 反向代理已启动:=["http://:7777"]
   1209 13:35:35 INF > 转发到后端地址:=["https://14.215.177.39","https://14.215.177.38","https://220.181.38.150"] 负载均衡:="WeightedRoundRobin"
   1209 13:35:35 INF > Host:="www.baidu.com" 请求时替换主机头
   1209 13:35:44 INF > client_ip="127.0.0.1:52429" method="GET" original_host="127.0.0.1:7777" uri="/" proxy_host="www.baidu.com" proxy_pass="https://14.215.177.39" 200 OK
   1209 13:35:44 INF > client_ip="127.0.0.1:52429" method="GET" original_host="127.0.0.1:7777" uri="/sugrec?prod=..." proxy_host="www.baidu.com" proxy_pass="https://14.215.177.39" 200 OK
   1209 13:35:45 INF > client_ip="127.0.0.1:52429" method="GET" original_host="127.0.0.1:7777" uri="/content-search.xml" proxy_host="www.baidu.com" proxy_pass="https://14.215.177.38" 200 OK
   1209 13:35:47 INF > client_ip="127.0.0.1:52429" method="GET" original_host="127.0.0.1:7777" uri="/sugrec?prod=..." proxy_host="www.baidu.com" proxy_pass="https://14.215.177.39" 200 OK
   1209 13:35:52 INF > client_ip="127.0.0.1:52429" method="GET" original_host="127.0.0.1:7777" uri="/sugrec?prod=..." proxy_host="www.baidu.com" proxy_pass="https://14.215.177.38" 200 OK
   1209 13:35:54 INF > client_ip="127.0.0.1:52429" method="GET" original_host="127.0.0.1:7777" uri="/sugrec?pre=..." proxy_host="www.baidu.com" proxy_pass="https://220.181.38.150" 200 OK
   ```

5. 也可以同时监听多个本地端口, 即多个 `-L=:nnn`

   `./rproxy -debug -L=:555 -L=:666 -L=:777 -F=http://192.168.1.100:7777`

6. 也可以同时监听 HTTP 和 HTTPS 多个本地端口

   `./rproxy -debug -L=https://:555 -L=http://127.0.0.1:666 -L=:777 -F=http://192.168.1.100:7777`

   ```shell
   ./rproxy -debug -L=:7777 -L=https://:555 -F=https://www.baidu.com
   1209 14:16:04 INF > 反向代理已启动:=["http://:7777","https://:555"]
   1209 14:16:04 INF > 转发到后端地址:=["https://www.baidu.com"] 负载均衡:="WeightedRoundRobin"
   1209 14:16:18 INF > client_ip="127.0.0.1:64215" method="GET" original_host="127.0.0.1:555" uri="/" proxy_host="www.baidu.com" proxy_pass="https://www.baidu.com" 200 OK
   1209 14:16:18 INF > client_ip="127.0.0.1:64215" method="GET" original_host="127.0.0.1:555" uri="/sugrec?prod=..." proxy_host="www.baidu.com" proxy_pass="https://www.baidu.com" 200 OK
   1209 14:16:18 INF > client_ip="127.0.0.1:64215" method="GET" original_host="127.0.0.1:555" uri="/content-search.xml" proxy_host="www.baidu.com" proxy_pass="https://www.baidu.com" 200 OK
   ```

   浏览器访问 `https://127.0.0.1:555/` 就能打开百度, 控制台可以看到上面的日志

   **注意: **

   - 内置证书为自签发证书, 不受浏览器信任, 可安装 [cert](cert) 中的 `ca.crt` 到系统受信的根证书颁发机构, 或安装到浏览器解决
   - 测试网址写入 HOSTS `127.0.0.1 test.dev.ops`, 使用 `https://test.dev.ops:555` 即可打开百度, 浏览器显示绿锁证书

7. 负载均衡默认为高性能平滑加权重轮询 (WRR)

   可以用 `-lb=3` 来指定一致性哈希算法, 相同客户端 IP 的所有请求均会转发到同一后端服务

   `./rproxy -debug -F=http://1.1.1.1:9001 -F=http://1.1.1.1:9002 -F=http://1.1.1.2 -host=ff.cn -lb=3`

8. 默认不限流转发, `-limit` `-burst` 都大于 0 时会启动限流

   按请求 IP 限制每秒 10 个请求, 允许突发请求 20 个:

   `./rproxy -debug -F=https://www.baidu.com -limit=10 -burst=20`

   全局限流, 限制每秒 10000 个请求, 允许突发请求 15000 个:

   `./rproxy -debug -F=https://www.baidu.com -limitmode=1 -limit=10000 -burst=15000`

   ```shell
   1009 11:11:19 INF > 反向代理已启动:=["http://:7777"] 
   1009 11:11:19 INF > 转发到后端地址:=["https://www.baidu.com"] 负载均衡:="WeightedRoundRobin" 
   1009 11:11:19 INF > 限制每秒请求数:=10000 最大突发请求数:=15000 限流器:="GlobalRateLimiter" 
   ```

9. 多域名多服务时可以使用端口转发, 简单示例:

   本地提供了不同域名的端口服务: `http://a.cn:777` `http://b.cn:777`, 对外提供统一服务: `http://相应域名:对外服务端口`, 如对外提供访问: `http://a.cn:666` 

   ```shell
   ./rproxy -debug -L=:666 -F=http://0.0.0.0:777
   1010 18:18:56 INF > 反向代理已启动:=["http://:666"] 
   1010 18:18:56 INF > 转发到后端地址:=["http://0.0.0.0:777"] 负载均衡:="WeightedRoundRobin" 
   1010 18:19:12 INF > client_ip="127.0.0.1:49411" method="GET" original_host="ff.php:666" uri="/" proxy_host="ff.php:777" proxy_pass="http://0.0.0.0:777" 200 OK
   1010 18:19:12 INF > client_ip="127.0.0.1:49411" method="GET" original_host="ff.php:666" uri="/v/css/ff.css" proxy_host="ff.php:777" proxy_pass="http://0.0.0.0:777" 200 OK
   1010 18:19:12 INF > client_ip="127.0.0.1:49411" method="GET" original_host="ff.php:666" uri="/favicon.ico" proxy_host="ff.php:777" proxy_pass="http://0.0.0.0:777" 404 Not Found
   1010 18:19:26 INF > client_ip="127.0.0.1:52804" method="POST" original_host="xy.oa:666" uri="/process/save/" proxy_host="xy.oa:777" proxy_pass="http://0.0.0.0:777" 200 OK
   1010 18:19:28 INF > client_ip="127.0.0.1:54262" method="GET" original_host="xy.uni:666" uri="/" proxy_host="xy.uni:777" proxy_pass="http://0.0.0.0:777" 302 Found
   1010 18:19:28 INF > client_ip="127.0.0.1:54262" method="GET" original_host="xy.uni:666" uri="/login/" proxy_host="xy.uni:777" proxy_pass="http://0.0.0.0:777" 200 OK
   ```

   代理会自动处理域名, 用对应的域名和转发端口去请求真实内容

   当然, 如果某域名对应的服务在其他服务器, 可以写个本地 HOSTS 让反向代理自身通过域名能访问到指定服务

10. 非调试模式时, 自动启动守护进程并后台运行, 日志记录会到文件









*ff*