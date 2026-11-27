package app

import (
	"regexp"
	"strings"
	"unicode/utf8"
)

// 入参长度上限与 MySQL 列宽保持一致（VARCHAR 按字符计）：
// db/migrations/000004_user_trip_carbon.up.sql、000005_mall_orders.up.sql。
// 放在这里是让 handler 能在写库前拒绝超长输入，避免 1406 变成 500。
const (
	MaxProductNameRunes        = 128 // product.name VARCHAR(128)
	MaxProductDescriptionRunes = 500 // product.description VARCHAR(500)
	MaxProductCategoryRunes    = 32  // product.category VARCHAR(32)
	MaxProductImageURLRunes    = 255 // product.cover_url VARCHAR(255)
	MaxSKUNameRunes            = 64  // product_sku.sku_name VARCHAR(64)
	MaxPhoneRunes              = 20  // user.phone VARCHAR(20)
	MaxNicknameRunes           = 64  // user.nickname VARCHAR(64)
	MaxPasswordRunes           = 128 // 上限只防超长输入；存储的是定长 PBKDF2 摘要
)

// 与前端 lib/validators.ts 保持同一口径：1 开头、第二位 3-9、共 11 位。
var phonePattern = regexp.MustCompile("^1[3-9][0-9]{9}$")

// productStatuses 与 product.status 枚举一致（repo 侧另有同样的白名单，
// 这里提前拒绝，保证 handler 返回 INVALID_ARGUMENT 而不是等到 SQL 才失败）。
var productStatuses = map[string]struct{}{
	"DRAFT": {}, "ONLINE": {}, "OFFLINE": {},
}

// ValidateProductInput 校验后台新建商品入参：必填项、字段长度、SKU 数量与唯一性。
func ValidateProductInput(in AdminProductInput) error {
	if in.MerchantID <= 0 {
		return ErrInvalidInput
	}
	if utf8.RuneCountInString(strings.TrimSpace(in.Name)) == 0 ||
		utf8.RuneCountInString(in.Name) > MaxProductNameRunes {
		return ErrInvalidInput
	}
	if utf8.RuneCountInString(strings.TrimSpace(in.Category)) == 0 ||
		utf8.RuneCountInString(in.Category) > MaxProductCategoryRunes {
		return ErrInvalidInput
	}
	if utf8.RuneCountInString(in.Description) > MaxProductDescriptionRunes {
		return ErrInvalidInput
	}
	if utf8.RuneCountInString(in.ImageURL) > MaxProductImageURLRunes {
		return ErrInvalidInput
	}
	if in.LimitPerUser < 0 {
		return ErrInvalidInput
	}
	if status := strings.ToUpper(strings.TrimSpace(in.Status)); status != "" {
		if _, ok := productStatuses[status]; !ok {
			return ErrInvalidInput
		}
	}
	if len(in.SKUs) == 0 {
		return ErrInvalidInput
	}
	seen := make(map[string]struct{}, len(in.SKUs))
	for _, sku := range in.SKUs {
		name := strings.TrimSpace(sku.Name)
		if name == "" || utf8.RuneCountInString(sku.Name) > MaxSKUNameRunes ||
			sku.PointsPrice <= 0 || sku.Stock < 0 {
			return ErrInvalidInput
		}
		// 唯一键 uk_sku_product_name 使用 *_ci 排序规则，大小写不同也算重名。
		key := strings.ToLower(name)
		if _, ok := seen[key]; ok {
			return ErrInvalidInput
		}
		seen[key] = struct{}{}
	}
	return nil
}

// ValidateProductUpdate 校验后台更新商品入参（nil 表示不改该字段）。
// 全部字段为空是合法的：仓储层会按“只读查询”返回当前商品。
func ValidateProductUpdate(in ProductUpdate) error {
	if in.Name != nil {
		name := strings.TrimSpace(*in.Name)
		if name == "" || utf8.RuneCountInString(*in.Name) > MaxProductNameRunes {
			return ErrInvalidInput
		}
	}
	if in.Description != nil && utf8.RuneCountInString(*in.Description) > MaxProductDescriptionRunes {
		return ErrInvalidInput
	}
	if in.Category != nil {
		c := strings.TrimSpace(*in.Category)
		if c == "" || utf8.RuneCountInString(*in.Category) > MaxProductCategoryRunes {
			return ErrInvalidInput
		}
	}
	if in.ImageURL != nil && utf8.RuneCountInString(*in.ImageURL) > MaxProductImageURLRunes {
		return ErrInvalidInput
	}
	if in.Status != nil {
		st := strings.ToUpper(strings.TrimSpace(*in.Status))
		if _, ok := productStatuses[st]; !ok {
			return ErrInvalidInput
		}
	}
	if in.LimitPerUser != nil && *in.LimitPerUser < 0 {
		return ErrInvalidInput
	}
	return nil
}

// ValidateRegisterInput 校验注册入参：手机号格式、昵称与密码长度。
// phone 已由 handler 从 phone / account 二者择一，nickname 为空时由调用方回落为手机号。
func ValidateRegisterInput(phone, nickname, password string) error {
	phone = strings.TrimSpace(phone)
	if utf8.RuneCountInString(phone) > MaxPhoneRunes || !phonePattern.MatchString(phone) {
		return ErrInvalidInput
	}
	if utf8.RuneCountInString(nickname) > MaxNicknameRunes {
		return ErrInvalidInput
	}
	n := utf8.RuneCountInString(password)
	if n < 6 || n > MaxPasswordRunes {
		return ErrInvalidInput
	}
	return nil
}
