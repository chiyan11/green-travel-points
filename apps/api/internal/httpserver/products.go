package httpserver

import (
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	"greentravelpoints/internal/app"
)

type catalogHandler struct {
	store  app.CatalogStore
	logger *slog.Logger
}

// list 返回商品列表：?page=&pageSize=&category=&keyword=&onlyInStock=true
func (h catalogHandler) list(w http.ResponseWriter, r *http.Request) {
	page, pageSize := parsePaging(r)
	q := app.ProductQuery{
		Page:        page,
		PageSize:    pageSize,
		Category:    strings.TrimSpace(r.URL.Query().Get("category")),
		Keyword:     strings.TrimSpace(r.URL.Query().Get("keyword")),
		OnlyInStock: r.URL.Query().Get("onlyInStock") == "true",
	}
	res, err := h.store.ListProducts(r.Context(), q)
	if err != nil {
		failFromStore(w, h.logger, err)
		return
	}
	OK(w, res)
}

// get 返回商品详情（含 SKU 列表）。
func (h catalogHandler) get(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || id <= 0 {
		Fail(w, http.StatusBadRequest, CodeInvalidArgument, "商品 ID 非法")
		return
	}
	p, err := h.store.GetProduct(r.Context(), id)
	if err != nil {
		failFromStore(w, h.logger, err)
		return
	}
	OK(w, p)
}
