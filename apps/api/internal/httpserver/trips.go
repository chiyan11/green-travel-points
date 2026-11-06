package httpserver

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"greentravelpoints/internal/domain"
	"greentravelpoints/internal/eventbus"
)

type TripHandler struct {
	db     *sql.DB
	logger *slog.Logger
}

func NewTripHandler(db *sql.DB, logger *slog.Logger) *TripHandler {
	return &TripHandler{db: db, logger: logger}
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

type tripResponse struct {
	ID               string    `json:"id"`
	Mode             string    `json:"mode"`
	DistanceKm       float64   `json:"distanceKm"`
	TraveledAt       time.Time `json:"traveledAt"`
	CarbonSavedGrams float64   `json:"carbonSavedGrams"`
	RawPoints        int64     `json:"rawPoints"`
	GrantedPoints    int64     `json:"grantedPoints"`
	CappedByDaily    bool      `json:"cappedByDaily"`
	Status           string    `json:"status"`
}

func (h *TripHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID, _ := r.Context().Value(authContextKey{}).(int64)
	var in tripRequest
	if err := decodeJSON(r, &in); err != nil || strings.TrimSpace(in.Mode) == "" {
		Fail(w, http.StatusBadRequest, CodeInvalidArgument, "请求参数非法")
		return
	}
	traveledAt, err := time.Parse(time.RFC3339, in.TraveledAt)
	if err != nil {
		Fail(w, http.StatusBadRequest, CodeInvalidArgument, "traveledAt 必须是 ISO 8601 时间")
		return
	}
	if h.db == nil {
		Fail(w, http.StatusServiceUnavailable, CodeUnavailable, "数据库不可用")
		return
	}

	ctx := r.Context()
	var emission domain.Emission
	var rule domain.PointsRule
	var carbonPerPoint, dailyCap int64
	var minDistance, maxDistance float64
	err = h.db.QueryRowContext(ctx,
		"SELECT ef_base, ef_actual FROM travel_mode WHERE code = ? AND enabled = TRUE",
		in.Mode).Scan(&emission.Base, &emission.Actual)
	if err != nil {
		Fail(w, http.StatusBadRequest, CodeInvalidArgument, "未知或已停用的出行方式")
		return
	}
	if err = h.loadRule(ctx, &carbonPerPoint, &dailyCap, &minDistance, &maxDistance); err != nil {
		FailInternal(w, h.logger, err)
		return
	}
	rule = domain.PointsRule{
		GramsPerPoint: carbonPerPoint, DailyCap: dailyCap,
		MinDistanceKm: minDistance, MaxDistanceKm: maxDistance,
	}
	tripHash := hashTrip(userID, in.Mode, in.DistanceKm, traveledAt)

	tx, err := h.db.BeginTx(ctx, nil)
	if err != nil {
		FailInternal(w, h.logger, err)
		return
	}
	var existing tripResponse
	if err := h.scanTrip(ctx, tx, userID, tripHash, &existing); err == nil {
		_ = tx.Commit()
		OK(w, existing)
		return
	}

	var alreadyToday int64
	dayStart := time.Date(traveledAt.Year(), traveledAt.Month(), traveledAt.Day(), 0, 0, 0, 0, traveledAt.Location())
	err = tx.QueryRowContext(ctx, `
		SELECT COALESCE(SUM(pt.amount), 0)
		FROM point_txn pt
		JOIN point_account pa ON pa.id = pt.account_id
		WHERE pa.user_id = ? AND pt.txn_type = 'TRAVEL' AND pt.occurred_at >= ?`, userID, dayStart).Scan(&alreadyToday)
	if err != nil {
		_ = tx.Rollback()
		FailInternal(w, h.logger, err)
		return
	}
	grant, err := domain.EvaluateTrip(in.DistanceKm, emission, rule, alreadyToday)
	if err != nil {
		_ = tx.Rollback()
		Fail(w, http.StatusBadRequest, CodeInvalidArgument, err.Error())
		return
	}
	res, err := tx.ExecContext(ctx, `
		INSERT IGNORE INTO trip_record
			(user_id, mode_code, distance_km, started_at, ended_at, trip_hash)
		VALUES (?, ?, ?, ?, ?, ?)`,
		userID, in.Mode, in.DistanceKm, traveledAt, traveledAt, tripHash)
	if err != nil {
		_ = tx.Rollback()
		FailInternal(w, h.logger, err)
		return
	}
	tripID, err := res.LastInsertId()
	if err != nil {
		_ = tx.Rollback()
		FailInternal(w, h.logger, err)
		return
	}
	if tripID == 0 {
		if err := h.scanTrip(ctx, tx, userID, tripHash, &existing); err != nil {
			_ = tx.Rollback()
			FailInternal(w, h.logger, err)
			return
		}
		_ = tx.Commit()
		OK(w, existing)
		return
	}
	ruleSnapshot := fmt.Sprintf("carbon_per_point=%d;daily_cap=%d;raw_points=%d;granted_points=%d",
		carbonPerPoint, dailyCap, grant.RawPoints, grant.GrantedPoints)
	if _, err = tx.ExecContext(ctx, `
		INSERT INTO carbon_record (trip_id, ef_base, ef_actual, carbon_saved_g, rule_snapshot)
		VALUES (?, ?, ?, ?, ?)`,
		tripID, emission.Base, emission.Actual, grant.CarbonSavedGrams, ruleSnapshot); err != nil {
		_ = tx.Rollback()
		FailInternal(w, h.logger, err)
		return
	}
	if _, err = tx.ExecContext(ctx, "INSERT IGNORE INTO point_account (user_id) VALUES (?)", userID); err != nil {
		_ = tx.Rollback()
		FailInternal(w, h.logger, err)
		return
	}
	var accountID, balance int64
	if err = tx.QueryRowContext(ctx,
		"SELECT id, balance FROM point_account WHERE user_id = ? FOR UPDATE", userID).
		Scan(&accountID, &balance); err != nil {
		_ = tx.Rollback()
		FailInternal(w, h.logger, err)
		return
	}
	if grant.GrantedPoints > 0 {
		balance += grant.GrantedPoints
		if _, err = tx.ExecContext(ctx, `
			INSERT INTO point_txn
				(account_id, txn_type, amount, balance_after, biz_ref_type, biz_ref_id)
			VALUES (?, 'TRAVEL', ?, ?, 'trip_record', ?)`,
			accountID, grant.GrantedPoints, balance, tripID); err != nil {
			_ = tx.Rollback()
			FailInternal(w, h.logger, err)
			return
		}
	}
	if _, err = tx.ExecContext(ctx,
		"UPDATE point_account SET balance = ?, total_saved_g = total_saved_g + ?, version = version + 1 WHERE id = ?",
		balance, grant.CarbonSavedGrams, accountID); err != nil {
		_ = tx.Rollback()
		FailInternal(w, h.logger, err)
		return
	}
	payload, _ := json.Marshal(map[string]any{
		"tripId": tripID, "userId": userID, "carbonSavedGrams": grant.CarbonSavedGrams,
		"grantedPoints": grant.GrantedPoints,
	})
	if _, err = tx.ExecContext(ctx, `
		INSERT INTO event_outbox (topic, aggregate_type, aggregate_id, payload)
		VALUES (?, 'trip', ?, ?)`, eventbus.TopicTripVerified, strconv.FormatInt(tripID, 10), payload); err != nil {
		_ = tx.Rollback()
		FailInternal(w, h.logger, err)
		return
	}
	if err = tx.Commit(); err != nil {
		FailInternal(w, h.logger, err)
		return
	}
	OK(w, tripResponse{
		ID: userTripID(tripID), Mode: in.Mode, DistanceKm: in.DistanceKm,
		TraveledAt: traveledAt, CarbonSavedGrams: grant.CarbonSavedGrams,
		RawPoints: grant.RawPoints, GrantedPoints: grant.GrantedPoints,
		CappedByDaily: grant.CappedByDaily, Status: "VERIFIED",
	})
}

func (h *TripHandler) List(w http.ResponseWriter, r *http.Request) {
	userID, _ := r.Context().Value(authContextKey{}).(int64)
	if h.db == nil {
		Fail(w, http.StatusServiceUnavailable, CodeUnavailable, "数据库不可用")
		return
	}
	rows, err := h.db.QueryContext(r.Context(), `
		SELECT tr.id, tr.mode_code, tr.distance_km, tr.started_at,
		       cr.carbon_saved_g, cr.rule_snapshot, COALESCE(pt.amount, 0), tr.status
		FROM trip_record tr
		JOIN carbon_record cr ON cr.trip_id = tr.id
		LEFT JOIN point_txn pt ON pt.biz_ref_type = 'trip_record'
			AND pt.biz_ref_id = tr.id AND pt.txn_type = 'TRAVEL'
		WHERE tr.user_id = ? ORDER BY tr.started_at DESC`, userID)
	if err != nil {
		FailInternal(w, h.logger, err)
		return
	}
	defer rows.Close()
	items := make([]tripResponse, 0)
	for rows.Next() {
		var item tripResponse
		var id int64
		var snapshot string
		var granted int64
		if err := rows.Scan(&id, &item.Mode, &item.DistanceKm, &item.TraveledAt,
			&item.CarbonSavedGrams, &snapshot, &granted, &item.Status); err != nil {
			FailInternal(w, h.logger, err)
			return
		}
		item.ID = userTripID(id)
		item.RawPoints = parseRawPoints(snapshot)
		item.GrantedPoints = granted
		item.CappedByDaily = granted < item.RawPoints
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		FailInternal(w, h.logger, err)
		return
	}
	OK(w, map[string]any{"items": items, "page": 1, "pageSize": len(items), "total": len(items)})
}

func (h *TripHandler) loadRule(ctx context.Context, carbonPerPoint, dailyCap *int64, minDistance, maxDistance *float64) error {
	values := map[string]string{}
	rows, err := h.db.QueryContext(ctx,
		"SELECT config_key, config_value FROM sys_config WHERE config_key IN (?, ?, ?, ?)",
		"carbon_per_point", "daily_points_cap", "trip_min_distance_km", "trip_max_distance_km")
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var key, value string
		if err := rows.Scan(&key, &value); err != nil {
			return err
		}
		values[key] = value
	}
	if err := rows.Err(); err != nil {
		return err
	}
	var parseErr error
	if *carbonPerPoint, parseErr = strconv.ParseInt(values["carbon_per_point"], 10, 64); parseErr != nil {
		return parseErr
	}
	if *dailyCap, parseErr = strconv.ParseInt(values["daily_points_cap"], 10, 64); parseErr != nil {
		return parseErr
	}
	if *minDistance, parseErr = strconv.ParseFloat(values["trip_min_distance_km"], 64); parseErr != nil {
		return parseErr
	}
	*maxDistance, parseErr = strconv.ParseFloat(values["trip_max_distance_km"], 64)
	return parseErr
}

func (h *TripHandler) scanTrip(ctx context.Context, tx *sql.Tx, userID int64, hash string, out *tripResponse) error {
	var id int64
	var granted int64
	var snapshot string
	err := tx.QueryRowContext(ctx, `
		SELECT tr.id, tr.mode_code, tr.distance_km, tr.started_at,
		       cr.carbon_saved_g, cr.rule_snapshot,
		       COALESCE(pt.amount, 0), tr.status
		FROM trip_record tr
		JOIN carbon_record cr ON cr.trip_id = tr.id
		LEFT JOIN point_txn pt ON pt.biz_ref_type = 'trip_record'
			AND pt.biz_ref_id = tr.id AND pt.txn_type = 'TRAVEL'
		WHERE tr.user_id = ? AND tr.trip_hash = ?`,
		userID, hash).Scan(&id, &out.Mode, &out.DistanceKm, &out.TraveledAt,
		&out.CarbonSavedGrams, &snapshot, &granted, &out.Status)
	if err != nil {
		return err
	}
	out.ID = userTripID(id)
	out.GrantedPoints = granted
	out.RawPoints = parseRawPoints(snapshot)
	out.CappedByDaily = out.GrantedPoints < out.RawPoints
	return nil
}

func parseRawPoints(snapshot string) int64 {
	const prefix = "raw_points="
	start := strings.Index(snapshot, prefix)
	if start < 0 {
		return 0
	}
	start += len(prefix)
	end := strings.IndexByte(snapshot[start:], ';')
	if end >= 0 {
		snapshot = snapshot[start : start+end]
	} else {
		snapshot = snapshot[start:]
	}
	n, _ := strconv.ParseInt(snapshot, 10, 64)
	return n
}

func hashTrip(userID int64, mode string, distance float64, at time.Time) string {
	raw := fmt.Sprintf("%d|%s|%.3f|%s", userID, mode, distance, at.UTC().Format(time.RFC3339Nano))
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

func userTripID(id int64) string { return fmt.Sprintf("trip_%d", id) }
