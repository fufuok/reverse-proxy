# HTTP Reverse Proxy (HTTP/HTTPS 反向代理)

## 特征

- 支持多个本地监听(HTTP)和多个后端服务(HTTP/HTTPS)
- 负载均衡: 平滑加权轮询
- 支持指定请求主机头: `Host`

## 使用

`bin\rproxy.exe -h`

`./bin/rproxy -h`

```shell
NAME:
   HTTP Reverse Proxy - HTTP 反向代理服务

USAGE:
   - 支持后端服务平滑加权轮询
   - 示例: ./rproxy -debug -L=:555 -F=http://1.1.1.1:12345,5 -F=https://ff.cn

VERSION:
   v0.0.1.21090616

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
   -L value              本地监听端口号, 可多个, -L=127.0.0.1:123 -L=:555 (default: ":7777")
   -F value              后端服务地址, 可多个, -F=协议://地址:端口,权重值, -F=http://fufu.cn:666,8
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
   0907 16:32:10 INF > 监听:=[":7777"] 反向代理服务已启动
   0907 16:32:10 INF > 后端:=["https://www.baidu.com"] 转发到后端服务地址
   0907 16:32:29 INF > client_ip="127.0.0.1:60039" request_uri="/s?ie=utf-8&wd=xunyou" proxy_pass="https://www.baidu.com" 200 OK
   0907 16:32:32 INF > client_ip="127.0.0.1:60039" request_uri="/sugrec?..." proxy_pass="https://www.baidu.com" 200 OK
   0907 13:14:40 INF > client_ip="127.0.0.1:64970" request_uri="/favicon.ico" proxy_pass="https://www.baidu.com" 200 OK
   ```

   **注意: 转发目标服务是域名时, 域名解析的结果不要是自身 IP, 可能造成循环转发**

2. 可以多级转发, 假如上面的机器 IP 是 `192.168.1.100`, 现在这台是 `192.168.1.13`

   `./rproxy -debug -L=:555 -F=http://192.168.1.100:7777`

   浏览器访问: `http://192.168.1.13:555/s?ie=utf-8&wd=golang`

   **1.13** 控制台日志:

   ```shell
   ./rproxy -debug -L=:555 -F=http://192.168.1.100:7777
   0907 16:31:54 INF > 监听:=[":555"] 反向代理服务已启动
   0907 16:31:54 INF > 后端:=["http://192.168.1.100:7777"] 转发到后端服务地址
   0907 16:33:33 INF > client_ip="192.168.1.100:50460" request_uri="/s?ie=utf-8&wd=golang" proxy_pass="http://192.168.1.100:7777" 200 OK
   0907 16:33:35 INF > client_ip="192.168.1.100:50460" request_uri="/sugrec?..." proxy_pass="http://192.168.1.100:7777" 200 OK
   0907 16:33:35 INF > client_ip="192.168.1.100:54383" request_uri="/favicon.ico" proxy_pass="http://192.168.1.100:7777" 200 OK
   ```

   **1.100** 控制台日志:

   ```shell
   0907 16:33:33 INF > client_ip="192.168.1.13:47134" request_uri="/s?ie=utf-8&wd=golang" proxy_pass="https://www.baidu.com" 200 OK
   0907 16:33:35 INF > client_ip="192.168.1.13:47134" request_uri="/sugrec?..." proxy_pass="https://www.baidu.com" 200 OK
   0907 16:33:35 INF > client_ip="192.168.1.13:47136" request_uri="/favicon.ico" proxy_pass="https://www.baidu.com" 200 OK
   ```

3. 可以转发到多个后端服务

   1. 支持设定每个后端服务的权重, `-F=协议://地址:端口,权重值`

   2. 默认权重为 1, 全部为 1 或未设置时, 使用轮询; 有权重时使用平滑加权轮询

      ```shell
      ./rproxy -debug -L=:888 -F=http://192.168.1.13:555 -F=http://192.168.1.100:7777,3
      0907 16:39:27 INF > 监听:=[":888"] 反向代理服务已启动
      0907 16:39:27 INF > 后端:=["http://192.168.1.13:555","http://192.168.1.100:7777"] 转发到后端服务地址
      0907 16:39:30 INF > client_ip="127.0.0.1:61877" request_uri="/" proxy_pass="http://192.168.1.100:7777" 200 OK
      0907 16:39:30 INF > client_ip="127.0.0.1:61877" request_uri="/sugrec?..." proxy_pass="http://192.168.1.13:555" 200 OK
      0907 16:39:30 INF > client_ip="127.0.0.1:61877" request_uri="/favicon.ico" proxy_pass="http://192.168.1.100:7777" 200 OK
      0907 16:39:55 INF > client_ip="127.0.0.1:61877" request_uri="/sugrec?..." proxy_pass="http://192.168.1.100:7777" 200 OK
      ```

      浏览器访问 `http://127.0.0.1:888/s?ie=utf-8&wd=fufuok` 就可以看到转发地址按权重均衡了

4. 可以指定请求主机头, 一般转发到 IP 时都最好指定 Host 参数

   **注意: 非 80/443 端口时, 主机头需要加上端口:**

   `./rproxy -debug -F=http://1.1.1.1 -F=http://2.2.2.2 -host=orign.fufuok.com:9001`

   指定 Host 反代 HTTPS 示例:

   ```shell
   ./rproxy -debug -F=https://14.215.177.39,3 -F=https://14.215.177.38,2 -F=https://220.181.38.150 -host=www.baidu.com
   0907 23:40:22 INF > 监听:=[":7777"] 反向代理服务已启动
   0907 23:40:22 INF > 后端:=["https://14.215.177.39","https://14.215.177.38","https://220.181.38.150"] 转发到后端服务地址
   0907 23:40:22 INF > Host:="www.baidu.com" 请求时替换主机头
   0907 23:40:24 INF > client_ip="127.0.0.1:55976" request_uri="/sugrec?..." proxy_pass="https://14.215.177.39" 200 OK
   0907 23:40:25 INF > client_ip="127.0.0.1:55976" request_uri="/s?ie=utf-8&mod=1..." proxy_pass="https://14.215.177.38" 302 Found
   0907 23:40:25 INF > client_ip="127.0.0.1:55976" request_uri="/s?ie=utf-8..." proxy_pass="https://14.215.177.39" 200 OK
   0907 23:40:26 INF > client_ip="127.0.0.1:55976" request_uri="/sugrec?..." proxy_pass="https://220.181.38.150" 200 OK
   ```

5. 也可以同时监听多个本地端口, 即多个 `-L=:nnn`

   `./rproxy -debug -L=:555 -L=:666 -L=:777 -F=http://192.168.1.100:7777`

6. 非调试模式时, 自动启动守护进程并后台运行, 日志记录到文件









*ff*