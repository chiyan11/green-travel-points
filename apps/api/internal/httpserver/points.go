package httpserver

import (
	"log/slog"
	"net/http"
	"strings"

	"greentravelpoints/internal/app"
)

type pointsHandler struct {
	store  app.PointsStore
	logger *slog.Logger
}

// summary 返回积分概览：GET /api/v1/points/summary
func (h pointsHandler) summary(w http.ResponseWriter, r *http.Request) {
	summary, err := h.store.Summary(r.Context(), UserID(r.Context()))
	if err != nil {
		failFromStore(w, h.logger, err)
		return
	}
	OK(w, summary)
}

// transactions 返回积分流水：
// GET /api/v1/points/transactions?page=&pageSize=&type=TRAVEL,REDEEM
func (h pointsHandler) transactions(w http.ResponseWriter, r *http.Request) {
	page, pageSize := parsePaging(r)
	types := make([]string, 0, 3)
	for _, raw := range strings.Split(r.URL.Query().Get("type"), ",") {
		if t := strings.TrimSpace(raw); t != "" {
			types = append(types, t)
		}
	}
	res, err := h.store.ListTransactions(r.Context(), app.TxnQuery{
		UserID:   UserID(r.Context()),
		Page:     page,
		PageSize: pageSize,
		Types:    types,
	})
	if err != nil {
		failFromStore(w, h.logger, err)
		return
	}
	OK(w, res)
}
