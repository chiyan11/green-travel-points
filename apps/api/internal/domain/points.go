package domain

import (
	"errors"
	"fmt"
	"math"
)

// PointsRule 是"碳减排量 → 积分"的换算与约束规则。
// 运行期取值来自 sys_config 表（carbon_per_point / daily_points_cap /
// trip_min_distance_km / trip_max_distance_km），禁止在代码里写死。
type PointsRule struct {
	// GramsPerPoint 表示多少克碳减排量兑换 1 积分，必须为正。
	GramsPerPoint int64
	// DailyCap 单日可获得积分上限，<=0 表示不限制。
	DailyCap int64
	// MinDistanceKm / MaxDistanceKm 单次行程距离的合法区间（含端点）。
	MinDistanceKm float64
	MaxDistanceKm float64
}

var (
	// ErrInvalidRule 换算规则自身非法（配置错误）。
	ErrInvalidRule = errors.New("domain: 积分规则非法")
	// ErrDistanceOutOfRange 行程距离超出业务允许区间。
	ErrDistanceOutOfRange = errors.New("domain: 行程距离超出允许范围")
)

// DefaultPointsRule 返回与数据库种子数据一致的默认规则，供测试与初始化使用。
func DefaultPointsRule() PointsRule {
	return PointsRule{
		GramsPerPoint: 100,
		DailyCap:      500,
		MinDistanceKm: 0.1,
		MaxDistanceKm: 100,
	}
}

// CheckRule 校验规则本身是否可用。
func (r PointsRule) CheckRule() error {
	if r.GramsPerPoint <= 0 {
		return fmt.Errorf("%w: grams_per_point=%d", ErrInvalidRule, r.GramsPerPoint)
	}
	if r.MaxDistanceKm <= r.MinDistanceKm {
		return fmt.Errorf("%w: min=%v max=%v", ErrInvalidRule, r.MinDistanceKm, r.MaxDistanceKm)
	}
	return nil
}

// ValidateDistance 校验单次行程距离是否落在允许区间内（含端点）。
func (r PointsRule) ValidateDistance(distanceKm float64) error {
	if math.IsNaN(distanceKm) || math.IsInf(distanceKm, 0) {
		return fmt.Errorf("%w: distance=%v", ErrInvalidDistance, distanceKm)
	}
	if distanceKm < r.MinDistanceKm || distanceKm > r.MaxDistanceKm {
		return fmt.Errorf("%w: distance=%v, allowed=[%v, %v]",
			ErrDistanceOutOfRange, distanceKm, r.MinDistanceKm, r.MaxDistanceKm)
	}
	return nil
}

// PointsFor 把碳减排克数换算为应发积分：
//
//	points = floor(carbon_saved / grams_per_point)
//
// 向下取整保证不透支；加入 1e-9 容差吸收浮点误差（避免 300g 算出 2 分）。
func (r PointsRule) PointsFor(carbonSavedGrams float64) (int64, error) {
	if r.GramsPerPoint <= 0 {
		return 0, fmt.Errorf("%w: grams_per_point=%d", ErrInvalidRule, r.GramsPerPoint)
	}
	if math.IsNaN(carbonSavedGrams) || math.IsInf(carbonSavedGrams, 0) {
		return 0, fmt.Errorf("domain: 碳减排量非法: %v", carbonSavedGrams)
	}
	if carbonSavedGrams <= 0 {
		return 0, nil
	}
	points := int64(math.Floor(carbonSavedGrams/float64(r.GramsPerPoint) + 1e-9))
	if points < 0 {
		return 0, nil
	}
	return points, nil
}

// CapDaily 套用单日上限，返回本次实际可发放的积分。
//
//	raw           本次按公式算出的积分
//	alreadyToday  当日已发放积分
//
// DailyCap <= 0 时视为不限制。
func (r PointsRule) CapDaily(raw, alreadyToday int64) int64 {
	if raw <= 0 {
		return 0
	}
	if r.DailyCap <= 0 {
		return raw
	}
	if alreadyToday < 0 {
		alreadyToday = 0
	}
	remain := r.DailyCap - alreadyToday
	if remain <= 0 {
		return 0
	}
	if raw > remain {
		return remain
	}
	return raw
}

// Grant 是一次行程可发放积分的完整计算结果，便于落库与测试断言。
type Grant struct {
	DistanceKm       float64
	CarbonSavedGrams float64
	RawPoints        int64
	GrantedPoints    int64
	CappedByDaily    bool
}

// EvaluateTrip 用给定排放因子与规则，算出一次行程的完整发放结果。
// 它不触碰数据库，是"上报行程"用例里可被单测覆盖的核心。
func EvaluateTrip(distanceKm float64, e Emission, rule PointsRule, alreadyToday int64) (Grant, error) {
	if err := rule.CheckRule(); err != nil {
		return Grant{}, err
	}
	if err := rule.ValidateDistance(distanceKm); err != nil {
		return Grant{}, err
	}
	saved, err := CarbonSavedGrams(distanceKm, e)
	if err != nil {
		return Grant{}, err
	}
	raw, err := rule.PointsFor(saved)
	if err != nil {
		return Grant{}, err
	}
	granted := rule.CapDaily(raw, alreadyToday)
	return Grant{
		DistanceKm:       distanceKm,
		CarbonSavedGrams: saved,
		RawPoints:        raw,
		GrantedPoints:    granted,
		CappedByDaily:    granted < raw,
	}, nil
}
