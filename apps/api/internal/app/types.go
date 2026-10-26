// Package app 定义 P3 业务（积分查询 / 商品目录 / 兑换订单）的 DTO、
// 查询条件、业务错误与端口接口。
//
// 设计意图：
//   - 传输对象与 SQL 查询解耦，HTTP 层只依赖接口，便于用假实现写单测；
//   - 业务错误用哨兵错误表达，HTTP 层统一映射成 {code, message}；
//   - 字段命名与 docs/前端接口需求清单.md 对齐（camelCase JSON）。
package app

import (
	"errors"
	"time"
)

// 业务错误。repo 负责返回，HTTP 层负责映射状态码与业务码。
var (
	ErrNotFound           = errors.New("app: 资源不存在")
	ErrInsufficientPoints = errors.New("app: 积分余额不足")
	ErrOutOfStock         = errors.New("app: 库存不足")
	ErrProductOffline     = errors.New("app: 商品不存在或已下架")
	ErrIllegalTransition  = errors.New("app: 非法的订单状态跃迁")
	ErrConflict           = errors.New("app: 数据冲突")
)

// Page 是统一分页响应体。
type Page[T any] struct {
	Items    []T   `json:"items"`
	Page     int   `json:"page"`
	PageSize int   `json:"pageSize"`
	Total    int64 `json:"total"`
}

// PointsSummary 是 GET /api/v1/points/summary 的响应。
type PointsSummary struct {
	Balance               int64   `json:"balance"`
	Frozen                int64   `json:"frozen"`
	Available             int64   `json:"available"`
	TotalCarbonSavedGrams float64 `json:"totalCarbonSavedGrams"`
	EarnedThisMonth       int64   `json:"earnedThisMonth"`
	EarnedToday           int64   `json:"earnedToday"`
	DailyCap              int64   `json:"dailyCap"`
}

// PointTxn 是一条积分流水。
type PointTxn struct {
	ID           int64     `json:"id"`
	Type         string    `json:"type"`
	Amount       int64     `json:"amount"`
	BalanceAfter int64     `json:"balanceAfter"`
	Reason       string    `json:"reason"`
	RefType      string    `json:"refType"`
	RefID        int64     `json:"refId"`
	CreatedAt    time.Time `json:"createdAt"`
}

// TxnQuery 是流水查询条件。Types 支持 DB 枚举值，
// 也接受前端历史别名 EARN / SPEND / REVERSE。
type TxnQuery struct {
	UserID   int64
	Page     int
	PageSize int
	Types    []string
}

// SKU 是商品规格。
type SKU struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	PointsPrice int64  `json:"pointsPrice"`
	Stock       int    `json:"stock"`
	SoldCount   int    `json:"soldCount"`
}

// Product 是商品。PointsPrice / Stock 为 SKU 聚合值（最低价 / 总库存）。
type Product struct {
	ID           int64  `json:"id"`
	Name         string `json:"name"`
	Description  string `json:"description"`
	Category     string `json:"category"`
	ImageURL     string `json:"imageUrl"`
	PointsPrice  int64  `json:"pointsPrice"`
	Stock        int    `json:"stock"`
	Active       bool   `json:"active"`
	LimitPerUser int    `json:"limitPerUser"`
	SKUs         []SKU  `json:"skus,omitempty"`
}

// ProductQuery 是商品列表查询条件。
type ProductQuery struct {
	Page        int
	PageSize    int
	Category    string
	Keyword     string
	OnlyInStock bool
}

// Order 是兑换订单。
type Order struct {
	ID           int64      `json:"id"`
	OrderNo      string     `json:"orderNo"`
	ProductID    int64      `json:"productId"`
	SKUId        int64      `json:"skuId"`
	ProductName  string     `json:"productName"`
	SKUName      string     `json:"skuName"`
	PointsSpent  int64      `json:"pointsSpent"`
	Quantity     int        `json:"quantity"`
	Receiver     string     `json:"receiver"`
	Status       string     `json:"status"`
	RedeemCode   string     `json:"redeemCode"`
	CreatedAt    time.Time  `json:"createdAt"`
	UpdatedAt    time.Time  `json:"updatedAt"`
	ShippedAt    *time.Time `json:"shippedAt,omitempty"`
	CompletedAt  *time.Time `json:"completedAt,omitempty"`
	CancelledAt  *time.Time `json:"cancelledAt,omitempty"`
	CancelReason string     `json:"cancelReason,omitempty"`
}

// CreateOrderInput 是 POST /api/v1/orders 的入参。
type CreateOrderInput struct {
	UserID         int64
	ProductID      int64
	SKUId          int64 // 0 表示由后端挑选第一个有库存的 SKU
	Receiver       string
	IdempotencyKey string
}

// OrderQuery 是订单列表查询条件。
type OrderQuery struct {
	UserID   int64
	Page     int
	PageSize int
	Status   string
}

// ErrLimitExceeded 表示超过单用户限兑数量。
var ErrLimitExceeded = errors.New("app: 超出单用户限兑数量")
