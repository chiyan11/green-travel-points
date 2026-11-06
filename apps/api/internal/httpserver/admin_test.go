package httpserver

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"greentravelpoints/internal/app"
)

type fakeRoles struct {
	role string
	err  error
}

func (f fakeRoles) RoleOf(context.Context, int64) (string, error) { return f.role, f.err }

type fakeAdmin struct {
	products   app.Page[app.Product]
	orders     app.Page[app.Order]
	created    app.Product
	updated    app.Product
	sku        app.SKU
	transition func(id int64, to, reason string) (app.Order, error)
	reconcile  app.ReconcileResult
	err        error
}

func (f fakeAdmin) ListAllProducts(context.Context, app.AdminProductQuery) (app.Page[app.Product], error) {
	return f.products, f.err
}

func (f fakeAdmin) CreateProduct(context.Context, app.AdminProductInput) (app.Product, error) {
	return f.created, f.err
}

func (f fakeAdmin) UpdateProduct(context.Context, int64, app.ProductUpdate) (app.Product, error) {
	return f.updated, f.err
}

func (f fakeAdmin) SetSKUStock(context.Context, int64, int) (app.SKU, error) {
	return f.sku, f.err
}

func (f fakeAdmin) ListAllOrders(context.Context, app.AdminOrderQuery) (app.Page[app.Order], error) {
	return f.orders, f.err
}

func (f fakeAdmin) TransitionOrder(_ context.Context, id int64, to, reason string) (app.Order, error) {
	if f.transition != nil {
		return f.transition(id, to, reason)
	}
	return app.Order{ID: id, Status: to}, nil
}

func (f fakeAdmin) ReconcilePoints(context.Context) (app.ReconcileResult, error) {
	return f.reconcile, f.err
}

func newAdminRouter(admin app.AdminStore, roles app.RoleStore) http.Handler {
	return NewRouter(Options{
		Logger:  slog.New(slog.NewTextHandler(io.Discard, nil)),
		Env:     "test",
		Admin:   admin,
		Roles:   roles,
		Catalog: fakeCatalog{},
	})
}

func TestAdminRequiresUser(t *testing.T) {
	h := newAdminRouter(fakeAdmin{}, fakeRoles{role: "ADMIN"})
	rec, env := doJSON(t, h, http.MethodGet, "/api/v1/admin/products", "", nil)
	if rec.Code != http.StatusUnauthorized || env.Code != CodeUnauthorized {
		t.Fatalf("期望 401/UNAUTHORIZED，实际 %d/%s", rec.Code, env.Code)
	}
}

func TestAdminRejectsNonAdmin(t *testing.T) {
	h := newAdminRouter(fakeAdmin{}, fakeRoles{role: "CITIZEN"})
	rec, env := doJSON(t, h, http.MethodGet, "/api/v1/admin/products", "",
		map[string]string{"X-User-Id": "1001"})
	if rec.Code != http.StatusForbidden || env.Code != CodeForbidden {
		t.Fatalf("期望 403/FORBIDDEN，实际 %d/%s", rec.Code, env.Code)
	}
}

func TestAdminListProducts(t *testing.T) {
	admin := fakeAdmin{products: app.Page[app.Product]{
		Items: []app.Product{{ID: 1001, Name: "帆布袋", Status: "ONLINE", Stock: 10}},
		Page:  1, PageSize: 20, Total: 1,
	}}
	h := newAdminRouter(admin, fakeRoles{role: "ADMIN"})
	rec, env := doJSON(t, h, http.MethodGet, "/api/v1/admin/products?status=ONLINE", "",
		map[string]string{"X-User-Id": "1002"})
	if rec.Code != http.StatusOK || env.Code != CodeOK {
		t.Fatalf("期望 200/OK，实际 %d/%s", rec.Code, env.Code)
	}
	var page app.Page[app.Product]
	if err := json.Unmarshal(env.Data, &page); err != nil {
		t.Fatalf("解析 data 失败: %v", err)
	}
	if len(page.Items) != 1 || page.Items[0].Status != "ONLINE" {
		t.Fatalf("返回商品不符: %+v", page)
	}
}

func TestAdminCreateProductValidation(t *testing.T) {
	h := newAdminRouter(fakeAdmin{}, fakeRoles{role: "ADMIN"})
	cases := []string{
		`{"merchantId":1001,"name":"","category":"文创","skus":[{"name":"款","pointsPrice":100,"stock":1}]}`,
		`{"merchantId":1001,"name":"商品","category":"文创","skus":[]}`,
		`{"merchantId":1001,"name":"商品","category":"文创","skus":[{"name":"款","pointsPrice":0,"stock":1}]}`,
	}
	for _, body := range cases {
		rec, env := doJSON(t, h, http.MethodPost, "/api/v1/admin/products", body,
			map[string]string{"X-User-Id": "1002"})
		if rec.Code != http.StatusBadRequest || env.Code != CodeInvalidArgument {
			t.Fatalf("body=%s 期望 400/INVALID_ARGUMENT，实际 %d/%s", body, rec.Code, env.Code)
		}
	}
}

func TestAdminTransitionIllegal(t *testing.T) {
	admin := fakeAdmin{transition: func(id int64, to, reason string) (app.Order, error) {
		return app.Order{}, app.ErrIllegalTransition
	}}
	h := newAdminRouter(admin, fakeRoles{role: "ADMIN"})
	rec, env := doJSON(t, h, http.MethodPost, "/api/v1/admin/orders/9/ship", "",
		map[string]string{"X-User-Id": "1002"})
	if rec.Code != http.StatusConflict || env.Code != CodeIllegalTransition {
		t.Fatalf("期望 409/ILLEGAL_TRANSITION，实际 %d/%s", rec.Code, env.Code)
	}
}

func TestAdminTransitionOK(t *testing.T) {
	admin := fakeAdmin{transition: func(id int64, to, reason string) (app.Order, error) {
		if to != "SHIPPED" {
			t.Fatalf("目标状态 = %s, 期望 SHIPPED", to)
		}
		return app.Order{ID: id, Status: to}, nil
	}}
	h := newAdminRouter(admin, fakeRoles{role: "ADMIN"})
	rec, env := doJSON(t, h, http.MethodPost, "/api/v1/admin/orders/9/ship", "",
		map[string]string{"X-User-Id": "1002"})
	if rec.Code != http.StatusOK || env.Code != CodeOK {
		t.Fatalf("期望 200/OK，实际 %d/%s", rec.Code, env.Code)
	}
}

func TestAdminReconcile(t *testing.T) {
	admin := fakeAdmin{reconcile: app.ReconcileResult{Accounts: 3, OK: true}}
	h := newAdminRouter(admin, fakeRoles{role: "ADMIN"})
	rec, env := doJSON(t, h, http.MethodGet, "/api/v1/admin/points/reconcile", "",
		map[string]string{"X-User-Id": "1002"})
	if rec.Code != http.StatusOK {
		t.Fatalf("状态码 = %d", rec.Code)
	}
	var res app.ReconcileResult
	if err := json.Unmarshal(env.Data, &res); err != nil {
		t.Fatalf("解析 data 失败: %v", err)
	}
	if !res.OK || res.Accounts != 3 {
		t.Fatalf("对账结果不符: %+v", res)
	}
}

func TestAdminSetStockInvalid(t *testing.T) {
	h := newAdminRouter(fakeAdmin{}, fakeRoles{role: "ADMIN"})
	rec, env := doJSON(t, h, http.MethodPut, "/api/v1/admin/skus/1/stock", `{"stock":-1}`,
		map[string]string{"X-User-Id": "1002"})
	_ = strings.TrimSpace("")
	if rec.Code != http.StatusBadRequest || env.Code != CodeInvalidArgument {
		t.Fatalf("期望 400/INVALID_ARGUMENT，实际 %d/%s", rec.Code, env.Code)
	}
}

var _ = httptest.NewRecorder
