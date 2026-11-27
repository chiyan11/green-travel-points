package httpserver

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
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
	products     app.Page[app.Product]
	orders       app.Page[app.Order]
	created      app.Product
	updated      app.Product
	sku          app.SKU
	config       []app.ConfigItem
	mode         app.TravelMode
	expire       app.ExpireResult
	adjust       app.AdjustResult
	audits       app.Page[app.AuditLog]
	transition   func(id int64, to, reason string) (app.Order, error)
	updateConfig func(operatorID int64, updates []app.ConfigUpdate) ([]app.ConfigItem, error)
	updateMode   func(operatorID int64, code string, in app.TravelModeUpdate) (app.TravelMode, error)
	adjustFn     func(operatorID int64, in app.AdjustPointsInput) (app.AdjustResult, error)
	auditFn      func(q app.AuditQuery) (app.Page[app.AuditLog], error)
	reconcile    app.ReconcileResult
	err          error
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

func (f fakeAdmin) ListConfig(context.Context) ([]app.ConfigItem, error) {
	return f.config, f.err
}

func (f fakeAdmin) UpdateConfig(_ context.Context, operatorID int64, updates []app.ConfigUpdate) ([]app.ConfigItem, error) {
	if f.updateConfig != nil {
		return f.updateConfig(operatorID, updates)
	}
	return f.config, f.err
}

func (f fakeAdmin) UpdateTravelMode(_ context.Context, operatorID int64, code string, in app.TravelModeUpdate) (app.TravelMode, error) {
	if f.updateMode != nil {
		return f.updateMode(operatorID, code, in)
	}
	return f.mode, f.err
}

func (f fakeAdmin) ExpirePoints(context.Context) (app.ExpireResult, error) {
	return f.expire, f.err
}

func (f fakeAdmin) AdjustPoints(_ context.Context, operatorID int64, in app.AdjustPointsInput) (app.AdjustResult, error) {
	if f.adjustFn != nil {
		return f.adjustFn(operatorID, in)
	}
	return f.adjust, f.err
}

func (f fakeAdmin) ListAuditLogs(_ context.Context, q app.AuditQuery) (app.Page[app.AuditLog], error) {
	if f.auditFn != nil {
		return f.auditFn(q)
	}
	return f.audits, f.err
}

func newAdminRouter(admin app.AdminStore, roles app.RoleStore) http.Handler {
	return NewRouter(Options{
		Logger:         slog.New(slog.NewTextHandler(io.Discard, nil)),
		Env:            "test",
		AllowDevHeader: true,
		Admin:          admin,
		Roles:          roles,
		Catalog:        fakeCatalog{},
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

// 后台商品列表必须带 SKU 明细：否则「改库存」屏拿不到 skuId，只能只读展示。
func TestAdminListProductsIncludesSKUs(t *testing.T) {
	admin := fakeAdmin{products: app.Page[app.Product]{
		Items: []app.Product{{
			ID: 1003, Name: "环保水杯", Status: "DRAFT", PointsPrice: 120, Stock: 20,
			SKUs: []app.SKU{{ID: 2001, Name: "标准款", PointsPrice: 120, Stock: 20}},
		}},
		Page: 1, PageSize: 20, Total: 1,
	}}
	h := newAdminRouter(admin, fakeRoles{role: "ADMIN"})
	rec, env := doJSON(t, h, http.MethodGet, "/api/v1/admin/products", "",
		map[string]string{"X-User-Id": "1002"})
	if rec.Code != http.StatusOK || env.Code != CodeOK {
		t.Fatalf("期望 200/OK，实际 %d/%s", rec.Code, env.Code)
	}
	var page struct {
		Items []struct {
			SKUs []app.SKU `json:"skus"`
		} `json:"items"`
	}
	if err := json.Unmarshal(env.Data, &page); err != nil {
		t.Fatalf("解析 data 失败: %v", err)
	}
	if len(page.Items) != 1 || len(page.Items[0].SKUs) != 1 || page.Items[0].SKUs[0].ID != 2001 {
		t.Fatalf("列表未带 SKU 明细: %+v", page)
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

// ---------- UC-14 参数配置 ----------

func TestAdminListConfig(t *testing.T) {
	admin := fakeAdmin{config: []app.ConfigItem{
		{Key: "carbon_per_point", Value: "100", ValueType: "int", Description: "多少克碳减排量兑换 1 积分"},
		{Key: "daily_points_cap", Value: "500", ValueType: "int", Description: "单日可获得积分上限"},
	}}
	h := newAdminRouter(admin, fakeRoles{role: "ADMIN"})
	rec, env := doJSON(t, h, http.MethodGet, "/api/v1/admin/config", "",
		map[string]string{"X-User-Id": "1002"})
	if rec.Code != http.StatusOK || env.Code != CodeOK {
		t.Fatalf("期望 200/OK，实际 %d/%s", rec.Code, env.Code)
	}
	var items []app.ConfigItem
	if err := json.Unmarshal(env.Data, &items); err != nil {
		t.Fatalf("解析 data 失败: %v", err)
	}
	if len(items) != 2 || items[1].Key != "daily_points_cap" || items[1].Value != "500" {
		t.Fatalf("参数列表不符: %+v", items)
	}
}

func TestAdminUpdateConfigForbidden(t *testing.T) {
	h := newAdminRouter(fakeAdmin{}, fakeRoles{role: "CITIZEN"})
	rec, env := doJSON(t, h, http.MethodPut, "/api/v1/admin/config",
		`{"items":[{"key":"daily_points_cap","value":"10"}]}`,
		map[string]string{"X-User-Id": "1001"})
	if rec.Code != http.StatusForbidden || env.Code != CodeForbidden {
		t.Fatalf("期望 403/FORBIDDEN，实际 %d/%s", rec.Code, env.Code)
	}
}

func TestAdminUpdateConfigOK(t *testing.T) {
	var got []app.ConfigUpdate
	var gotOperator int64
	admin := fakeAdmin{updateConfig: func(operatorID int64, updates []app.ConfigUpdate) ([]app.ConfigItem, error) {
		gotOperator = operatorID
		got = updates
		return []app.ConfigItem{{Key: "daily_points_cap", Value: "10", ValueType: "int"}}, nil
	}}
	h := newAdminRouter(admin, fakeRoles{role: "ADMIN"})
	rec, env := doJSON(t, h, http.MethodPut, "/api/v1/admin/config",
		`{"items":[{"key":"daily_points_cap","value":"10"}]}`,
		map[string]string{"X-User-Id": "1002"})
	if rec.Code != http.StatusOK || env.Code != CodeOK {
		t.Fatalf("期望 200/OK，实际 %d/%s", rec.Code, env.Code)
	}
	if gotOperator != 1002 {
		t.Fatalf("操作人透传错误：got %d, 期望 1002（审计要用）", gotOperator)
	}
	if len(got) != 1 || got[0].Key != "daily_points_cap" || got[0].Value != "10" {
		t.Fatalf("透传入参不符: %+v", got)
	}
	var items []app.ConfigItem
	if err := json.Unmarshal(env.Data, &items); err != nil {
		t.Fatalf("解析 data 失败: %v", err)
	}
	if len(items) != 1 || items[0].Value != "10" {
		t.Fatalf("返回参数不符: %+v", items)
	}
}

func TestAdminUpdateConfigInvalidBody(t *testing.T) {
	h := newAdminRouter(fakeAdmin{}, fakeRoles{role: "ADMIN"})
	cases := []string{
		`{"items":[]}`,                                                    // 空批次
		`{"items":[{"key":"","value":"10"}]}`,                             // 缺 key
		`{"items":[{"key":"daily_points_cap","value":""}]}`,               // 缺 value
		`{"items":[{"key":"daily_points_cap","value":"10","extra":1}]}`,   // 未知字段
		`{"items":[{"key":"daily_points_cap","value":"10"}],"unknown":1}`, // 未知顶层字段
	}
	for _, body := range cases {
		rec, env := doJSON(t, h, http.MethodPut, "/api/v1/admin/config", body,
			map[string]string{"X-User-Id": "1002"})
		if rec.Code != http.StatusBadRequest || env.Code != CodeInvalidArgument {
			t.Fatalf("body=%s 期望 400/INVALID_ARGUMENT，实际 %d/%s", body, rec.Code, env.Code)
		}
	}
}

func TestAdminUpdateConfigUnknownKey(t *testing.T) {
	admin := fakeAdmin{updateConfig: func(int64, []app.ConfigUpdate) ([]app.ConfigItem, error) {
		return nil, fmt.Errorf("%w: 未知参数键 nope", app.ErrInvalidInput)
	}}
	h := newAdminRouter(admin, fakeRoles{role: "ADMIN"})
	rec, env := doJSON(t, h, http.MethodPut, "/api/v1/admin/config",
		`{"items":[{"key":"nope","value":"1"}]}`, map[string]string{"X-User-Id": "1002"})
	if rec.Code != http.StatusBadRequest || env.Code != CodeInvalidArgument {
		t.Fatalf("期望 400/INVALID_ARGUMENT，实际 %d/%s", rec.Code, env.Code)
	}
}

func TestAdminUpdateTravelModeOK(t *testing.T) {
	admin := fakeAdmin{updateMode: func(operatorID int64, code string, in app.TravelModeUpdate) (app.TravelMode, error) {
		if code != "bus" {
			t.Fatalf("code = %q, 期望 bus", code)
		}
		if in.EFActual == nil || *in.EFActual != 50 {
			t.Fatalf("efActual 未透传: %+v", in)
		}
		return app.TravelMode{Code: code, Name: "公交", EFBase: 192, EFActual: 50, Enabled: true}, nil
	}}
	h := newAdminRouter(admin, fakeRoles{role: "ADMIN"})
	rec, env := doJSON(t, h, http.MethodPut, "/api/v1/admin/travel-modes/bus",
		`{"efActual":50}`, map[string]string{"X-User-Id": "1002"})
	if rec.Code != http.StatusOK || env.Code != CodeOK {
		t.Fatalf("期望 200/OK，实际 %d/%s", rec.Code, env.Code)
	}
	var mode app.TravelMode
	if err := json.Unmarshal(env.Data, &mode); err != nil {
		t.Fatalf("解析 data 失败: %v", err)
	}
	if mode.Code != "bus" || mode.EFActual != 50 || !mode.Enabled {
		t.Fatalf("返回出行方式不符: %+v", mode)
	}
}

func TestAdminUpdateTravelModeInvalid(t *testing.T) {
	h := newAdminRouter(fakeAdmin{}, fakeRoles{role: "ADMIN"})
	cases := []string{
		`{}`,                       // 空更新
		`{"efActual":-1}`,          // 负排放因子
		`{"efBase":0}`,             // 基准因子必须为正
		`{"name":"  "}`,            // 空名称
		`{"efActual":1,"foo":"x"}`, // 未知字段
	}
	for _, body := range cases {
		rec, env := doJSON(t, h, http.MethodPut, "/api/v1/admin/travel-modes/bus", body,
			map[string]string{"X-User-Id": "1002"})
		if rec.Code != http.StatusBadRequest || env.Code != CodeInvalidArgument {
			t.Fatalf("body=%s 期望 400/INVALID_ARGUMENT，实际 %d/%s", body, rec.Code, env.Code)
		}
	}
}

func TestAdminUpdateTravelModeNotFound(t *testing.T) {
	admin := fakeAdmin{updateMode: func(int64, string, app.TravelModeUpdate) (app.TravelMode, error) {
		return app.TravelMode{}, app.ErrNotFound
	}}
	h := newAdminRouter(admin, fakeRoles{role: "ADMIN"})
	rec, env := doJSON(t, h, http.MethodPut, "/api/v1/admin/travel-modes/hoverboard",
		`{"enabled":false}`, map[string]string{"X-User-Id": "1002"})
	if rec.Code != http.StatusNotFound || env.Code != CodeNotFound {
		t.Fatalf("期望 404/NOT_FOUND，实际 %d/%s", rec.Code, env.Code)
	}
}

// ---------- P3-3 积分过期清算 ----------

func TestAdminExpirePointsOK(t *testing.T) {
	admin := fakeAdmin{expire: app.ExpireResult{Batches: 2, Points: 150, Accounts: 1}}
	h := newAdminRouter(admin, fakeRoles{role: "ADMIN"})
	rec, env := doJSON(t, h, http.MethodPost, "/api/v1/admin/points/expire", "",
		map[string]string{"X-User-Id": "1002"})
	if rec.Code != http.StatusOK || env.Code != CodeOK {
		t.Fatalf("期望 200/OK，实际 %d/%s", rec.Code, env.Code)
	}
	var res app.ExpireResult
	if err := json.Unmarshal(env.Data, &res); err != nil {
		t.Fatalf("解析 data 失败: %v", err)
	}
	if res.Batches != 2 || res.Points != 150 || res.Accounts != 1 {
		t.Fatalf("清算结果不符: %+v", res)
	}
}

func TestAdminExpirePointsForbidden(t *testing.T) {
	h := newAdminRouter(fakeAdmin{}, fakeRoles{role: "CITIZEN"})
	rec, env := doJSON(t, h, http.MethodPost, "/api/v1/admin/points/expire", "",
		map[string]string{"X-User-Id": "1001"})
	if rec.Code != http.StatusForbidden || env.Code != CodeForbidden {
		t.Fatalf("期望 403/FORBIDDEN，实际 %d/%s", rec.Code, env.Code)
	}
}

func TestAdminExpirePointsInternalError(t *testing.T) {
	h := newAdminRouter(fakeAdmin{err: errors.New("db down")}, fakeRoles{role: "ADMIN"})
	rec, env := doJSON(t, h, http.MethodPost, "/api/v1/admin/points/expire", "",
		map[string]string{"X-User-Id": "1002"})
	if rec.Code != http.StatusInternalServerError || env.Code != CodeInternal {
		t.Fatalf("期望 500/INTERNAL，实际 %d/%s", rec.Code, env.Code)
	}
}

// ---------- P3-8 人工调账 + 审计查询 ----------

func TestAdminAdjustPointsOK(t *testing.T) {
	var gotOperator int64
	var gotInput app.AdjustPointsInput
	admin := fakeAdmin{adjustFn: func(operatorID int64, in app.AdjustPointsInput) (app.AdjustResult, error) {
		gotOperator = operatorID
		gotInput = in
		return app.AdjustResult{AccountID: 7, UserID: in.UserID, Balance: 950, AuditID: 42}, nil
	}}
	h := newAdminRouter(admin, fakeRoles{role: "ADMIN"})
	rec, env := doJSON(t, h, http.MethodPost, "/api/v1/admin/points/adjust",
		`{"userId":1001,"amount":-50,"reason":"刷分扣回"}`,
		map[string]string{"X-User-Id": "1002"})
	if rec.Code != http.StatusOK || env.Code != CodeOK {
		t.Fatalf("期望 200/OK，实际 %d/%s", rec.Code, env.Code)
	}
	if gotOperator != 1002 {
		t.Fatalf("操作人 = %d，期望 1002（审计要记操作人）", gotOperator)
	}
	if gotInput.UserID != 1001 || gotInput.Amount != -50 || gotInput.Reason != "刷分扣回" {
		t.Fatalf("调账入参不符: %+v", gotInput)
	}
	var res app.AdjustResult
	if err := json.Unmarshal(env.Data, &res); err != nil {
		t.Fatalf("解析 data 失败: %v", err)
	}
	if res.Balance != 950 || res.AuditID != 42 {
		t.Fatalf("调账结果不符: %+v", res)
	}
}

func TestAdminAdjustPointsInvalid(t *testing.T) {
	h := newAdminRouter(fakeAdmin{}, fakeRoles{role: "ADMIN"})
	cases := []string{
		`{}`, // 全缺
		`{"userId":1001,"amount":0,"reason":"零值"}`,           // amount 必须非 0
		`{"userId":1001,"amount":10}`,                        // 缺理由（BR-14 必填）
		`{"userId":0,"amount":10,"reason":"缺用户"}`,            // 缺 userId
		`{"userId":1001,"amount":10,"reason":"x","extra":1}`, // 未知字段
	}
	for _, body := range cases {
		rec, env := doJSON(t, h, http.MethodPost, "/api/v1/admin/points/adjust", body,
			map[string]string{"X-User-Id": "1002"})
		if rec.Code != http.StatusBadRequest || env.Code != CodeInvalidArgument {
			t.Fatalf("body=%s 期望 400/INVALID_ARGUMENT，实际 %d/%s", body, rec.Code, env.Code)
		}
	}
}

func TestAdminAdjustPointsInsufficient(t *testing.T) {
	admin := fakeAdmin{adjustFn: func(int64, app.AdjustPointsInput) (app.AdjustResult, error) {
		return app.AdjustResult{}, fmt.Errorf("%w: 余额不足", app.ErrInsufficientPoints)
	}}
	h := newAdminRouter(admin, fakeRoles{role: "ADMIN"})
	rec, env := doJSON(t, h, http.MethodPost, "/api/v1/admin/points/adjust",
		`{"userId":1001,"amount":-99999,"reason":"超额扣减"}`,
		map[string]string{"X-User-Id": "1002"})
	if rec.Code != http.StatusConflict || env.Code != CodeInsufficientPoints {
		t.Fatalf("期望 409/INSUFFICIENT_POINTS，实际 %d/%s", rec.Code, env.Code)
	}
}

func TestAdminAdjustPointsForbidden(t *testing.T) {
	h := newAdminRouter(fakeAdmin{}, fakeRoles{role: "CITIZEN"})
	rec, env := doJSON(t, h, http.MethodPost, "/api/v1/admin/points/adjust",
		`{"userId":1001,"amount":10,"reason":"越权"}`,
		map[string]string{"X-User-Id": "1001"})
	if rec.Code != http.StatusForbidden || env.Code != CodeForbidden {
		t.Fatalf("期望 403/FORBIDDEN，实际 %d/%s", rec.Code, env.Code)
	}
}

func TestAdminAuditLogs(t *testing.T) {
	var gotQuery app.AuditQuery
	admin := fakeAdmin{auditFn: func(q app.AuditQuery) (app.Page[app.AuditLog], error) {
		gotQuery = q
		return app.Page[app.AuditLog]{
			Items: []app.AuditLog{{
				ID: 42, OperatorID: 1002, Action: "ADJUST", TargetType: "point_account",
				TargetKey: "1001", Amount: -50, BeforeValue: "1000", AfterValue: "950",
				Reason: "刷分扣回",
			}},
			Page: 1, PageSize: 20, Total: 1,
		}, nil
	}}
	h := newAdminRouter(admin, fakeRoles{role: "ADMIN"})
	rec, env := doJSON(t, h, http.MethodGet,
		"/api/v1/admin/audit-logs?operatorId=1002&action=adjust&targetType=point_account&targetKey=1001",
		"", map[string]string{"X-User-Id": "1002"})
	if rec.Code != http.StatusOK || env.Code != CodeOK {
		t.Fatalf("期望 200/OK，实际 %d/%s", rec.Code, env.Code)
	}
	if gotQuery.OperatorID != 1002 || gotQuery.Action != "adjust" ||
		gotQuery.TargetType != "point_account" || gotQuery.TargetKey != "1001" {
		t.Fatalf("检索条件透传不符: %+v", gotQuery)
	}
	var page app.Page[app.AuditLog]
	if err := json.Unmarshal(env.Data, &page); err != nil {
		t.Fatalf("解析 data 失败: %v", err)
	}
	if page.Total != 1 || page.Items[0].Action != "ADJUST" || page.Items[0].Reason != "刷分扣回" {
		t.Fatalf("审计列表不符: %+v", page)
	}
}

func TestAdminAuditLogsForbidden(t *testing.T) {
	h := newAdminRouter(fakeAdmin{}, fakeRoles{role: "CITIZEN"})
	rec, env := doJSON(t, h, http.MethodGet, "/api/v1/admin/audit-logs", "",
		map[string]string{"X-User-Id": "1001"})
	if rec.Code != http.StatusForbidden || env.Code != CodeForbidden {
		t.Fatalf("期望 403/FORBIDDEN，实际 %d/%s", rec.Code, env.Code)
	}
}
