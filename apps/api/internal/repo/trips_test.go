package repo

import (
	"context"
	"sync"
	"testing"
	"time"

	"greentravelpoints/internal/app"
	"greentravelpoints/internal/eventbus"
)

func TestStartOfDay(t *testing.T) {
	loc := time.FixedZone("CST", 8*60*60)
	now := time.Date(2026, time.November, 25, 23, 59, 59, 0, loc)
	want := time.Date(2026, time.November, 25, 0, 0, 0, 0, loc)
	if got := startOfDay(now); !got.Equal(want) || got.Location() != loc {
		t.Fatalf("startOfDay() = %v (%v), want %v (%v)", got, got.Location(), want, loc)
	}
}

func TestCreateTripConcurrentDailyCap(t *testing.T) {
	store := newTestStore(t)
	mustExec(t, store.db, "UPDATE point_txn SET occurred_at = DATE_SUB(NOW(3), INTERVAL 1 DAY) WHERE account_id = ?", testAccountID)
	mustExec(t, store.db, "UPDATE point_txn SET occurred_at = NOW(3) WHERE id = 9002")
	var wg sync.WaitGroup
	errs := make(chan error, 2)
	start := make(chan struct{})
	now := time.Now().Add(-time.Minute).UTC().Truncate(time.Millisecond)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-start
			_, err := store.CreateTrip(context.Background(), app.CreateTripInput{
				UserID: testUserID, Mode: "walk", DistanceKm: 100,
				TraveledAt: now.Add(time.Duration(i) * time.Millisecond),
			})
			errs <- err
		}(i)
	}
	close(start)
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatalf("并发上报失败: %v", err)
		}
	}
	total := mustQueryInt(t, store.db, `SELECT COALESCE(SUM(amount), 0) FROM point_txn
		WHERE account_id = ? AND txn_type = 'TRAVEL' AND occurred_at >= ? AND occurred_at < ?`,
		testAccountID, startOfDay(time.Now()), startOfDay(time.Now()).AddDate(0, 0, 1))
	if total != 500 {
		t.Fatalf("当日发放积分 = %d，期望 500", total)
	}
	assertBatchLedgerBalanced(t, store.db)
}

func TestTripSnapshotAndHash(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	if hashTrip(1, "bus", 10.000001, now) == hashTrip(1, "bus", 10.000002, now) {
		t.Fatal("六位小数不同的距离不应产生相同行程哈希")
	}
}

func TestCreateTripIdempotentKeepsLedgerBalanced(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	// 夹具的历史发放不应占用本次上报所在自然日的积分上限。
	mustExec(t, store.db,
		"UPDATE point_txn SET occurred_at = DATE_SUB(NOW(3), INTERVAL 1 DAY) WHERE account_id = ?", testAccountID)
	mustExec(t, store.db, "UPDATE point_txn SET occurred_at = NOW(3) WHERE id = 9002")
	input := app.CreateTripInput{
		UserID: testUserID, Mode: "walk", DistanceKm: 100,
		TraveledAt: time.Now().Add(-time.Minute).UTC().Truncate(time.Millisecond),
	}

	created, err := store.CreateTrip(ctx, input)
	if err != nil {
		t.Fatalf("首次上报失败: %v", err)
	}
	if created.GrantedPoints <= 0 {
		t.Fatalf("发放积分 = %d，期望正数", created.GrantedPoints)
	}
	if created.RawPoints <= created.GrantedPoints {
		t.Fatalf("应触发每日上限: %+v", created)
	}
	mustExec(t, store.db, "UPDATE carbon_record SET rule_snapshot = 'changed-format' WHERE trip_id = (SELECT id FROM trip_record WHERE user_id = ? AND trip_hash = ?)",
		input.UserID, hashTrip(input.UserID, input.Mode, input.DistanceKm, input.TraveledAt))
	replayed, err := store.CreateTrip(ctx, input)
	if err != nil {
		t.Fatalf("幂等重放失败: %v", err)
	}
	if replayed.ID != created.ID || replayed.GrantedPoints != created.GrantedPoints || replayed.RawPoints != created.RawPoints {
		t.Fatalf("幂等重放结果不一致: created=%+v replayed=%+v", created, replayed)
	}
	list, err := store.ListTrips(ctx, app.TripQuery{UserID: input.UserID, Page: 1, PageSize: 20})
	if err != nil || len(list.Items) != 1 || list.Items[0].RawPoints != created.RawPoints || !list.Items[0].CappedByDaily {
		t.Fatalf("快照变更后的列表结果: %+v, err=%v", list, err)
	}

	tripID := mustQueryInt(t, store.db,
		"SELECT id FROM trip_record WHERE user_id = ? AND trip_hash = ?", input.UserID,
		hashTrip(input.UserID, input.Mode, input.DistanceKm, input.TraveledAt))
	if got := mustQueryInt(t, store.db,
		"SELECT COUNT(*) FROM trip_record WHERE id = ?", tripID); got != 1 {
		t.Fatalf("行程数 = %d，期望 1", got)
	}
	if got := mustQueryInt(t, store.db,
		"SELECT COUNT(*) FROM point_txn WHERE account_id = ? AND txn_type = 'TRAVEL' AND biz_ref_id = ?",
		testAccountID, tripID); got != 1 {
		t.Fatalf("行程积分流水数 = %d，期望 1", got)
	}
	if got := mustQueryInt(t, store.db, `
		SELECT COALESCE(SUM(pb.remaining), 0)
		FROM point_batch pb JOIN point_txn pt ON pt.id = pb.grant_txn_id
		WHERE pt.account_id = ? AND pt.txn_type = 'TRAVEL' AND pt.biz_ref_id = ?`,
		testAccountID, tripID); got != created.GrantedPoints {
		t.Fatalf("行程积分批次剩余 = %d，期望 %d", got, created.GrantedPoints)
	}
	if got := mustQueryInt(t, store.db, "SELECT balance FROM point_account WHERE id = ?", testAccountID); got != 1000+created.GrantedPoints {
		t.Fatalf("账户余额 = %d，期望 %d", got, 1000+created.GrantedPoints)
	}
	if got := mustQueryInt(t, store.db,
		"SELECT COUNT(*) FROM event_outbox WHERE topic = ? AND aggregate_type = 'trip'", eventbus.TopicTripVerified); got != 1 {
		t.Fatalf("行程 Outbox 事件数 = %d，期望 1", got)
	}
	assertBatchLedgerBalanced(t, store.db)
	if result, err := store.ReconcilePoints(ctx); err != nil || !result.OK {
		t.Fatalf("行程发放后积分对账不一致: %+v, err=%v", result, err)
	}
}
