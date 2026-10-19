// Package repo 负责数据库连接与查询实现。
//
// Sprint 0 只提供连接池；Sprint 1 起用 sqlc 依据 db/migrations 生成强类型查询，
// 生成的代码放在 internal/repo/sqlcgen（见根目录 apps/api/sqlc.yaml）。
package repo

import (
	"database/sql"

	_ "github.com/go-sql-driver/mysql" // MySQL 驱动，注册到 database/sql

	"greentravelpoints/internal/config"
)

// Open 创建 MySQL 连接池。
//
// 注意：sql.Open 不建立实际连接，因此本函数在数据库未启动时也会成功；
// 连通性由 /readyz 探针（Ping）判断，这样开发期可以只跑后台服务。
func Open(cfg config.Config) (*sql.DB, error) {
	db, err := sql.Open("mysql", cfg.DatabaseDSN)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(cfg.DBMaxOpenConns)
	db.SetMaxIdleConns(cfg.DBMaxIdleConns)
	db.SetConnMaxLifetime(cfg.DBConnMaxLifetime)
	return db, nil
}
