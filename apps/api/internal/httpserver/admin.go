package httpserver

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	"greentravelpoints/internal/app"
)

type adminHandler struct {
	store  app.AdminStore
	logger *slog.Logger
}

func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) bool {
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	if err := dec.Decode(dst); err != nil {
		Fail(w, http.StatusBadRequest, CodeInvalidArgument, "请求体不是合法 JSON")
		return false
	}
	return true
}

// listProducts 后台商品列表：GET /api/v1/admin/products?status=&keyword=&page=
func (h adminHandler) listProducts(w http.ResponseWriter, r *http.Request) {
	page, pageSize := parsePaging(r)
	res, err := h.store.ListAllProducts(r.Context(), app.AdminProductQuery{
		Page:     page,
		PageSize: pageSize,
		Status:   strings.TrimSpace(r.URL.Query().Get("status")),
		Keyword:  strings.TrimSpace(r.URL.Query().Get("keyword")),
	})
	if err != nil {
		failFromStore(w, h.logger, err)
		return
	}
	OK(w, res)
}

// createProduct 新建商品：POST /api/v1/admin/products
func (h adminHandler) createProduct(w http.ResponseWriter, r *http.Request) {
	var req app.AdminProductInput
	if !decodeJSON(w, r, &req) {
		return
	}
	req.Name = strings.TrimSpace(req.Name)
	req.Category = strings.TrimSpace(req.Category)
	if req.MerchantID <= 0 || req.Name == "" || req.Category == "" {
		Fail(w, http.StatusBadRequest, CodeInvalidArgument, "merchantId / name / category 必填")
		return
	}
	if len(req.SKUs) == 0 {
		Fail(w, http.StatusBadRequest, CodeInvalidArgument, "至少需要一个 SKU")
		return
	}
	for _, sku := range req.SKUs {
		if strings.TrimSpace(sku.Name) == "" || sku.PointsPrice <= 0 || sku.Stock < 0 {
			Fail(w, http.StatusBadRequest, CodeInvalidArgument, "SKU 名称必填、pointsPrice > 0、stock ≥ 0")
			return
		}
	}
	p, err := h.store.CreateProduct(r.Context(), req)
	if err != nil {
		failFromStore(w, h.logger, err)
		return
	}
	WriteJSON(w, http.StatusCreated, Envelope{Code: CodeOK, Message: "created", Data: p})
}

// updateProduct 更新商品（含上下架）：PATCH /api/v1/admin/products/{id}
func (h adminHandler) updateProduct(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || id <= 0 {
		Fail(w, http.StatusBadRequest, CodeInvalidArgument, "商品 ID 非法")
		return
	}
	var req app.ProductUpdate
	if !decodeJSON(w, r, &req) {
		return
	}
	p, err := h.store.UpdateProduct(r.Context(), id, req)
	if err != nil {
		failFromStore(w, h.logger, err)
		return
	}
	OK(w, p)
}

type stockRequest struct {
	Stock int `json:"stock"`
}

// setStock 设置 SKU 库存：PUT /api/v1/admin/skus/{id}/stock
func (h adminHandler) setStock(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || id <= 0 {
		Fail(w, http.StatusBadRequest, CodeInvalidArgument, "SKU ID 非法")
		return
	}
	var req stockRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	if req.Stock < 0 {
		Fail(w, http.StatusBadRequest, CodeInvalidArgument, "stock 不能为负")
		return
	}
	sku, err := h.store.SetSKUStock(r.Context(), id, req.Stock)
	if err != nil {
		failFromStore(w, h.logger, err)
		return
	}
	OK(w, sku)
}

// listOrders 后台订单列表：GET /api/v1/admin/orders?status=&page=
func (h adminHandler) listOrders(w http.ResponseWriter, r *http.Request) {
	page, pageSize := parsePaging(r)
	res, err := h.store.ListAllOrders(r.Context(), app.AdminOrderQuery{
		Page:     page,
		PageSize: pageSize,
		Status:   strings.TrimSpace(r.URL.Query().Get("status")),
	})
	if err != nil {
		failFromStore(w, h.logger, err)
		return
	}
	OK(w, res)
}

// transition 按状态机流转订单：POST /api/v1/admin/orders/{id}/{action}
// action ∈ confirm(→PAID) / ship(→SHIPPED) / complete(→COMPLETED)
func (h adminHandler) transition(action, target string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
		if err != nil || id <= 0 {
			Fail(w, http.StatusBadRequest, CodeInvalidArgument, "订单 ID 非法")
			return
		}
		var body struct {
			Reason string `json:"reason"`
		}
		if r.Body != nil {
			_ = json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&body)
		}
		order, err := h.store.TransitionOrder(r.Context(), id, target, body.Reason)
		if err != nil {
			failFromStore(w, h.logger, err)
			return
		}
		OK(w, order)
	}
}

// reconcile 积分对账：GET /api/v1/admin/points/reconcile
func (h adminHandler) reconcile(w http.ResponseWriter, r *http.Request) {
	res, err := h.store.ReconcilePoints(r.Context())
	if err != nil {
		failFromStore(w, h.logger, err)
		return
	}
	OK(w, res)
}
