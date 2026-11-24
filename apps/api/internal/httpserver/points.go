package httpserver

import (
	"log/slog"
	"net/http"
	"strconv"
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

// dailyCarbon 返回最近 N 天逐日碳减排量：
// GET /api/v1/points/daily-carbon?days=7
func (h pointsHandler) dailyCarbon(w http.ResponseWriter, r *http.Request) {
	days, _ := strconv.Atoi(r.URL.Query().Get("days"))
	if days <= 0 {
		days = 7
	}
	list, err := h.store.DailyCarbon(r.Context(), UserID(r.Context()), days)
	if err != nil {
		failFromStore(w, h.logger, err)
		return
	}
	OK(w, list)
}
