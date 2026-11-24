package app

import (
	"context"
	"math"
	"strconv"
	"strings"
	"time"
)

// 以下是运营后台（M8）使用的查询条件与入参。
// 与前台接口的区别：后台可以看到非上架商品、全部订单，并执行状态流转。

// AdminProductQuery 是后台商品列表查询条件。
type AdminProductQuery struct {
	Page     int
	PageSize int
	Status   string // DRAFT / ONLINE / OFFLINE，空表示全部
	Keyword  string
}

// AdminSKUInput 是新建商品时随商品一起提交的 SKU。
type AdminSKUInput struct {
	Name        string `json:"name"`
	PointsPrice int64  `json:"pointsPrice"`
	Stock       int    `json:"stock"`
}

// AdminProductInput 是后台新建商品的入参。
type AdminProductInput struct {
	MerchantID   int64           `json:"merchantId"`
	Name         string          `json:"name"`
	Description  string          `json:"description"`
	Category     string          `json:"category"`
	ImageURL     string          `json:"imageUrl"`
	Status       string          `json:"status"` // 默认 DRAFT
	SortOrder    int             `json:"sortOrder"`
	LimitPerUser int             `json:"limitPerUser"`
	SKUs         []AdminSKUInput `json:"skus"`
}

// ProductUpdate 是后台更新商品的入参（nil 表示不修改该字段）。
type ProductUpdate struct {
	Name         *string `json:"name"`
	Description  *string `json:"description"`
	Category     *string `json:"category"`
	ImageURL     *string `json:"imageUrl"`
	Status       *string `json:"status"`
	SortOrder    *int    `json:"sortOrder"`
	LimitPerUser *int    `json:"limitPerUser"`
}

// AdminOrderQuery 是后台订单列表查询条件。
type AdminOrderQuery struct {
	Page     int
	PageSize int
	Status   string
}

// ReconcileItem 是一条对账不一致的账户记录（BR-07 证据）。
type ReconcileItem struct {
	UserID    int64 `json:"userId"`
	AccountID int64 `json:"accountId"`
	Balance   int64 `json:"balance"`
	LedgerSum int64 `json:"ledgerSum"`
}

// ReconcileResult 是积分对账结果。
type ReconcileResult struct {
	Accounts   int64           `json:"accounts"`
	Mismatches []ReconcileItem `json:"mismatches"`
	OK         bool            `json:"ok"`
}

// ExpireResult 是一次积分过期清算的结果（POST /api/v1/admin/points/expire）。
// Batches 为被置为 EXPIRED 的批次数，Points 为实际过期扣减的积分，Accounts 为余额发生变化的账户数。
type ExpireResult struct {
	Batches  int64 `json:"batches"`
	Points   int64 `json:"points"`
	Accounts int64 `json:"accounts"`
}

// AdjustPointsInput 是人工调账入参（UC-15，BR-14 要求必须填理由）。
type AdjustPointsInput struct {
	UserID int64  `json:"userId"`
	Amount int64  `json:"amount"` // 正数=增加，负数=扣减，0 非法
	Reason string `json:"reason"`
}

// AdjustResult 是调账结果，带上审计记录 ID 便于对照审计日志。
type AdjustResult struct {
	AccountID int64 `json:"accountId"`
	UserID    int64 `json:"userId"`
	Balance   int64 `json:"balance"`
	AuditID   int64 `json:"auditId"`
}

// AuditLog 是一条运营审计记录（UC-19）。
type AuditLog struct {
	ID          int64     `json:"id"`
	OperatorID  int64     `json:"operatorId"`
	Action      string    `json:"action"` // ADJUST / CONFIG_UPDATE / TRAVEL_MODE_UPDATE
	TargetType  string    `json:"targetType"`
	TargetID    int64     `json:"targetId"`
	TargetKey   string    `json:"targetKey"`
	Amount      int64     `json:"amount"`
	BeforeValue string    `json:"beforeValue"`
	AfterValue  string    `json:"afterValue"`
	Reason      string    `json:"reason"`
	CreatedAt   time.Time `json:"createdAt"`
}

// AuditQuery 是审计日志检索条件：按操作人 / 动作 / 对象过滤。
type AuditQuery struct {
	OperatorID int64
	Action     string
	TargetType string
	TargetKey  string
	Page       int
	PageSize   int
}

// ConfigItem 是一条系统参数（UC-14 参数配置）。值一律用字符串承载，
// 语义由 ValueType（int / decimal / bool / string）决定，与 sys_config 表一致。
type ConfigItem struct {
	Key         string `json:"key"`
	Value       string `json:"value"`
	ValueType   string `json:"valueType"`
	Description string `json:"description"`
}

// ConfigUpdate 是批量修改系统参数中的一项。
type ConfigUpdate struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// TravelMode 是一种出行方式及其排放因子（UC-14 可配置项）。
type TravelMode struct {
	Code     string  `json:"code"`
	Name     string  `json:"name"`
	EFBase   float64 `json:"efBase"`
	EFActual float64 `json:"efActual"`
	Enabled  bool    `json:"enabled"`
}

// TravelModeUpdate 是后台更新出行方式的入参（nil 表示不修改该字段）。
type TravelModeUpdate struct {
	Name     *string  `json:"name"`
	EFBase   *float64 `json:"efBase"`
	EFActual *float64 `json:"efActual"`
	Enabled  *bool    `json:"enabled"`
}

// AdminStore 是运营后台端口（M8）：商品维护、订单状态流转、积分对账、参数配置。
type AdminStore interface {
	ListAllProducts(ctx context.Context, q AdminProductQuery) (Page[Product], error)
	CreateProduct(ctx context.Context, in AdminProductInput) (Product, error)
	UpdateProduct(ctx context.Context, id int64, in ProductUpdate) (Product, error)
	SetSKUStock(ctx context.Context, skuID int64, stock int) (SKU, error)
	ListAllOrders(ctx context.Context, q AdminOrderQuery) (Page[Order], error)
	// TransitionOrder 按状态机执行 PENDING→PAID→SHIPPED→COMPLETED。
	TransitionOrder(ctx context.Context, orderID int64, to, reason string) (Order, error)
	ReconcilePoints(ctx context.Context) (ReconcileResult, error)

	// ListConfig / UpdateConfig 读、批量写系统参数（sys_config）。
	ListConfig(ctx context.Context) ([]ConfigItem, error)
	// UpdateConfig 批量改参数，并把每次变更写入 audit_log（BR-14：操作人/前后值/理由）。
	UpdateConfig(ctx context.Context, operatorID int64, updates []ConfigUpdate) ([]ConfigItem, error)
	// UpdateTravelMode 更新一种出行方式的名称 / 排放因子 / 启用状态（travel_mode），同样留审计。
	UpdateTravelMode(ctx context.Context, operatorID int64, code string, in TravelModeUpdate) (TravelMode, error)
	// ExpirePoints 清算已到期批次：写 EXPIRE 反向流水、扣余额、批次置 EXPIRED。
	ExpirePoints(ctx context.Context) (ExpireResult, error)
	// AdjustPoints 人工调增 / 调减积分（UC-15），必须写 audit_log。
	AdjustPoints(ctx context.Context, operatorID int64, in AdjustPointsInput) (AdjustResult, error)
	// ListAuditLogs 按操作人 / 动作 / 对象检索审计日志（UC-19）。
	ListAuditLogs(ctx context.Context, q AuditQuery) (Page[AuditLog], error)
}

// configKeyNumericRules 是"取 0 或负值会让主链路失效"的参数语义约束：
// carbon_per_point=0 会让积分换算除零，points_expire_days=0 会让新批次立即过期，
// distance 区间反了会让所有行程被拒。不在表里的键只按 value_type 校验，
// 后续新增参数无需改代码。
var configKeyNumericRules = map[string]func(n float64) bool{
	"carbon_per_point":     func(n float64) bool { return n > 0 },
	"daily_points_cap":     func(n float64) bool { return n >= 0 }, // 0 表示不限制
	"trip_min_distance_km": func(n float64) bool { return n > 0 },
	"trip_max_distance_km": func(n float64) bool { return n > 0 },
	"points_expire_days":   func(n float64) bool { return n > 0 },
}

// ValidateConfigValue 校验一条系统参数能否写入：
// 先按 value_type 校验字面量（int / decimal / bool / string），再套 configKeyNumericRules。
func ValidateConfigValue(key, valueType, value string) error {
	key = strings.TrimSpace(key)
	raw := strings.TrimSpace(value)
	if key == "" || raw == "" || len(raw) > 255 {
		return ErrInvalidInput
	}

	var numeric float64
	switch strings.ToLower(strings.TrimSpace(valueType)) {
	case "int":
		n, err := strconv.ParseInt(raw, 10, 64)
		if err != nil {
			return ErrInvalidInput
		}
		numeric = float64(n)
	case "decimal":
		f, err := strconv.ParseFloat(raw, 64)
		if err != nil || math.IsNaN(f) || math.IsInf(f, 0) {
			return ErrInvalidInput
		}
		numeric = f
	case "bool":
		switch strings.ToLower(raw) {
		case "true", "false", "1", "0":
			return nil
		default:
			return ErrInvalidInput
		}
	case "string":
		return nil
	default:
		return ErrInvalidInput
	}

	if rule, ok := configKeyNumericRules[key]; ok && !rule(numeric) {
		return ErrInvalidInput
	}
	return nil
}

// ValidateTravelModeUpdate 校验出行方式更新入参：至少一个字段，且数值必须有限、非负。
// efBase 必须为正；efActual 与 efBase 的大小关系由仓储层结合当前值判定。
func ValidateTravelModeUpdate(in TravelModeUpdate) error {
	if in.Name == nil && in.EFBase == nil && in.EFActual == nil && in.Enabled == nil {
		return ErrInvalidInput
	}
	if in.Name != nil && strings.TrimSpace(*in.Name) == "" {
		return ErrInvalidInput
	}
	if in.EFBase != nil && (math.IsNaN(*in.EFBase) || math.IsInf(*in.EFBase, 0) || *in.EFBase <= 0) {
		return ErrInvalidInput
	}
	if in.EFActual != nil && (math.IsNaN(*in.EFActual) || math.IsInf(*in.EFActual, 0) || *in.EFActual < 0) {
		return ErrInvalidInput
	}
	return nil
}
