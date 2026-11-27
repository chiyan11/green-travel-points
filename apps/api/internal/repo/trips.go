package repo

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

	"greentravelpoints/internal/app"
	"greentravelpoints/internal/domain"
	"greentravelpoints/internal/eventbus"
)

func (s *Store) CreateTrip(ctx context.Context, in app.CreateTripInput) (app.Trip, error) {
	var emission domain.Emission
	if err := s.db.QueryRowContext(ctx,
		"SELECT ef_base, ef_actual FROM travel_mode WHERE code = ? AND enabled = TRUE", in.Mode,
	).Scan(&emission.Base, &emission.Actual); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return app.Trip{}, app.ErrInvalidInput
		}
		return app.Trip{}, err
	}
	rule, expireDays, err := s.tripRule(ctx)
	if err != nil {
		return app.Trip{}, err
	}
	tripHash := hashTrip(in.UserID, in.Mode, in.DistanceKm, in.TraveledAt)
	if _, err := s.db.ExecContext(ctx, "INSERT IGNORE INTO point_account (user_id) VALUES (?)", in.UserID); err != nil {
		return app.Trip{}, err
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return app.Trip{}, err
	}
	defer func() { _ = tx.Rollback() }()
	var accountID, balance int64
	if err := tx.QueryRowContext(ctx, "SELECT id, balance FROM point_account WHERE user_id = ? FOR UPDATE", in.UserID).
		Scan(&accountID, &balance); err != nil {
		return app.Trip{}, err
	}
	if existing, err := scanTrip(ctx, tx, in.UserID, tripHash); err == nil {
		if err := tx.Commit(); err != nil {
			return app.Trip{}, err
		}
		return existing, nil
	} else if !errors.Is(err, sql.ErrNoRows) {
		return app.Trip{}, err
	}

	now := time.Now()
	dayStart := startOfDay(now)
	var alreadyToday int64
	if err := tx.QueryRowContext(ctx, `
		SELECT COALESCE(SUM(pt.amount), 0)
		FROM point_txn pt JOIN point_account pa ON pa.id = pt.account_id
		WHERE pa.user_id = ? AND pt.txn_type = 'TRAVEL' AND pt.occurred_at >= ? AND pt.occurred_at < ?`,
		in.UserID, dayStart, dayStart.AddDate(0, 0, 1)).Scan(&alreadyToday); err != nil {
		return app.Trip{}, err
	}
	grant, err := domain.EvaluateTrip(in.DistanceKm, emission, rule, alreadyToday)
	if err != nil {
		return app.Trip{}, fmt.Errorf("%w: %v", app.ErrInvalidInput, err)
	}
	res, err := tx.ExecContext(ctx, `
		INSERT INTO trip_record (user_id, mode_code, distance_km, started_at, ended_at, trip_hash)
		VALUES (?, ?, ?, ?, ?, ?)`, in.UserID, in.Mode, in.DistanceKm, in.TraveledAt, in.TraveledAt, tripHash)
	if err != nil {
		if isDuplicate(err) {
			existing, lookupErr := scanTrip(ctx, tx, in.UserID, tripHash)
			if lookupErr == nil {
				if err := tx.Commit(); err != nil {
					return app.Trip{}, err
				}
				return existing, nil
			}
			if !errors.Is(lookupErr, sql.ErrNoRows) {
				return app.Trip{}, lookupErr
			}
		}
		return app.Trip{}, err
	}
	tripID, err := res.LastInsertId()
	if err != nil {
		return app.Trip{}, err
	}
	ruleSnapshot, err := json.Marshal(tripSnapshot{
		CarbonPerPoint: rule.GramsPerPoint, DailyCap: rule.DailyCap,
		RawPoints: grant.RawPoints, GrantedPoints: grant.GrantedPoints,
	})
	if err != nil {
		return app.Trip{}, err
	}
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO carbon_record (trip_id, ef_base, ef_actual, carbon_saved_g, rule_snapshot)
		VALUES (?, ?, ?, ?, ?)`, tripID, emission.Base, emission.Actual, grant.CarbonSavedGrams, ruleSnapshot); err != nil {
		return app.Trip{}, err
	}
	if grant.GrantedPoints > 0 {
		balance += grant.GrantedPoints
		res, err := tx.ExecContext(ctx, `
			INSERT INTO point_txn (account_id, txn_type, amount, balance_after, biz_ref_type, biz_ref_id)
			VALUES (?, 'TRAVEL', ?, ?, 'trip_record', ?)`, accountID, grant.GrantedPoints, balance, tripID)
		if err != nil {
			return app.Trip{}, err
		}
		txnID, err := res.LastInsertId()
		if err != nil {
			return app.Trip{}, err
		}
		if err := GrantPointsWithBatch(ctx, tx, accountID, txnID, grant.GrantedPoints,
			now.AddDate(0, 0, int(expireDays))); err != nil {
			return app.Trip{}, err
		}
	}
	if _, err := tx.ExecContext(ctx,
		"UPDATE point_account SET balance = ?, total_saved_g = total_saved_g + ?, version = version + 1 WHERE id = ?",
		balance, grant.CarbonSavedGrams, accountID); err != nil {
		return app.Trip{}, err
	}
	payload, _ := json.Marshal(map[string]any{
		"tripId": tripID, "userId": in.UserID, "carbonSavedGrams": grant.CarbonSavedGrams,
		"grantedPoints": grant.GrantedPoints,
	})
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO event_outbox (topic, aggregate_type, aggregate_id, payload)
		VALUES (?, 'trip', ?, ?)`, eventbus.TopicTripVerified, strconv.FormatInt(tripID, 10), payload); err != nil {
		return app.Trip{}, err
	}
	if err := tx.Commit(); err != nil {
		return app.Trip{}, err
	}
	return app.Trip{ID: tripIDString(tripID), Mode: in.Mode, DistanceKm: in.DistanceKm, TraveledAt: in.TraveledAt,
		CarbonSavedGrams: grant.CarbonSavedGrams, RawPoints: grant.RawPoints, GrantedPoints: grant.GrantedPoints,
		CappedByDaily: grant.CappedByDaily, Status: "VERIFIED"}, nil
}

func (s *Store) ListTrips(ctx context.Context, q app.TripQuery) (app.Page[app.Trip], error) {
	var total int64
	if err := s.db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM trip_record WHERE user_id = ?", q.UserID).Scan(&total); err != nil {
		return app.Page[app.Trip]{}, err
	}
	if q.Page < 1 {
		q.Page = 1
	}
	if q.PageSize < 1 {
		q.PageSize = 20
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT tr.id, tr.mode_code, tr.distance_km, tr.started_at, cr.carbon_saved_g, cr.rule_snapshot,
		       COALESCE(pt.amount, 0), tr.status
		FROM trip_record tr JOIN carbon_record cr ON cr.trip_id = tr.id
		LEFT JOIN point_txn pt ON pt.biz_ref_type = 'trip_record' AND pt.biz_ref_id = tr.id AND pt.txn_type = 'TRAVEL'
		WHERE tr.user_id = ? ORDER BY tr.started_at DESC, tr.id DESC LIMIT ? OFFSET ?`,
		q.UserID, q.PageSize, (q.Page-1)*q.PageSize)
	if err != nil {
		return app.Page[app.Trip]{}, err
	}
	defer rows.Close()
	items := make([]app.Trip, 0, q.PageSize)
	for rows.Next() {
		var trip app.Trip
		var id int64
		var snapshot string
		if err := rows.Scan(&id, &trip.Mode, &trip.DistanceKm, &trip.TraveledAt, &trip.CarbonSavedGrams,
			&snapshot, &trip.GrantedPoints, &trip.Status); err != nil {
			return app.Page[app.Trip]{}, err
		}
		snap, err := parseTripSnapshot(snapshot)
		if err != nil {
			return app.Page[app.Trip]{}, err
		}
		trip.ID = tripIDString(id)
		trip.RawPoints = snap.RawPoints
		trip.CappedByDaily = trip.GrantedPoints < trip.RawPoints
		items = append(items, trip)
	}
	if err := rows.Err(); err != nil {
		return app.Page[app.Trip]{}, err
	}
	return app.Page[app.Trip]{Items: items, Page: q.Page, PageSize: q.PageSize, Total: total}, nil
}

func (s *Store) tripRule(ctx context.Context) (domain.PointsRule, int64, error) {
	values := map[string]string{}
	rows, err := s.db.QueryContext(ctx,
		"SELECT config_key, config_value FROM sys_config WHERE config_key IN (?, ?, ?, ?, ?)",
		"carbon_per_point", "daily_points_cap", "points_expire_days", "trip_min_distance_km", "trip_max_distance_km")
	if err != nil {
		return domain.PointsRule{}, 0, err
	}
	defer rows.Close()
	for rows.Next() {
		var key, value string
		if err := rows.Scan(&key, &value); err != nil {
			return domain.PointsRule{}, 0, err
		}
		values[key] = value
	}
	if err := rows.Err(); err != nil {
		return domain.PointsRule{}, 0, err
	}
	carbonPerPoint, err := strconv.ParseInt(values["carbon_per_point"], 10, 64)
	if err != nil {
		return domain.PointsRule{}, 0, err
	}
	dailyCap, err := strconv.ParseInt(values["daily_points_cap"], 10, 64)
	if err != nil {
		return domain.PointsRule{}, 0, err
	}
	expireDays, err := strconv.ParseInt(values["points_expire_days"], 10, 64)
	if err != nil || expireDays <= 0 {
		return domain.PointsRule{}, 0, errors.New("points_expire_days must be a positive integer")
	}
	minDistance, err := strconv.ParseFloat(values["trip_min_distance_km"], 64)
	if err != nil {
		return domain.PointsRule{}, 0, err
	}
	maxDistance, err := strconv.ParseFloat(values["trip_max_distance_km"], 64)
	if err != nil {
		return domain.PointsRule{}, 0, err
	}
	return domain.PointsRule{GramsPerPoint: carbonPerPoint, DailyCap: dailyCap,
		MinDistanceKm: minDistance, MaxDistanceKm: maxDistance}, expireDays, nil
}

func scanTrip(ctx context.Context, tx *sql.Tx, userID int64, hash string) (app.Trip, error) {
	var trip app.Trip
	var id int64
	var snapshot string
	err := tx.QueryRowContext(ctx, `
		SELECT tr.id, tr.mode_code, tr.distance_km, tr.started_at, cr.carbon_saved_g, cr.rule_snapshot,
		       COALESCE(pt.amount, 0), tr.status
		FROM trip_record tr JOIN carbon_record cr ON cr.trip_id = tr.id
		LEFT JOIN point_txn pt ON pt.biz_ref_type = 'trip_record' AND pt.biz_ref_id = tr.id AND pt.txn_type = 'TRAVEL'
		WHERE tr.user_id = ? AND tr.trip_hash = ?`, userID, hash).
		Scan(&id, &trip.Mode, &trip.DistanceKm, &trip.TraveledAt, &trip.CarbonSavedGrams,
			&snapshot, &trip.GrantedPoints, &trip.Status)
	if err != nil {
		return app.Trip{}, err
	}
	snap, err := parseTripSnapshot(snapshot)
	if err != nil {
		return app.Trip{}, err
	}
	trip.ID = tripIDString(id)
	trip.RawPoints = snap.RawPoints
	trip.CappedByDaily = trip.GrantedPoints < trip.RawPoints
	return trip, nil
}

// tripSnapshot 是 carbon_record.rule_snapshot 的 JSON 结构（迁移 000008 起）。
type tripSnapshot struct {
	CarbonPerPoint int64 `json:"carbonPerPoint"`
	DailyCap       int64 `json:"dailyCap"`
	RawPoints      int64 `json:"rawPoints"`
	GrantedPoints  int64 `json:"grantedPoints"`
}

// parseTripSnapshot 解析 rule_snapshot；开发期不保留旧拼串兼容，失败即报错暴露脏数据。
func parseTripSnapshot(raw string) (tripSnapshot, error) {
	var snap tripSnapshot
	if err := json.Unmarshal([]byte(raw), &snap); err != nil {
		return tripSnapshot{}, fmt.Errorf("解析 rule_snapshot 失败（%q）：%w", raw, err)
	}
	return snap, nil
}

func hashTrip(userID int64, mode string, distance float64, at time.Time) string {
	raw := fmt.Sprintf("%d|%s|%.6f|%s", userID, mode, distance, at.UTC().Format(time.RFC3339Nano))
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

func tripIDString(id int64) string { return fmt.Sprintf("trip_%d", id) }

func startOfDay(now time.Time) time.Time {
	return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
}

var _ app.TripStore = (*Store)(nil)
