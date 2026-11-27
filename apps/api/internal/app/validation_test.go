package app

import (
	"strings"
	"testing"
)

func validProductInput() AdminProductInput {
	return AdminProductInput{
		MerchantID: 1,
		Name:       "环保帆布袋",
		Category:   "文创",
		SKUs:       []AdminSKUInput{{Name: "默认", PointsPrice: 100, Stock: 10}},
	}
}

func TestValidateProductInput(t *testing.T) {
	cases := []struct {
		name    string
		mutate  func(*AdminProductInput)
		wantErr bool
	}{
		{"valid", func(*AdminProductInput) {}, false},
		{"missing merchant", func(in *AdminProductInput) { in.MerchantID = 0 }, true},
		{"empty name", func(in *AdminProductInput) { in.Name = "  " }, true},
		{"name 128 runes", func(in *AdminProductInput) { in.Name = strings.Repeat("名", MaxProductNameRunes) }, false},
		{"name 129 runes", func(in *AdminProductInput) { in.Name = strings.Repeat("名", MaxProductNameRunes+1) }, true},
		{"description 500", func(in *AdminProductInput) { in.Description = strings.Repeat("d", MaxProductDescriptionRunes) }, false},
		{"description 501", func(in *AdminProductInput) { in.Description = strings.Repeat("d", MaxProductDescriptionRunes+1) }, true},
		{"category 33", func(in *AdminProductInput) { in.Category = strings.Repeat("类", MaxProductCategoryRunes+1) }, true},
		{"image url 256", func(in *AdminProductInput) { in.ImageURL = strings.Repeat("u", MaxProductImageURLRunes+1) }, true},
		{"negative limit", func(in *AdminProductInput) { in.LimitPerUser = -1 }, true},
		{"unknown status", func(in *AdminProductInput) { in.Status = "PUBLISHED" }, true},
		{"no sku", func(in *AdminProductInput) { in.SKUs = nil }, true},
		{"empty sku name", func(in *AdminProductInput) { in.SKUs[0].Name = " " }, true},
		{"sku name 65", func(in *AdminProductInput) { in.SKUs[0].Name = strings.Repeat("S", MaxSKUNameRunes+1) }, true},
		{"zero price", func(in *AdminProductInput) { in.SKUs[0].PointsPrice = 0 }, true},
		{"negative stock", func(in *AdminProductInput) { in.SKUs[0].Stock = -1 }, true},
		{"duplicate sku case-insensitive", func(in *AdminProductInput) {
			in.SKUs = append(in.SKUs, AdminSKUInput{Name: "DEFAULT", PointsPrice: 200, Stock: 1})
			in.SKUs[0].Name = "default"
		}, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			in := validProductInput()
			tc.mutate(&in)
			err := ValidateProductInput(in)
			if tc.wantErr && err == nil {
				t.Fatalf("期望校验失败，实际通过")
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("期望通过，实际 %v", err)
			}
		})
	}
}

func TestValidateProductUpdate(t *testing.T) {
	str := func(s string) *string { return &s }
	num := func(n int) *int { return &n }
	cases := []struct {
		name    string
		in      ProductUpdate
		wantErr bool
	}{
		{"empty patch", ProductUpdate{}, false},
		{"empty name", ProductUpdate{Name: str(" ")}, true},
		{"name too long", ProductUpdate{Name: str(strings.Repeat("名", MaxProductNameRunes+1))}, true},
		{"description too long", ProductUpdate{Description: str(strings.Repeat("d", MaxProductDescriptionRunes+1))}, true},
		{"empty category", ProductUpdate{Category: str("")}, true},
		{"bad status", ProductUpdate{Status: str("PAID")}, true},
		{"negative limit", ProductUpdate{LimitPerUser: num(-1)}, true},
		{"valid patch", ProductUpdate{Name: str("新名字"), Status: str("online")}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateProductUpdate(tc.in)
			if tc.wantErr != (err != nil) {
				t.Fatalf("wantErr=%v 实际 err=%v", tc.wantErr, err)
			}
		})
	}
}

func TestValidateRegisterInput(t *testing.T) {
	cases := []struct {
		name     string
		phone    string
		nickname string
		password string
		wantErr  bool
	}{
		{"valid", "13800000001", "小李", "demo123456", false},
		{"empty nickname", "13800000001", "", "demo123456", false},
		{"phone 10 digits", "1380000000", "小李", "demo123456", true},
		{"phone starts with 12", "12800000001", "小李", "demo123456", true},
		{"phone letters", "1380000000a", "小李", "demo123456", true},
		{"phone 21 runes", "13800000001" + strings.Repeat("0", 10), "小李", "demo123456", true},
		{"nickname 65", "13800000001", strings.Repeat("昵", MaxNicknameRunes+1), "demo123456", true},
		{"password 5", "13800000001", "小李", "12345", true},
		{"password 6", "13800000001", "小李", "123456", false},
		{"password 129", "13800000001", "小李", strings.Repeat("p", MaxPasswordRunes+1), true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateRegisterInput(tc.phone, tc.nickname, tc.password)
			if tc.wantErr != (err != nil) {
				t.Fatalf("wantErr=%v 实际 err=%v", tc.wantErr, err)
			}
		})
	}
}
