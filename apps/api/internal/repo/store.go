// Package repo 实现数据库访问：商品目录（M6）、积分查询（M5）、兑换订单（M7）。
//
// 约定：
//   - 所有金额 / 积分用 int64，禁止浮点；
//   - 写操作一律在事务内完成，账户与 SKU 用 SELECT ... FOR UPDATE 锁后复核（BR-09）；
//   - 订单状态机由 domain.CheckOrderTransition 校验（BR-10）；
//   - 积分流水 append-only，取消订单用 REFUND 反向流水（BR-08 / BR-12）。
package repo

import (
	"database/sql"
	"errors"

	"github.com/go-sql-driver/mysql"
)

// Store 聚合 P3 的三个端口的 MySQL 实现。
type Store struct {
	db *sql.DB
}

// New 用连接池构造 Store。
func New(db *sql.DB) *Store { return &Store{db: db} }

// isDuplicate 判断是否为唯一键冲突（MySQL 1062）。
func isDuplicate(err error) bool {
	var me *mysql.MySQLError
	return errors.As(err, &me) && me.Number == 1062
}
