package httpserver

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"greentravelpoints/internal/app"
)

// ---------- 假实现（端口接口的测试替身，不依赖数据库） ----------

type fakeCatalog struct {
	list app.Page[app.Product]
	one  app.Product
	err  error
}

func (f fakeCatalog) ListProducts(context.Context, app.ProductQuery) (app.Page[app.Product], error) {
	if f.err != nil {
		return app.Page[app.Product]{}, f.err
	}
	return f.list, nil
}

func (f fakeCatalog) GetProduct(context.Context, int64) (app.Product, error) {
	if f.err != nil {
		return app.Product{}, f.err
	}
	return f.one, nil
}

type fakePoints struct {
	summary app.PointsSummary
	page    app.Page[app.PointTxn]
	daily   []app.DailyCarbonPoint
	err     error
}

func (f fakePoints) Summary(context.Context, int64) (app.PointsSummary, error) {
	if f.err != nil {
		return app.PointsSummary{}, f.err
	}
	return f.summary, nil
}

func (f fakePoints) ListTransactions(context.Context, app.TxnQuery) (app.Page[app.PointTxn], error) {
	if f.err != nil {
		return app.Page[app.PointTxn]{}, f.err
	}
	return f.page, nil
}

func (f fakePoints) DailyCarbon(context.Context, int64, int) ([]app.DailyCarbonPoint, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.daily, nil
}

type fakeOrders struct {
	create func(app.CreateOrderInput) (app.Order, bool, error)
	list   func(app.OrderQuery) (app.Page[app.Order], error)
	cancel func(int64, int64, string) (app.Order, error)
}

func (f fakeOrders) CreateOrder(_ context.Context, in app.CreateOrderInput) (app.Order, bool, error) {
	return f.create(in)
}

func (f fakeOrders) ListOrders(_ context.Context, q app.OrderQuery) (app.Page[app.Order], error) {
	return f.list(q)
}

func (f fakeOrders) CancelOrder(_ context.Context, userID, orderID int64, reason string) (app.Order, error) {
	return f.cancel(userID, orderID, reason)
}

type fakeTrips struct {
	create func(app.CreateTripInput) (app.Trip, error)
	list   func(app.TripQuery) (app.Page[app.Trip], error)
}

func (f fakeTrips) CreateTrip(_ context.Context, in app.CreateTripInput) (app.Trip, error) {
	return f.create(in)
}

func (f fakeTrips) ListTrips(_ context.Context, q app.TripQuery) (app.Page[app.Trip], error) {
	return f.list(q)
}

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func newTestRouter(c app.CatalogStore, p app.PointsStore, o app.OrderStore) http.Handler {
	return newTestRouterWithTrips(c, p, o, nil)
}

func newTestRouterWithTrips(c app.CatalogStore, p app.PointsStore, o app.OrderStore, trips app.TripStore) http.Handler {
	return NewRouter(Options{
		Logger: discardLogger(),
		Env:    "test",
		// 用例靠 X-User-Id 指定用户，显式开启；默认关闭由 authz_test.go 覆盖。
		AllowDevHeader: true,
		Catalog:        c,
		Points:         p,
		Orders:         o,
		Trips:          trips,
	})
}

type envelope struct {
	Code    string          `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

func doJSON(t *testing.T, h http.Handler, method, path, body string, headers map[string]string) (*httptest.ResponseRecorder, envelope) {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	var env envelope
	_ = json.Unmarshal(rec.Body.Bytes(), &env)
	return rec, env
}

// ---------- 商品目录 ----------

func TestProductsList(t *testing.T) {
	c := fakeCatalog{list: app.Page[app.Product]{
		Items: []app.Product{{ID: 1, Name: "帆布袋", PointsPrice: 100, Stock: 3, Active: true}},
		Page:  1, PageSize: 20, Total: 1,
	}}
	h := newTestRouter(c, nil, nil)

	rec, env := doJSON(t, h, http.MethodGet, "/api/v1/products?page=1&pageSize=20", "", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("状态码 = %d, 期望 200, body=%s", rec.Code, rec.Body.String())
	}
	if env.Code != CodeOK {
		t.Fatalf("business code = %s, 期望 OK", env.Code)
	}
	var page app.Page[app.Product]
	if err := json.Unmarshal(env.Data, &page); err != nil {
		t.Fatalf("解析 data 失败: %v", err)
	}
	if len(page.Items) != 1 || page.Items[0].Name != "帆布袋" {
		t.Fatalf("返回商品不符: %+v", page)
	}
}

func TestProductDetailNotFound(t *testing.T) {
	h := newTestRouter(fakeCatalog{err: app.ErrNotFound}, nil, nil)
	rec, env := doJSON(t, h, http.MethodGet, "/api/v1/products/99", "", nil)
	if rec.Code != http.StatusNotFound || env.Code != CodeNotFound {
		t.Fatalf("期望 404/NOT_FOUND，实际 %d/%s", rec.Code, env.Code)
	}
}

func TestProductDetailInvalidID(t *testing.T) {
	h := newTestRouter(fakeCatalog{}, nil, nil)
	rec, env := doJSON(t, h, http.MethodGet, "/api/v1/products/abc", "", nil)
	if rec.Code != http.StatusBadRequest || env.Code != CodeInvalidArgument {
		t.Fatalf("期望 400/INVALID_ARGUMENT，实际 %d/%s", rec.Code, env.Code)
	}
}

// ---------- 积分 ----------

func TestCreateTrip(t *testing.T) {
	traveledAt := time.Now().Add(-time.Hour).Truncate(time.Second)
	trips := fakeTrips{create: func(in app.CreateTripInput) (app.Trip, error) {
		if in.UserID != 7 || in.Mode != "bus" || in.DistanceKm != 10 || !in.TraveledAt.Equal(traveledAt) {
			t.Fatalf("透传参数不符: %+v", in)
		}
		return app.Trip{ID: "trip_1", Mode: in.Mode, DistanceKm: in.DistanceKm, TraveledAt: in.TraveledAt,
			CarbonSavedGrams: 1240, RawPoints: 12, GrantedPoints: 12, Status: "VERIFIED"}, nil
	}}
	h := newTestRouterWithTrips(nil, nil, nil, trips)
	rec, env := doJSON(t, h, http.MethodPost, "/api/v1/trips",
		`{"mode":"bus","distanceKm":10,"traveledAt":"`+traveledAt.Format(time.RFC3339)+`"}`,
		map[string]string{"X-User-Id": "7"})
	if rec.Code != http.StatusOK || env.Code != CodeOK {
		t.Fatalf("期望 200/OK，实际 %d/%s", rec.Code, env.Code)
	}
}

func TestCreateTripInvalidInput(t *testing.T) {
	trips := fakeTrips{create: func(app.CreateTripInput) (app.Trip, error) {
		return app.Trip{}, app.ErrInvalidInput
	}}
	h := newTestRouterWithTrips(nil, nil, nil, trips)
	rec, env := doJSON(t, h, http.MethodPost, "/api/v1/trips",
		`{"mode":"bus","distanceKm":10,"traveledAt":"`+time.Now().Add(-time.Hour).Format(time.RFC3339)+`"}`,
		map[string]string{"X-User-Id": "7"})
	if rec.Code != http.StatusBadRequest || env.Code != CodeInvalidArgument {
		t.Fatalf("期望 400/INVALID_ARGUMENT，实际 %d/%s", rec.Code, env.Code)
	}
}

func TestCreateTripStoreErrorIsInternal(t *testing.T) {
	trips := fakeTrips{create: func(app.CreateTripInput) (app.Trip, error) {
		return app.Trip{}, errors.New("database unavailable")
	}}
	h := newTestRouterWithTrips(nil, nil, nil, trips)
	rec, env := doJSON(t, h, http.MethodPost, "/api/v1/trips",
		`{"mode":"bus","distanceKm":10,"traveledAt":"`+time.Now().Add(-time.Hour).Format(time.RFC3339)+`"}`,
		map[string]string{"X-User-Id": "7"})
	if rec.Code != http.StatusInternalServerError || env.Code != CodeInternal {
		t.Fatalf("期望 500/INTERNAL，实际 %d/%s", rec.Code, env.Code)
	}
	if strings.Contains(rec.Body.String(), "database unavailable") {
		t.Fatal("内部数据库错误不应返回给客户端")
	}
}

func TestCreateTripFutureTimeDoesNotCallStore(t *testing.T) {
	trips := fakeTrips{create: func(app.CreateTripInput) (app.Trip, error) {
		t.Fatal("未来时间不应调用存储")
		return app.Trip{}, nil
	}}
	h := newTestRouterWithTrips(nil, nil, nil, trips)
	rec, env := doJSON(t, h, http.MethodPost, "/api/v1/trips",
		`{"mode":"bus","distanceKm":10,"traveledAt":"`+time.Now().Add(time.Hour).Format(time.RFC3339)+`"}`,
		map[string]string{"X-User-Id": "7"})
	if rec.Code != http.StatusBadRequest || env.Code != CodeInvalidArgument {
		t.Fatalf("期望 400/INVALID_ARGUMENT，实际 %d/%s", rec.Code, env.Code)
	}
}

func TestListTrips(t *testing.T) {
	trips := fakeTrips{list: func(q app.TripQuery) (app.Page[app.Trip], error) {
		if q.UserID != 7 || q.Page != 2 || q.PageSize != 1 {
			t.Fatalf("分页参数不符: %+v", q)
		}
		return app.Page[app.Trip]{Items: []app.Trip{{ID: "trip_1", Mode: "walk", Status: "VALID"}},
			Page: q.Page, PageSize: q.PageSize, Total: 3}, nil
	}}
	h := newTestRouterWithTrips(nil, nil, nil, trips)
	rec, env := doJSON(t, h, http.MethodGet, "/api/v1/trips?page=2&pageSize=1", "", map[string]string{"X-User-Id": "7"})
	if rec.Code != http.StatusOK || env.Code != CodeOK || !strings.Contains(string(env.Data), `"total":3`) {
		t.Fatalf("期望分页行程列表，实际 %d/%s", rec.Code, env.Data)
	}
}

func TestPointsSummaryRequiresUser(t *testing.T) {
	h := newTestRouter(nil, fakePoints{}, nil)
	rec, env := doJSON(t, h, http.MethodGet, "/api/v1/points/summary", "", nil)
	if rec.Code != http.StatusUnauthorized || env.Code != CodeUnauthorized {
		t.Fatalf("期望 401/UNAUTHORIZED，实际 %d/%s", rec.Code, env.Code)
	}
}

func TestPointsSummary(t *testing.T) {
	p := fakePoints{summary: app.PointsSummary{Balance: 1280, Available: 1280, DailyCap: 500}}
	h := newTestRouter(nil, p, nil)
	rec, env := doJSON(t, h, http.MethodGet, "/api/v1/points/summary", "",
		map[string]string{"X-User-Id": "7"})
	if rec.Code != http.StatusOK {
		t.Fatalf("状态码 = %d, body=%s", rec.Code, rec.Body.String())
	}
	var s app.PointsSummary
	if err := json.Unmarshal(env.Data, &s); err != nil {
		t.Fatalf("解析 data 失败: %v", err)
	}
	if s.Balance != 1280 || s.DailyCap != 500 {
		t.Fatalf("积分概览不符: %+v", s)
	}
}

func TestPointsSummaryRejectsHeaderFallbackInProd(t *testing.T) {
	h := NewRouter(Options{
		Logger:     discardLogger(),
		Env:        "prod",
		AuthSecret: "test-secret",
		Points:     fakePoints{},
	})
	rec, env := doJSON(t, h, http.MethodGet, "/api/v1/points/summary", "",
		map[string]string{"X-User-Id": "7"})
	if rec.Code != http.StatusUnauthorized || env.Code != CodeUnauthorized {
		t.Fatalf("生产环境应拒绝 X-User-Id，实际 %d/%s", rec.Code, env.Code)
	}
}

// ---------- 订单 ----------

func TestCreateOrderInsufficientPoints(t *testing.T) {
	o := fakeOrders{create: func(app.CreateOrderInput) (app.Order, bool, error) {
		return app.Order{}, false, app.ErrInsufficientPoints
	}}
	h := newTestRouter(nil, nil, o)
	rec, env := doJSON(t, h, http.MethodPost, "/api/v1/orders",
		`{"productId":1,"receiver":"本人"}`, map[string]string{"X-User-Id": "7"})
	if rec.Code != http.StatusConflict || env.Code != CodeInsufficientPoints {
		t.Fatalf("期望 409/INSUFFICIENT_POINTS，实际 %d/%s", rec.Code, env.Code)
	}
}

func TestCreateOrderCreated(t *testing.T) {
	o := fakeOrders{create: func(in app.CreateOrderInput) (app.Order, bool, error) {
		if in.UserID != 7 || in.ProductID != 1 || in.Receiver != "本人" {
			t.Fatalf("透传参数不符: %+v", in)
		}
		return app.Order{ID: 1, OrderNo: "RO1", Status: "PENDING"}, false, nil
	}}
	h := newTestRouter(nil, nil, o)
	rec, env := doJSON(t, h, http.MethodPost, "/api/v1/orders",
		`{"productId":1}`, map[string]string{"X-User-Id": "7"})
	if rec.Code != http.StatusCreated || env.Code != CodeOK {
		t.Fatalf("期望 201/OK，实际 %d/%s", rec.Code, env.Code)
	}
}

func TestCreateOrderIdempotentReplay(t *testing.T) {
	o := fakeOrders{create: func(app.CreateOrderInput) (app.Order, bool, error) {
		return app.Order{ID: 1, OrderNo: "RO1"}, true, nil
	}}
	h := newTestRouter(nil, nil, o)
	rec, _ := doJSON(t, h, http.MethodPost, "/api/v1/orders",
		`{"productId":1}`, map[string]string{"X-User-Id": "7", "Idempotency-Key": "k-1"})
	if rec.Code != http.StatusOK {
		t.Fatalf("幂等重放应返回 200，实际 %d", rec.Code)
	}
}

func TestCreateOrderBadBody(t *testing.T) {
	h := newTestRouter(nil, nil, fakeOrders{})
	rec, env := doJSON(t, h, http.MethodPost, "/api/v1/orders", `{`,
		map[string]string{"X-User-Id": "7"})
	if rec.Code != http.StatusBadRequest || env.Code != CodeInvalidArgument {
		t.Fatalf("期望 400/INVALID_ARGUMENT，实际 %d/%s", rec.Code, env.Code)
	}
}

func TestCancelOrderIllegalTransition(t *testing.T) {
	o := fakeOrders{cancel: func(_, _ int64, _ string) (app.Order, error) {
		return app.Order{}, app.ErrIllegalTransition
	}}
	h := newTestRouter(nil, nil, o)
	rec, env := doJSON(t, h, http.MethodPost, "/api/v1/orders/3/cancel", `{"reason":"不要了"}`,
		map[string]string{"X-User-Id": "7"})
	if rec.Code != http.StatusConflict || env.Code != CodeIllegalTransition {
		t.Fatalf("期望 409/ILLEGAL_TRANSITION，实际 %d/%s", rec.Code, env.Code)
	}
}

func TestOrderNotFound(t *testing.T) {
	o := fakeOrders{cancel: func(_, _ int64, _ string) (app.Order, error) {
		return app.Order{}, app.ErrNotFound
	}}
	h := newTestRouter(nil, nil, o)
	rec, env := doJSON(t, h, http.MethodPost, "/api/v1/orders/99/cancel", "", map[string]string{"X-User-Id": "7"})
	if rec.Code != http.StatusNotFound || env.Code != CodeNotFound {
		t.Fatalf("期望 404/NOT_FOUND，实际 %d/%s", rec.Code, env.Code)
	}
}

func TestInternalErrorHidden(t *testing.T) {
	o := fakeOrders{create: func(app.CreateOrderInput) (app.Order, bool, error) {
		return app.Order{}, false, errors.New("db down")
	}}
	h := newTestRouter(nil, nil, o)
	rec, env := doJSON(t, h, http.MethodPost, "/api/v1/orders", `{"productId":1}`,
		map[string]string{"X-User-Id": "7"})
	if rec.Code != http.StatusInternalServerError || env.Code != CodeInternal {
		t.Fatalf("期望 500/INTERNAL，实际 %d/%s", rec.Code, env.Code)
	}
	if strings.Contains(env.Message, "db down") {
		t.Fatalf("内部错误细节不应外泄: %s", env.Message)
	}
}

func TestPointsDailyCarbon(t *testing.T) {
	p := fakePoints{daily: []app.DailyCarbonPoint{
		{Date: "2026-11-23", Grams: 0},
		{Date: "2026-11-24", Grams: 483.2},
	}}
	h := newTestRouter(nil, p, nil)
	rec, env := doJSON(t, h, http.MethodGet, "/api/v1/points/daily-carbon?days=7", "",
		map[string]string{"X-User-Id": "7"})
	if rec.Code != http.StatusOK || env.Code != CodeOK {
		t.Fatalf("期望 200/OK，实际 %d/%s", rec.Code, env.Code)
	}
	var list []app.DailyCarbonPoint
	if err := json.Unmarshal(env.Data, &list); err != nil {
		t.Fatalf("解析 data 失败: %v", err)
	}
	if len(list) != 2 || list[1].Grams != 483.2 {
		t.Fatalf("daily-carbon 返回不符: %+v", list)
	}
}

func TestPointsDailyCarbonRejectsExcessiveDays(t *testing.T) {
	h := newTestRouter(nil, fakePoints{}, nil)
	rec, env := doJSON(t, h, http.MethodGet, "/api/v1/points/daily-carbon?days=366", "",
		map[string]string{"X-User-Id": "7"})
	if rec.Code != http.StatusBadRequest || env.Code != CodeInvalidArgument {
		t.Fatalf("期望 400/INVALID_ARGUMENT，实际 %d/%s", rec.Code, env.Code)
	}
}
