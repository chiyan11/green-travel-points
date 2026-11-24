package app

import (
	"errors"
	"strings"
	"testing"
)

// UC-14 参数配置：写入前的值校验必须挡住"把主链路改坏"的取值。
func TestValidateConfigValue(t *testing.T) {
	longValue := strings.Repeat("9", 256)
	cases := []struct {
		name      string
		key       string
		valueType string
		value     string
		wantErr   bool
	}{
		{"日上限正常", "daily_points_cap", "int", "10", false},
		{"日上限 0 表示不限制", "daily_points_cap", "int", "0", false},
		{"日上限为负", "daily_points_cap", "int", "-1", true},
		{"日上限非整数", "daily_points_cap", "int", "10.5", true},
		{"换算基数 0 会除零", "carbon_per_point", "int", "0", true},
		{"换算基数正常", "carbon_per_point", "int", "100", false},
		{"过期天数为 0", "points_expire_days", "int", "0", true},
		{"最长距离非法字符", "trip_max_distance_km", "decimal", "abc", true},
		{"最短距离为 0", "trip_min_distance_km", "decimal", "0", true},
		{"最短距离正常", "trip_min_distance_km", "decimal", "0.1", false},
		{"最长距离 NaN", "trip_max_distance_km", "decimal", "NaN", true},
		{"最长距离 Inf", "trip_max_distance_km", "decimal", "Inf", true},
		{"最长距离正常", "trip_max_distance_km", "decimal", "120.5", false},
		{"未登记键只校验类型", "brand_new_param", "int", "7", false},
		{"布尔 true", "flag", "bool", "true", false},
		{"布尔 1", "flag", "bool", "1", false},
		{"布尔非法", "flag", "bool", "yes", true},
		{"字符串任意值", "note", "string", "运营自定义", false},
		{"值类型未知", "note", "json", "1", true},
		{"空 key", "  ", "int", "1", true},
		{"空 value", "daily_points_cap", "int", "  ", true},
		{"超长 value", "note", "string", longValue, true},
		{"首尾空格按 trim 后处理", "daily_points_cap", "int", "  20  ", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := ValidateConfigValue(c.key, c.valueType, c.value)
			if c.wantErr {
				if !errors.Is(err, ErrInvalidInput) {
					t.Fatalf("期望 ErrInvalidInput，实际 %v", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("期望通过，实际 %v", err)
			}
		})
	}
}

func TestValidateTravelModeUpdate(t *testing.T) {
	name := "公交"
	blank := "  "
	zero := 0.0
	neg := -1.0
	base := 192.0
	actual := 68.0
	enabled := false

	cases := []struct {
		name    string
		in      TravelModeUpdate
		wantErr bool
	}{
		{"只改名称", TravelModeUpdate{Name: &name}, false},
		{"只改排放因子", TravelModeUpdate{EFActual: &actual}, false},
		{"只改启用状态", TravelModeUpdate{Enabled: &enabled}, false},
		{"排放因子 0 合法（零排放方式）", TravelModeUpdate{EFActual: &zero}, false},
		{"同时改基准与实测", TravelModeUpdate{EFBase: &base, EFActual: &actual}, false},
		{"空更新", TravelModeUpdate{}, true},
		{"空名称", TravelModeUpdate{Name: &blank}, true},
		{"基准因子必须为正", TravelModeUpdate{EFBase: &zero}, true},
		{"实测因子不能为负", TravelModeUpdate{EFActual: &neg}, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := ValidateTravelModeUpdate(c.in)
			if c.wantErr {
				if !errors.Is(err, ErrInvalidInput) {
					t.Fatalf("期望 ErrInvalidInput，实际 %v", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("期望通过，实际 %v", err)
			}
		})
	}
}
