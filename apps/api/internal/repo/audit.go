package repo

import (
	"context"
	"database/sql"
	"strings"

	"greentravelpoints/internal/app"
)

// 审计日志（BR-14 / NFR-12 / UC-19）：人工调账与参数变更必须留痕，
// 记录操作人、对象、前后值与理由；本表只追加，不提供更新/删除路径。

// auditEntry 是一次待写入的审计记录。
type auditEntry struct {
	operatorID  int64
	action      string // ADJUST / CONFIG_UPDATE / TRAVEL_MODE_UPDATE
	targetType  string // point_account / sys_config / travel_mode
	targetID    int64
	targetKey   string
	amount      int64
	beforeValue string
	afterValue  string
	reason      string
}

// writeAudit 在调用方事务内写一条审计记录，返回其 id（调账流水会用 biz_ref_type='audit_log' 回指它）。
func writeAudit(ctx context.Context, tx *sql.Tx, e auditEntry) (int64, error) {
	res, err := tx.ExecContext(ctx,
		`INSERT INTO audit_log
		   (operator_id, action, target_type, target_id, target_key,
		    amount, before_value, after_value, reason)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		e.operatorID, e.action, e.targetType, e.targetID, e.targetKey,
		e.amount, e.beforeValue, e.afterValue, e.reason)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// ListAuditLogs 分页检索审计日志（UC-19），按 id 倒序（最新在前）。
// 过滤条件都可缺省：operatorId / action / targetType / targetKey。
func (s *Store) ListAuditLogs(ctx context.Context, q app.AuditQuery) (app.Page[app.AuditLog], error) {
	page, pageSize := normalizePage(q.Page, q.PageSize)

	where := []string{"1 = 1"}
	args := make([]any, 0, 4)
	if q.OperatorID > 0 {
		where = append(where, "operator_id = ?")
		args = append(args, q.OperatorID)
	}
	if action := strings.ToUpper(strings.TrimSpace(q.Action)); action != "" {
		where = append(where, "action = ?")
		args = append(args, action)
	}
	if targetType := strings.TrimSpace(q.TargetType); targetType != "" {
		where = append(where, "target_type = ?")
		args = append(args, targetType)
	}
	if targetKey := strings.TrimSpace(q.TargetKey); targetKey != "" {
		where = append(where, "target_key = ?")
		args = append(args, targetKey)
	}
	whereSQL := strings.Join(where, " AND ")

	var total int64
	if err := s.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM audit_log WHERE `+whereSQL, args...).Scan(&total); err != nil {
		return app.Page[app.AuditLog]{}, err
	}

	rows, err := s.db.QueryContext(ctx,
		`SELECT id, operator_id, action, target_type, target_id, target_key,
		        amount, before_value, after_value, reason, created_at
		 FROM audit_log
		 WHERE `+whereSQL+`
		 ORDER BY id DESC
		 LIMIT ? OFFSET ?`,
		append(args, pageSize, (page-1)*pageSize)...)
	if err != nil {
		return app.Page[app.AuditLog]{}, err
	}
	defer rows.Close()

	items := make([]app.AuditLog, 0, pageSize)
	for rows.Next() {
		var it app.AuditLog
		if err := rows.Scan(&it.ID, &it.OperatorID, &it.Action, &it.TargetType, &it.TargetID,
			&it.TargetKey, &it.Amount, &it.BeforeValue, &it.AfterValue, &it.Reason,
			&it.CreatedAt); err != nil {
			return app.Page[app.AuditLog]{}, err
		}
		items = append(items, it)
	}
	if err := rows.Err(); err != nil {
		return app.Page[app.AuditLog]{}, err
	}
	return app.Page[app.AuditLog]{Items: items, Page: page, PageSize: pageSize, Total: total}, nil
}
