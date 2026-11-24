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
	dec.DisallowUnknownFields() // 多余字段直接 400，避免前后端字段漂移悄悄通过
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

// expirePoints 手动触发积分过期清算：POST /api/v1/admin/points/expire
// 把 expires_at <= NOW() 的 ACTIVE 批次置为 EXPIRED、写 EXPIRE 流水并扣余额；
// 演示时也用它把过期数据造出来的效果立刻落地。
func (h adminHandler) expirePoints(w http.ResponseWriter, r *http.Request) {
	res, err := h.store.ExpirePoints(r.Context())
	if err != nil {
		failFromStore(w, h.logger, err)
		return
	}
	OK(w, res)
}

// listConfig 系统参数列表：GET /api/v1/admin/config
// 返回 sys_config 全部参数（键 / 值 / 值类型 / 说明），值一律为字符串。
func (h adminHandler) listConfig(w http.ResponseWriter, r *http.Request) {
	items, err := h.store.ListConfig(r.Context())
	if err != nil {
		failFromStore(w, h.logger, err)
		return
	}
	OK(w, items)
}

// updateConfig 批量修改系统参数：PUT /api/v1/admin/config
// 请求体 {"items":[{"key":"daily_points_cap","value":"10"}]}，整批要么全成、要么全回滚；
// 返回更新后的全量参数，调用方可直接刷新界面。
func (h adminHandler) updateConfig(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Items []app.ConfigUpdate `json:"items"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	if len(req.Items) == 0 {
		Fail(w, http.StatusBadRequest, CodeInvalidArgument, "items 不能为空")
		return
	}
	for _, it := range req.Items {
		if strings.TrimSpace(it.Key) == "" || strings.TrimSpace(it.Value) == "" {
			Fail(w, http.StatusBadRequest, CodeInvalidArgument, "key / value 不能为空")
			return
		}
	}
	res, err := h.store.UpdateConfig(r.Context(), UserID(r.Context()), req.Items)
	if err != nil {
		failFromStore(w, h.logger, err)
		return
	}
	OK(w, res)
}

// updateTravelMode 更新出行方式：PUT /api/v1/admin/travel-modes/{code}
// 可改 name / efBase / efActual / enabled；nil 字段不动。
func (h adminHandler) updateTravelMode(w http.ResponseWriter, r *http.Request) {
	var req app.TravelModeUpdate
	if !decodeJSON(w, r, &req) {
		return
	}
	if err := app.ValidateTravelModeUpdate(req); err != nil {
		Fail(w, http.StatusBadRequest, CodeInvalidArgument,
			"至少要改一个字段；name 不能为空；efBase > 0；efActual ≥ 0")
		return
	}
	mode, err := h.store.UpdateTravelMode(r.Context(), UserID(r.Context()), chi.URLParam(r, "code"), req)
	if err != nil {
		failFromStore(w, h.logger, err)
		return
	}
	OK(w, mode)
}

// adjustPoints 人工调增 / 调减积分：POST /api/v1/admin/points/adjust
// 请求体 {"userId":1001,"amount":-50,"reason":"刷分扣回"}；reason 必填（BR-14），变动写 audit_log。
func (h adminHandler) adjustPoints(w http.ResponseWriter, r *http.Request) {
	var req app.AdjustPointsInput
	if !decodeJSON(w, r, &req) {
		return
	}
	req.Reason = strings.TrimSpace(req.Reason)
	if req.UserID <= 0 || req.Amount == 0 || req.Reason == "" {
		Fail(w, http.StatusBadRequest, CodeInvalidArgument, "userId / amount（非 0）/ reason 均为必填")
		return
	}
	res, err := h.store.AdjustPoints(r.Context(), UserID(r.Context()), req)
	if err != nil {
		failFromStore(w, h.logger, err)
		return
	}
	OK(w, res)
}

// auditLogs 审计日志查询：GET /api/v1/admin/audit-logs?operatorId=&action=&targetType=&targetKey=&page=
func (h adminHandler) auditLogs(w http.ResponseWriter, r *http.Request) {
	page, pageSize := parsePaging(r)
	operatorID, _ := strconv.ParseInt(strings.TrimSpace(r.URL.Query().Get("operatorId")), 10, 64)
	res, err := h.store.ListAuditLogs(r.Context(), app.AuditQuery{
		OperatorID: operatorID,
		Action:     strings.TrimSpace(r.URL.Query().Get("action")),
		TargetType: strings.TrimSpace(r.URL.Query().Get("targetType")),
		TargetKey:  strings.TrimSpace(r.URL.Query().Get("targetKey")),
		Page:       page,
		PageSize:   pageSize,
	})
	if err != nil {
		failFromStore(w, h.logger, err)
		return
	}
	OK(w, res)
}
