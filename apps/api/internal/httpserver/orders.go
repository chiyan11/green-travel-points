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

type orderHandler struct {
	store  app.OrderStore
	logger *slog.Logger
}

type createOrderRequest struct {
	ProductID int64  `json:"productId"`
	SKUId     int64  `json:"skuId"`
	Receiver  string `json:"receiver"`
}

type cancelOrderRequest struct {
	Reason string `json:"reason"`
}

// create 兑换下单：POST /api/v1/orders
// 请求头可选 Idempotency-Key，重复请求返回原订单（BR-15）。
func (h orderHandler) create(w http.ResponseWriter, r *http.Request) {
	var req createOrderRequest
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	if err := dec.Decode(&req); err != nil {
		Fail(w, http.StatusBadRequest, CodeInvalidArgument, "请求体不是合法 JSON")
		return
	}
	if req.ProductID <= 0 {
		Fail(w, http.StatusBadRequest, CodeInvalidArgument, "productId 必填")
		return
	}
	receiver := strings.TrimSpace(req.Receiver)
	if receiver == "" {
		receiver = "本人"
	}
	order, replayed, err := h.store.CreateOrder(r.Context(), app.CreateOrderInput{
		UserID:         UserID(r.Context()),
		ProductID:      req.ProductID,
		SKUId:          req.SKUId,
		Receiver:       receiver,
		IdempotencyKey: strings.TrimSpace(r.Header.Get("Idempotency-Key")),
	})
	if err != nil {
		failFromStore(w, h.logger, err)
		return
	}
	if replayed {
		OK(w, order)
		return
	}
	WriteJSON(w, http.StatusCreated, Envelope{Code: CodeOK, Message: "created", Data: order})
}

// list 返回当前用户订单：GET /api/v1/orders?page=&pageSize=&status=
func (h orderHandler) list(w http.ResponseWriter, r *http.Request) {
	page, pageSize := parsePaging(r)
	res, err := h.store.ListOrders(r.Context(), app.OrderQuery{
		UserID:   UserID(r.Context()),
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

// cancel 取消订单：POST /api/v1/orders/{id}/cancel
func (h orderHandler) cancel(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || id <= 0 {
		Fail(w, http.StatusBadRequest, CodeInvalidArgument, "订单 ID 非法")
		return
	}
	var req cancelOrderRequest
	if r.Body != nil {
		_ = json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&req)
	}
	order, err := h.store.CancelOrder(r.Context(), UserID(r.Context()), id, req.Reason)
	if err != nil {
		failFromStore(w, h.logger, err)
		return
	}
	OK(w, order)
}
