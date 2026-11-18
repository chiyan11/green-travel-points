package main

import (
	"testing"

	"greentravelpoints/internal/config"
)

func TestDBEndpoint(t *testing.T) {
	tests := []struct {
		name      string
		dsn       string
		wantAddr  string
		wantLocal bool
		wantOK    bool
	}{
		{
			name:      "默认 DSN 指向本机",
			dsn:       config.DefaultDatabaseDSN,
			wantAddr:  "127.0.0.1:3306",
			wantLocal: true,
			wantOK:    true,
		},
		{
			name:      "局域网共享地址不算本机",
			dsn:       "greentravel:greentravel@tcp(10.111.230.162:3306)/green_travel_points?parseTime=true",
			wantAddr:  "10.111.230.162:3306",
			wantLocal: false,
			wantOK:    true,
		},
		{
			name:      "ZeroTier 地址不算本机",
			dsn:       "greentravel:greentravel@tcp(10.213.180.114:3306)/green_travel_points",
			wantAddr:  "10.213.180.114:3306",
			wantLocal: false,
			wantOK:    true,
		},
		{
			name:      "主机名 localhost 算本机",
			dsn:       "u:p@tcp(localhost:3306)/db",
			wantAddr:  "localhost:3306",
			wantLocal: true,
			wantOK:    true,
		},
		{
			name:      "IPv6 回环算本机",
			dsn:       "u:p@tcp([::1]:3306)/db",
			wantAddr:  "[::1]:3306",
			wantLocal: true,
			wantOK:    true,
		},
		{
			name:      "省略地址按驱动默认回落本机",
			dsn:       "u:p@/db",
			wantAddr:  "127.0.0.1:3306",
			wantLocal: true,
			wantOK:    true,
		},
		{
			name:      "127.0.0.0/8 整段算本机",
			dsn:       "u:p@tcp(127.0.0.2:3306)/db",
			wantAddr:  "127.0.0.2:3306",
			wantLocal: true,
			wantOK:    true,
		},
		{
			name:      "unix socket 算本机",
			dsn:       "u:p@unix(/tmp/mysql.sock)/db",
			wantAddr:  "/tmp/mysql.sock",
			wantLocal: true,
			wantOK:    true,
		},
		{
			name:      "非法 DSN 解析失败",
			dsn:       "not-a-valid-dsn",
			wantAddr:  "",
			wantLocal: false,
			wantOK:    false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			addr, local, ok := dbEndpoint(tt.dsn)
			if addr != tt.wantAddr || local != tt.wantLocal || ok != tt.wantOK {
				t.Fatalf("dbEndpoint(%q) = (%q, %v, %v)，期望 (%q, %v, %v)",
					tt.dsn, addr, local, ok, tt.wantAddr, tt.wantLocal, tt.wantOK)
			}
		})
	}
}
