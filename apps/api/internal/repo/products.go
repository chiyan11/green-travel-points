package repo

import (
	"context"
	"database/sql"
	"errors"

	"greentravelpoints/internal/app"
)

// ListProducts 分页返回上架商品（M6）。
//
// 聚合口径：pointsPrice 取该商品所有 SKU 的最低价，stock 取 SKU 库存之和；
// 仅返回 status = 'ONLINE' 的商品，onlyInStock 时过滤掉总库存为 0 的。
func (s *Store) ListProducts(ctx context.Context, q app.ProductQuery) (app.Page[app.Product], error) {
	page, pageSize := normalizePage(q.Page, q.PageSize)

	where := []string{"p.status = 'ONLINE'"}
	args := make([]any, 0, 4)
	if q.Category != "" {
		where = append(where, "p.category = ?")
		args = append(args, q.Category)
	}
	if q.Keyword != "" {
		like := "%" + q.Keyword + "%"
		where = append(where, "(p.name LIKE ? OR p.description LIKE ?)")
		args = append(args, like, like)
	}
	whereSQL := joinAnd(where)

	having := ""
	if q.OnlyInStock {
		having = " HAVING COALESCE(SUM(s.stock), 0) > 0"
	}

	var total int64
	countSQL := `SELECT COUNT(*) FROM (
		SELECT p.id
		FROM product p LEFT JOIN product_sku s ON s.product_id = p.id
		WHERE ` + whereSQL + `
		GROUP BY p.id` + having + `
	) t`
	if err := s.db.QueryRowContext(ctx, countSQL, args...).Scan(&total); err != nil {
		return app.Page[app.Product]{}, err
	}

	listSQL := `SELECT p.id, p.name, p.description, p.category, p.cover_url, p.limit_per_user,
	       COALESCE(MIN(s.points_price), 0) AS min_price,
	       COALESCE(SUM(s.stock), 0) AS total_stock
	FROM product p LEFT JOIN product_sku s ON s.product_id = p.id
	WHERE ` + whereSQL + `
	GROUP BY p.id, p.name, p.description, p.category, p.cover_url, p.limit_per_user,
	         p.sort_order` + having + `
	ORDER BY p.sort_order ASC, p.id ASC
	LIMIT ? OFFSET ?`

	rows, err := s.db.QueryContext(ctx, listSQL, append(args, pageSize, (page-1)*pageSize)...)
	if err != nil {
		return app.Page[app.Product]{}, err
	}
	defer rows.Close()

	items := make([]app.Product, 0, pageSize)
	for rows.Next() {
		var p app.Product
		if err := rows.Scan(&p.ID, &p.Name, &p.Description, &p.Category, &p.ImageURL,
			&p.LimitPerUser, &p.PointsPrice, &p.Stock); err != nil {
			return app.Page[app.Product]{}, err
		}
		p.Active = true
		items = append(items, p)
	}
	if err := rows.Err(); err != nil {
		return app.Page[app.Product]{}, err
	}
	return app.Page[app.Product]{Items: items, Page: page, PageSize: pageSize, Total: total}, nil
}

// GetProduct 返回上架商品详情（含 SKU 列表）。
func (s *Store) GetProduct(ctx context.Context, id int64) (app.Product, error) {
	var p app.Product
	err := s.db.QueryRowContext(ctx,
		`SELECT id, name, description, category, cover_url, limit_per_user
		 FROM product WHERE id = ? AND status = 'ONLINE'`, id,
	).Scan(&p.ID, &p.Name, &p.Description, &p.Category, &p.ImageURL, &p.LimitPerUser)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return app.Product{}, app.ErrNotFound
		}
		return app.Product{}, err
	}
	p.Active = true

	rows, err := s.db.QueryContext(ctx,
		`SELECT id, sku_name, points_price, stock, sold_count
		 FROM product_sku WHERE product_id = ? ORDER BY id ASC`, id)
	if err != nil {
		return app.Product{}, err
	}
	defer rows.Close()

	first := true
	for rows.Next() {
		var sku app.SKU
		if err := rows.Scan(&sku.ID, &sku.Name, &sku.PointsPrice, &sku.Stock, &sku.SoldCount); err != nil {
			return app.Product{}, err
		}
		if first || sku.PointsPrice < p.PointsPrice {
			p.PointsPrice = sku.PointsPrice
			first = false
		}
		p.Stock += sku.Stock
		p.SKUs = append(p.SKUs, sku)
	}
	if err := rows.Err(); err != nil {
		return app.Product{}, err
	}
	return p, nil
}

func joinAnd(parts []string) string {
	out := ""
	for i, p := range parts {
		if i > 0 {
			out += " AND "
		}
		out += p
	}
	return out
}
