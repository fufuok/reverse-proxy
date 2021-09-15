# HTTP(s) Reverse Proxy (HTTP/HTTPS 反向代理)

## 特征

- 支持多个本地监听(HTTP/HTTPS)和多个后端服务(HTTP/HTTPS)
- 负载均衡: 平滑加权轮询
- 支持指定请求主机头: `Host`
- 需要 HTTPS 时可选指定证书和私钥, 不指定是使用内置证书

## 使用

`bin\rproxy.exe -h`

`./bin/rproxy -h`

```shell
NAME:
   HTTP(s) Reverse Proxy - HTTP/HTTPS 反向代理

USAGE:
   - 支持同时监听 HTTP/HTTPS, 指定或使用默认证书
   - 支持后端服务平滑加权轮询
   - 示例: ./rproxy -debug -L=:7777 -L=https://:555 -F=http://1.1.1.1:12345,5 -F=https://ff.cn

VERSION:
   v0.0.2.21091515

AUTHOR:
   Fufu <fufuok.com>

COMMANDS:
   help, h  Shows a list of commands or help for one command

GLOBAL OPTIONS:
   --debug               调试模式, 控制台输出日志 (default: false)
   --loglevel value      文件日志级别: debug, info, warn, error, fatal, panic (default: "info")
   --logfile value       日志文件位置 (default: "/mnt/e/Go/src/github.com/fufuok/reverse-proxy/log/rproxy.log")
   --errorlogfile value  错误级别的日志文件位置 (default: "/mnt/e/Go/src/github.com/fufuok/reverse-proxy/log/rproxy.error.log")
   --host value          指定请求主机头, 非80/443时带上端口, -host=fufuok.com:999
   --cert value          指定 HTTPS 服务端证书文件, 为空时使用内置证书
   --key value           指定 HTTPS 服务端私钥文件, 为空时使用内置私钥
   -L value              本地监听端口号, 默认 HTTP, 可多个, -L=127.0.0.1:123 -L=https://:555 (default: ":7777")
   -F value              后端服务地址, 可多个, -F=协议://地址:端口,权重值(可选), -F=http://fufu.cn:666,8
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
   0915 09:51:56 INF > 监听:=["http://:7777"] 反向代理服务已启动
   0915 09:51:56 INF > 后端:=["https://www.baidu.com"] 转发到后端服务地址
   0915 09:52:12 INF > client_ip="127.0.0.1:64874" method="GET" host="www.baidu.com" uri="/s?ie=utf-8&wd=xunyou" proxy_pass="https://www.baidu.com" 200 OK
   0915 09:52:15 INF > client_ip="127.0.0.1:64874" method="GET" host="www.baidu.com" uri="/sugrec?prod=..." proxy_pass="https://www.baidu.com" 200 OK
   ```
   
**注意: 转发目标服务是域名时, 域名解析的结果不要是自身 IP, 可能造成循环转发**
   
2. 可以多级转发, 假如上面的机器 IP 是 `192.168.1.100`, 现在这台是 `192.168.1.13`

   `./rproxy -debug -L=:555 -F=http://192.168.1.100:7777`

   浏览器访问: `http://192.168.1.13:555/s?ie=utf-8&wd=golang`

   **1.13** 控制台日志:

   ```shell
   ./rproxy -debug -L=:555 -F=http://192.168.1.100:7777
   0915 09:57:32 INF > client_ip="192.168.1.100:63125" method="GET" host="192.168.1.100:7777" uri="/s?ie=utf-8&wd=golang" proxy_pass="http://192.168.1.100:7777" 200 OK
   0915 09:57:34 INF > client_ip="192.168.1.100:63125" method="GET" host="192.168.1.100:7777" uri="/sugrec?prod=..." proxy_pass="http://192.168.1.100:7777" 200 OK
   0915 09:57:36 INF > client_ip="192.168.1.100:63125" method="GET" host="192.168.1.100:7777" uri="/favicon.ico" proxy_pass="http://192.168.1.100:7777" 200 OK
   ```
   
   **1.100** 控制台日志:

   ```shell
0915 09:57:31 INF > client_ip="192.168.1.13:48052" method="GET" host="www.baidu.com" uri="/s?ie=utf-8&wd=golang" proxy_pass="https://www.baidu.com" 200 OK
   0915 09:57:34 INF > client_ip="192.168.1.13:48054" method="GET" host="www.baidu.com" uri="/sugrec?prod=..." proxy_pass="https://www.baidu.com" 200 OK
   0915 09:57:35 INF > client_ip="192.168.1.13:48056" method="GET" host="www.baidu.com" uri="/favicon.ico" proxy_pass="https://www.baidu.com" 200 OK
   ```
   
3. 可以转发到多个后端服务

   1. 支持设定每个后端服务的权重, `-F=协议://地址:端口,权重值`

   2. 默认权重为 1, 全部为 1 或未设置时, 使用轮询; 有权重时使用平滑加权轮询

      浏览器访问 `http://127.0.0.1:888/s?ie=utf-8&wd=fufuok` 就可以看到转发地址按权重均衡了
      
      ```shell
      ./rproxy -debug -L=:888 -F=http://192.168.1.13:555 -F=http://192.168.1.100:7777,3
      0915 09:59:29 INF > 监听:=["http://:888"] 反向代理服务已启动
      0915 09:59:29 INF > 后端:=["http://192.168.1.13:555","http://192.168.1.100:7777"] 转发到后端服务地址
      0915 09:59:43 INF > client_ip="127.0.0.1:62919" method="GET" host="192.168.1.100:7777" uri="/s?ie=utf-8&wd=fufuok" proxy_pass="http://192.168.1.100:7777" 200 OK
      0915 09:59:45 INF > client_ip="127.0.0.1:62919" method="GET" host="192.168.1.13:555" uri="/sugrec?prod=..." proxy_pass="http://192.168.1.13:555" 200 OK
      0915 10:00:36 INF > client_ip="127.0.0.1:62919" method="GET" host="192.168.1.100:7777" uri="/" proxy_pass="http://192.168.1.100:7777" 200 OK
   0915 10:00:36 INF > client_ip="127.0.0.1:62919" method="GET" host="192.168.1.100:7777" uri="/sugrec?prod=..." proxy_pass="http://192.168.1.100:7777" 200 OK
      0915 10:00:36 INF > client_ip="127.0.0.1:59369" method="GET" host="192.168.1.100:7777" uri="/content-search.xml" proxy_pass="http://192.168.1.100:7777" 200 OK
      ```
      

4. 可以指定请求主机头, 一般转发到 IP 时都最好指定 Host 参数

   **注意: 非 80/443 端口时, 主机头需要加上端口:**

   `./rproxy -debug -F=http://1.1.1.1 -F=http://2.2.2.2 -host=orign.fufuok.com:9001`

   指定 Host 反代 HTTPS 示例:

   ```shell
   ./rproxy -debug -F=https://14.215.177.39,3 -F=https://14.215.177.38,2 -F=https://220.181.38.150 -host=www.baidu.com
   0915 10:02:03 INF > 监听:=["http://:7777"] 反向代理服务已启动
   0915 10:02:03 INF > 后端:=["https://14.215.177.39","https://14.215.177.38","https://220.181.38.150"] 转发到后端服务地址
   0915 10:02:03 INF > Host:="www.baidu.com" 请求时替换主机头
   0915 10:02:27 INF > client_ip="127.0.0.1:59224" method="GET" host="www.baidu.com" uri="/" proxy_pass="https://14.215.177.39" 200 OK
   0915 10:02:27 INF > client_ip="127.0.0.1:59224" method="GET" host="www.baidu.com" uri="/sugrec?prod=..." proxy_pass="https://14.215.177.38" 200 OK
   0915 10:02:27 INF > client_ip="127.0.0.1:63963" method="GET" host="www.baidu.com" uri="/content-search.xml" proxy_pass="https://14.215.177.39" 200 OK
   0915 10:02:48 INF > client_ip="127.0.0.1:63963" method="GET" host="www.baidu.com" uri="/sugrec?prod=..." proxy_pass="https://220.181.38.150" 200 OK
   ```

5. 也可以同时监听多个本地端口, 即多个 `-L=:nnn`

   `./rproxy -debug -L=:555 -L=:666 -L=:777 -F=http://192.168.1.100:7777`

6. 也可以同时监听 HTTP 和 HTTPS 多个本地端口

   `./rproxy -debug -L=https://:555 -L=http://127.0.0.1:666 -L=:777 -F=http://192.168.1.100:7777`

   ```shell
   ./rproxy -debug -L=:7777 -L=https://:555 -F=https://www.baidu.com
   0915 10:06:22 INF > 监听:=["http://:7777","https://:555"] 反向代理服务已启动
   0915 10:06:22 INF > 后端:=["https://www.baidu.com"] 转发到后端服务地址
   2021/09/15 10:06:44 http: TLS handshake error from 127.0.0.1:52488: remote error: tls: unknown certificate
   0915 10:06:45 INF > client_ip="127.0.0.1:64279" method="GET" host="www.baidu.com" uri="/" proxy_pass="https://www.baidu.com" 200 OK
   0915 10:06:45 INF > client_ip="127.0.0.1:64279" method="GET" host="www.baidu.com" uri="/sugrec?prod=..." proxy_pass="https://www.baidu.com" 200 OK
   0915 10:06:45 INF > client_ip="127.0.0.1:64279" method="GET" host="www.baidu.com" uri="/content-search.xml" proxy_pass="https://www.baidu.com" 200 OK
   ```

   浏览器访问 `https://127.0.0.1:555/` 就能打开百度, 控制台可以看到上面的日志

   **注意: 内置证书为自签发证书, 不受浏览器信任, 可安装 [cert](cert) 中的 `ca.crt` 到系统受信的根证书颁发机构, 或安装到浏览器解决**

7. 非调试模式时, 自动启动守护进程并后台运行, 日志记录到文件









*ff*