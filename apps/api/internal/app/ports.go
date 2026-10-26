package app

import "context"

// PointsStore 是积分查询端口（M5）。
type PointsStore interface {
	Summary(ctx context.Context, userID int64) (PointsSummary, error)
	ListTransactions(ctx context.Context, q TxnQuery) (Page[PointTxn], error)
}

// CatalogStore 是商品目录端口（M6）。
type CatalogStore interface {
	ListProducts(ctx context.Context, q ProductQuery) (Page[Product], error)
	GetProduct(ctx context.Context, id int64) (Product, error)
}

// OrderStore 是兑换订单端口（M7）。
type OrderStore interface {
	// CreateOrder 返回订单与"是否为幂等重放"；重放时不重复扣分扣库存。
	CreateOrder(ctx context.Context, in CreateOrderInput) (Order, bool, error)
	ListOrders(ctx context.Context, q OrderQuery) (Page[Order], error)
	CancelOrder(ctx context.Context, userID, orderID int64, reason string) (Order, error)
}
