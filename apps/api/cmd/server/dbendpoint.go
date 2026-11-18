package main

import (
	"net"
	"strings"

	"github.com/go-sql-driver/mysql"
)

// defaultDBAddr 与 go-sql-driver/mysql 的默认地址一致：DSN 里没写地址时驱动回落到本机 3306。
const defaultDBAddr = "127.0.0.1:3306"

// dbEndpoint 从 DB_DSN 中取出数据库目标地址，用于启动日志提示"连的是本机库还是他人共享库"。
//
// 返回值：
//   - addr：host:port；unix socket / 命名管道时原样返回路径
//   - local：目标是否为本机实例
//   - ok：DSN 能否解析；false 时调用方跳过提示即可
//
// 只返回地址，不包含 DSN 中的账号密码，可安全写入日志。
func dbEndpoint(dsn string) (addr string, local bool, ok bool) {
	cfg, err := mysql.ParseDSN(dsn)
	if err != nil {
		return "", false, false
	}
	if cfg.Net != "" && cfg.Net != "tcp" {
		// unix socket / 命名管道：只可能在本机。
		return cfg.Addr, true, true
	}
	addr = cfg.Addr
	if addr == "" {
		addr = defaultDBAddr
	}
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		// 只有主机名没有端口时退化成按主机名判断。
		return addr, isLoopbackHost(addr), true
	}
	return addr, isLoopbackHost(host), true
}

// isLoopbackHost 判断主机名是否指向本机：localhost、127.0.0.0/8、::1。
func isLoopbackHost(host string) bool {
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}
