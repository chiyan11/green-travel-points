// Package domain 承载与框架无关的业务规则。
//
// 这里是"碳减排量 → 积分"的唯一事实来源：HTTP 层、增量 Worker 与测试共同复用，
// 不依赖数据库、不依赖 HTTP，因此可以用纯函数单测覆盖（报告第四章"组件测试·基本路径法"）。
//
// 重要约定：排放因子与换算参数在运行期必须来自数据库（travel_mode / sys_config 表），
// SeedEmissionFactors 仅用于数据库种子数据与单测，不得在业务路径上直接取用。
package domain

import (
	"errors"
	"fmt"
	"math"
)

// TravelMode 是出行方式代码，与 travel_mode.code 一一对应。
type TravelMode string

const (
	ModeWalk       TravelMode = "walk"
	ModeBike       TravelMode = "bike"
	ModeSubway     TravelMode = "subway"
	ModeBus        TravelMode = "bus"
	ModeSharedBike TravelMode = "shared_bike"
)

// Emission 表示某种出行方式的排放因子，单位 gCO2e/km。
//
//	Base   —— 基准排放（对标燃油私家车）
//	Actual —— 该方式的实际排放
type Emission struct {
	Base   float64
	Actual float64
}

var (
	// ErrUnknownTravelMode 出行方式未在因子表中登记。
	ErrUnknownTravelMode = errors.New("domain: 未知的出行方式")
	// ErrInvalidDistance 行程距离非法（<=0、NaN 或无穷）。
	ErrInvalidDistance = errors.New("domain: 行程距离非法")
)

// EmissionFactorBase 是燃油私家车的基准排放因子（gCO2e/km）。
// 该值同时也是报告中"业务规则"一节的基线口径。
const EmissionFactorBase = 192.0

// SeedEmissionFactors 是 travel_mode 表的种子数据，也是单测的期望值来源。
// 运行期请从数据库读取，以便运营调整因子后无需发版。
var SeedEmissionFactors = map[TravelMode]Emission{
	ModeWalk:       {Base: EmissionFactorBase, Actual: 0},
	ModeBike:       {Base: EmissionFactorBase, Actual: 0},
	ModeSubway:     {Base: EmissionFactorBase, Actual: 41},
	ModeBus:        {Base: EmissionFactorBase, Actual: 68},
	ModeSharedBike: {Base: EmissionFactorBase, Actual: 0},
}

// LookupEmission 从种子表取排放因子，仅供种子数据初始化与测试使用。
func LookupEmission(mode TravelMode) (Emission, error) {
	e, ok := SeedEmissionFactors[mode]
	if !ok {
		return Emission{}, fmt.Errorf("%w: %q", ErrUnknownTravelMode, string(mode))
	}
	return e, nil
}

// CarbonSavedGrams 计算一次行程相对"燃油私家车基准"少排放的二氧化碳克数。
//
//	carbon_saved = distance × (EF_base − EF_actual)，下界为 0
//
// 返回值为 gCO2e，可能带小数（数据库用 DECIMAL(12,3) 存储）。
func CarbonSavedGrams(distanceKm float64, e Emission) (float64, error) {
	if math.IsNaN(distanceKm) || math.IsInf(distanceKm, 0) || distanceKm <= 0 {
		return 0, fmt.Errorf("%w: distance=%v", ErrInvalidDistance, distanceKm)
	}
	saved := distanceKm * (e.Base - e.Actual)
	if saved < 0 {
		return 0, nil
	}
	return saved, nil
}
