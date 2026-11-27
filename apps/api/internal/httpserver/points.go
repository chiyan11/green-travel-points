package httpserver

import (
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"greentravelpoints/internal/app"
)

// maxCarbonDays 限制 daily-carbon 的查询窗口，避免一次拉取过多年份的逐日数据。
const maxCarbonDays = 365

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
	days := 7
	if values, present := r.URL.Query()["days"]; present {
		var err error
		days, err = strconv.Atoi(values[0])
		if err != nil || days < 1 || days > maxCarbonDays {
			Fail(w, http.StatusBadRequest, CodeInvalidArgument, "days 取值范围 1–365")
			return
		}
	}
	list, err := h.store.DailyCarbon(r.Context(), UserID(r.Context()), days)
	if err != nil {
		failFromStore(w, h.logger, err)
		return
	}
	OK(w, list)
}
