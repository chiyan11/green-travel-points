package domain

import (
	"errors"
	"math"
	"testing"
)

func TestPointsFor(t *testing.T) {
	rule := DefaultPointsRule() // 100 gCO2e = 1 分

	tests := []struct {
		name   string
		carbon float64
		want   int64
	}{
		{"零排放得零分", 0, 0},
		{"不足一分的容差下界", 99.999, 0},
		{"恰好一百分", 100, 1},
		{"浮点误差不丢分（300g）", 300, 3},
		{"372g 向下取整为 3 分", 372, 3},
		{"453g 向下取整为 4 分", 453, 4},
		{"负值不产生负分", -500, 0},
		{"NaN 取零分", math.NaN(), 0},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := rule.PointsFor(tc.carbon)
			if tc.name == "NaN 取零分" {
				// PointsFor 对 NaN 返回错误而非静默 0，这里显式断言错误分支
				if err == nil {
					t.Fatalf("NaN 期望返回错误，实际 0 分")
				}
				return
			}
			if err != nil {
				t.Fatalf("意外错误: %v", err)
			}
			if got != tc.want {
				t.Fatalf("积分 = %d, 期望 %d", got, tc.want)
			}
		})
	}
}

func TestPointsForInvalidRule(t *testing.T) {
	rule := PointsRule{GramsPerPoint: 0}
	if _, err := rule.PointsFor(1000); !errors.Is(err, ErrInvalidRule) {
		t.Fatalf("期望 ErrInvalidRule，实际 %v", err)
	}
}

func TestValidateDistance(t *testing.T) {
	rule := DefaultPointsRule() // [0.1, 100]
	tests := []struct {
		name     string
		distance float64
		wantErr  error
	}{
		{"下边界 0.1 合法", 0.1, nil},
		{"上边界 100 合法", 100, nil},
		{"中间值 5 合法", 5, nil},
		{"低于下边界", 0.09, ErrDistanceOutOfRange},
		{"高于上边界", 100.01, ErrDistanceOutOfRange},
		{"零距离", 0, ErrDistanceOutOfRange},
		{"负距离", -1, ErrDistanceOutOfRange},
		{"NaN", math.NaN(), ErrInvalidDistance},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := rule.ValidateDistance(tc.distance)
			if tc.wantErr == nil {
				if err != nil {
					t.Fatalf("期望通过，实际错误 %v", err)
				}
				return
			}
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("期望 %v，实际 %v", tc.wantErr, err)
			}
		})
	}
}

func TestCapDaily(t *testing.T) {
	rule := DefaultPointsRule() // 上限 500
	tests := []struct {
		name         string
		raw          int64
		alreadyToday int64
		want         int64
	}{
		{"未达上限全额发放", 40, 100, 40},
		{"接近上限被削平", 40, 480, 20},
		{"恰好用满上限", 10, 495, 5},
		{"已达上限不再发放", 10, 500, 0},
		{"超上限不再发放", 10, 520, 0},
		{"零积分不发", 0, 0, 0},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := rule.CapDaily(tc.raw, tc.alreadyToday); got != tc.want {
				t.Fatalf("实际发放 %d, 期望 %d", got, tc.want)
			}
		})
	}

	noCap := rule
	noCap.DailyCap = 0
	if got := noCap.CapDaily(9999, 9999); got != 9999 {
		t.Fatalf("不限制时应收全额，实际 %d", got)
	}
}

func TestCheckRule(t *testing.T) {
	if err := DefaultPointsRule().CheckRule(); err != nil {
		t.Fatalf("默认规则应合法: %v", err)
	}
	bad := PointsRule{GramsPerPoint: 100, MinDistanceKm: 10, MaxDistanceKm: 5}
	if err := bad.CheckRule(); !errors.Is(err, ErrInvalidRule) {
		t.Fatalf("期望 ErrInvalidRule，实际 %v", err)
	}
}

// TestEvaluateTrip 覆盖"上报行程 → 算碳减排 → 算积分"的完整组件路径。
func TestEvaluateTrip(t *testing.T) {
	rule := DefaultPointsRule()
	subway, _ := LookupEmission(ModeSubway)

	g, err := EvaluateTrip(3, subway, rule, 0)
	if err != nil {
		t.Fatalf("意外错误: %v", err)
	}
	if math.Abs(g.CarbonSavedGrams-453) > 1e-9 {
		t.Fatalf("碳减排 = %v, 期望 453", g.CarbonSavedGrams)
	}
	if g.RawPoints != 4 || g.GrantedPoints != 4 || g.CappedByDaily {
		t.Fatalf("积分计算错误: %+v", g)
	}

	// 当日已得 498 分，本次应只补 2 分
	g2, err := EvaluateTrip(3, subway, rule, 498)
	if err != nil {
		t.Fatalf("意外错误: %v", err)
	}
	if g2.GrantedPoints != 2 || !g2.CappedByDaily {
		t.Fatalf("单日上限未生效: %+v", g2)
	}

	// 超范围距离必须报错，且不产生任何发放
	if _, err := EvaluateTrip(0.05, subway, rule, 0); !errors.Is(err, ErrDistanceOutOfRange) {
		t.Fatalf("期望 ErrDistanceOutOfRange，实际 %v", err)
	}
}
