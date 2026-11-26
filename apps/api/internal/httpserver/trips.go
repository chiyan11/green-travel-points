package httpserver

import (
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"greentravelpoints/internal/app"
)

type TripHandler struct {
	store  app.TripStore
	logger *slog.Logger
}

func NewTripHandler(store app.TripStore, logger *slog.Logger) *TripHandler {
	return &TripHandler{store: store, logger: logger}
}

func (h *TripHandler) Routes(r chi.Router) {
	r.Get("/trips", h.List)
	r.Post("/trips", h.Create)
}

type tripRequest struct {
	Mode       string  `json:"mode"`
	DistanceKm float64 `json:"distanceKm"`
	TraveledAt string  `json:"traveledAt"`
}

func (h *TripHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID, _ := r.Context().Value(authContextKey{}).(int64)
	var in tripRequest
	if err := decodeJSONBody(w, r, &in); err != nil {
		failDecode(w, err)
		return
	}
	if strings.TrimSpace(in.Mode) == "" {
		Fail(w, http.StatusBadRequest, CodeInvalidArgument, "请求参数非法")
		return
	}
	traveledAt, err := time.Parse(time.RFC3339, in.TraveledAt)
	if err != nil {
		Fail(w, http.StatusBadRequest, CodeInvalidArgument, "traveledAt 必须是 ISO 8601 时间")
		return
	}
	if err := validateTraveledAt(traveledAt, time.Now()); err != nil {
		Fail(w, http.StatusBadRequest, CodeInvalidArgument, err.Error())
		return
	}
	if h.store == nil {
		Fail(w, http.StatusServiceUnavailable, CodeUnavailable, "数据库不可用")
		return
	}

	trip, err := h.store.CreateTrip(r.Context(), app.CreateTripInput{
		UserID: userID, Mode: in.Mode, DistanceKm: in.DistanceKm, TraveledAt: traveledAt,
	})
	if err != nil {
		if errors.Is(err, app.ErrInvalidInput) {
			Fail(w, http.StatusBadRequest, CodeInvalidArgument, "请求参数非法")
			return
		}
		FailInternal(w, h.logger, err)
		return
	}
	OK(w, trip)
}

func (h *TripHandler) List(w http.ResponseWriter, r *http.Request) {
	userID, _ := r.Context().Value(authContextKey{}).(int64)
	if h.store == nil {
		Fail(w, http.StatusServiceUnavailable, CodeUnavailable, "数据库不可用")
		return
	}
	page, pageSize := parsePaging(r)
	res, err := h.store.ListTrips(r.Context(), app.TripQuery{UserID: userID, Page: page, PageSize: pageSize})
	if err != nil {
		FailInternal(w, h.logger, err)
		return
	}
	OK(w, res)
}

func validateTraveledAt(traveledAt, now time.Time) error {
	if traveledAt.After(now) || traveledAt.Before(now.AddDate(0, 0, -7)) {
		return errors.New("traveledAt 必须在当前时间前 7 天内")
	}
	return nil
}
