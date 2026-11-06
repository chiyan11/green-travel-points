package app

import "context"

// 以下是运营后台（M8）使用的查询条件与入参。
// 与前台接口的区别：后台可以看到非上架商品、全部订单，并执行状态流转。

// AdminProductQuery 是后台商品列表查询条件。
type AdminProductQuery struct {
	Page     int
	PageSize int
	Status   string // DRAFT / ONLINE / OFFLINE，空表示全部
	Keyword  string
}

// AdminSKUInput 是新建商品时随商品一起提交的 SKU。
type AdminSKUInput struct {
	Name        string `json:"name"`
	PointsPrice int64  `json:"pointsPrice"`
	Stock       int    `json:"stock"`
}

// AdminProductInput 是后台新建商品的入参。
type AdminProductInput struct {
	MerchantID   int64           `json:"merchantId"`
	Name         string          `json:"name"`
	Description  string          `json:"description"`
	Category     string          `json:"category"`
	ImageURL     string          `json:"imageUrl"`
	Status       string          `json:"status"` // 默认 DRAFT
	SortOrder    int             `json:"sortOrder"`
	LimitPerUser int             `json:"limitPerUser"`
	SKUs         []AdminSKUInput `json:"skus"`
}

// ProductUpdate 是后台更新商品的入参（nil 表示不修改该字段）。
type ProductUpdate struct {
	Name         *string `json:"name"`
	Description  *string `json:"description"`
	Category     *string `json:"category"`
	ImageURL     *string `json:"imageUrl"`
	Status       *string `json:"status"`
	SortOrder    *int    `json:"sortOrder"`
	LimitPerUser *int    `json:"limitPerUser"`
}

// AdminOrderQuery 是后台订单列表查询条件。
type AdminOrderQuery struct {
	Page     int
	PageSize int
	Status   string
}

// ReconcileItem 是一条对账不一致的账户记录（BR-07 证据）。
type ReconcileItem struct {
	UserID    int64 `json:"userId"`
	AccountID int64 `json:"accountId"`
	Balance   int64 `json:"balance"`
	LedgerSum int64 `json:"ledgerSum"`
}

// ReconcileResult 是积分对账结果。
type ReconcileResult struct {
	Accounts   int64           `json:"accounts"`
	Mismatches []ReconcileItem `json:"mismatches"`
	OK         bool            `json:"ok"`
}

// AdminStore 是运营后台端口（M8）：商品维护、订单状态流转、积分对账。
type AdminStore interface {
	ListAllProducts(ctx context.Context, q AdminProductQuery) (Page[Product], error)
	CreateProduct(ctx context.Context, in AdminProductInput) (Product, error)
	UpdateProduct(ctx context.Context, id int64, in ProductUpdate) (Product, error)
	SetSKUStock(ctx context.Context, skuID int64, stock int) (SKU, error)
	ListAllOrders(ctx context.Context, q AdminOrderQuery) (Page[Order], error)
	// TransitionOrder 按状态机执行 PENDING→PAID→SHIPPED→COMPLETED。
	TransitionOrder(ctx context.Context, orderID int64, to, reason string) (Order, error)
	ReconcilePoints(ctx context.Context) (ReconcileResult, error)
}
