package domain

import (
	"errors"
	"math"
	"testing"
)

// TestCarbonSavedGrams 用表驱动覆盖五类出行方式与非法输入（等价类划分）。
func TestCarbonSavedGrams(t *testing.T) {
	tests := []struct {
		name     string
		distance float64
		mode     TravelMode
		want     float64
		wantErr  error
	}{
		// ---- 有效等价类：五种出行方式 ----
		{"步行 5km", 5, ModeWalk, 960, nil},
		{"骑行 2.5km", 2.5, ModeBike, 480, nil},
		{"地铁 10km", 10, ModeSubway, 1510, nil},    // 10 × (192 − 41)
		{"公交 10km", 10, ModeBus, 1240, nil},       // 10 × (192 − 68)
		{"共享单车 1km", 1, ModeSharedBike, 192, nil}, // 1 × (192 − 0)
		{"最短距离边界 0.1km", 0.1, ModeWalk, 19.2, nil},
		{"最长距离边界 100km", 100, ModeWalk, 19200, nil},

		// ---- 无效等价类 ----
		{"距离为 0", 0, ModeWalk, 0, ErrInvalidDistance},
		{"距离为负", -3, ModeSubway, 0, ErrInvalidDistance},
		{"距离为 NaN", math.NaN(), ModeBus, 0, ErrInvalidDistance},
		{"距离为 +Inf", math.Inf(1), ModeBus, 0, ErrInvalidDistance},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			em, err := LookupEmission(tc.mode)
			if err != nil {
				t.Fatalf("LookupEmission(%q) 意外失败: %v", tc.mode, err)
			}
			got, err := CarbonSavedGrams(tc.distance, em)
			if tc.wantErr != nil {
				if !errors.Is(err, tc.wantErr) {
					t.Fatalf("期望错误 %v，实际 %v", tc.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("意外错误: %v", err)
			}
			if math.Abs(got-tc.want) > 1e-9 {
				t.Fatalf("碳减排量 = %v, 期望 %v", got, tc.want)
			}
		})
	}
}

// TestCarbonSavedGramsNeverNegative 校验 EF_actual > EF_base 时下界为 0，不产生负积分。
func TestCarbonSavedGramsNeverNegative(t *testing.T) {
	got, err := CarbonSavedGrams(10, Emission{Base: 50, Actual: 80})
	if err != nil {
		t.Fatalf("意外错误: %v", err)
	}
	if got != 0 {
		t.Fatalf("期望 0，实际 %v", got)
	}
}

func TestLookupEmissionUnknown(t *testing.T) {
	if _, err := LookupEmission(TravelMode("rocket")); !errors.Is(err, ErrUnknownTravelMode) {
		t.Fatalf("期望 ErrUnknownTravelMode，实际 %v", err)
	}
}
