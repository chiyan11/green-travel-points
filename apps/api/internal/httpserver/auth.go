package httpserver

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type authContextKey struct{}

type AuthHandler struct {
	db     *sql.DB
	secret []byte
	logger *slog.Logger
}

func NewAuthHandler(db *sql.DB, secret string, logger *slog.Logger) *AuthHandler {
	return &AuthHandler{db: db, secret: []byte(secret), logger: logger}
}

type authUser struct {
	ID        int64
	Nickname  string
	Phone     string
	CreatedAt time.Time
}

type authRequest struct {
	Phone    string `json:"phone"`
	Account  string `json:"account"`
	Nickname string `json:"nickname"`
	Password string `json:"password"`
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var in authRequest
	if err := decodeJSON(r, &in); err != nil || len(in.Password) < 6 {
		Fail(w, http.StatusBadRequest, CodeInvalidArgument, "密码至少 6 位")
		return
	}
	phone := strings.TrimSpace(in.Phone)
	if phone == "" {
		phone = strings.TrimSpace(in.Account)
	}
	if phone == "" {
		Fail(w, http.StatusBadRequest, CodeInvalidArgument, "手机号不能为空")
		return
	}
	nickname := strings.TrimSpace(in.Nickname)
	if nickname == "" {
		nickname = phone
	}
	if h.db == nil {
		Fail(w, http.StatusServiceUnavailable, CodeUnavailable, "数据库不可用")
		return
	}
	hash, err := passwordHash(in.Password)
	if err != nil {
		FailInternal(w, h.logger, err)
		return
	}
	tx, err := h.db.BeginTx(r.Context(), nil)
	if err != nil {
		FailInternal(w, h.logger, err)
		return
	}
	res, err := tx.ExecContext(r.Context(),
		"INSERT INTO `user` (phone, nickname, password_hash) VALUES (?, ?, ?)",
		phone, nickname, hash)
	if err != nil {
		_ = tx.Rollback()
		if strings.Contains(strings.ToLower(err.Error()), "duplicate") {
			Fail(w, http.StatusConflict, CodeConflict, "手机号已注册")
			return
		}
		FailInternal(w, h.logger, err)
		return
	}
	id, err := res.LastInsertId()
	if err == nil {
		_, err = tx.ExecContext(r.Context(), "INSERT INTO point_account (user_id) VALUES (?)", id)
	}
	if err == nil {
		err = tx.Commit()
	} else {
		_ = tx.Rollback()
	}
	if err != nil {
		FailInternal(w, h.logger, err)
		return
	}
	token, err := h.issue(id)
	if err != nil {
		FailInternal(w, h.logger, err)
		return
	}
	OK(w, map[string]any{"userId": userID(id), "nickname": nickname, "token": token})
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var in authRequest
	if err := decodeJSON(r, &in); err != nil || in.Password == "" {
		Fail(w, http.StatusBadRequest, CodeInvalidArgument, "账号和密码不能为空")
		return
	}
	if h.db == nil {
		Fail(w, http.StatusServiceUnavailable, CodeUnavailable, "数据库不可用")
		return
	}
	account := strings.TrimSpace(in.Account)
	if account == "" {
		account = strings.TrimSpace(in.Phone)
	}
	var u authUser
	var hash, status string
	err := h.db.QueryRowContext(r.Context(),
		"SELECT id, phone, nickname, password_hash, status, created_at FROM `user` WHERE phone = ?",
		account).Scan(&u.ID, &u.Phone, &u.Nickname, &hash, &status, &u.CreatedAt)
	if err != nil || status != "ACTIVE" || !passwordMatches(hash, in.Password) {
		Fail(w, http.StatusUnauthorized, CodeUnauthorized, "账号或密码错误")
		return
	}
	token, err := h.issue(u.ID)
	if err != nil {
		FailInternal(w, h.logger, err)
		return
	}
	OK(w, map[string]any{"token": token, "user": map[string]any{
		"userId": userID(u.ID), "nickname": u.Nickname,
	}})
}

func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	id, _ := r.Context().Value(authContextKey{}).(int64)
	if h.db == nil {
		Fail(w, http.StatusServiceUnavailable, CodeUnavailable, "数据库不可用")
		return
	}
	var u authUser
	err := h.db.QueryRowContext(r.Context(),
		"SELECT id, phone, nickname, created_at FROM `user` WHERE id = ? AND status = 'ACTIVE'",
		id).Scan(&u.ID, &u.Phone, &u.Nickname, &u.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		Fail(w, http.StatusUnauthorized, CodeUnauthorized, "登录已失效")
		return
	}
	if err != nil {
		FailInternal(w, h.logger, err)
		return
	}
	OK(w, map[string]any{"userId": userID(u.ID), "nickname": u.Nickname, "joinedAt": u.CreatedAt})
}

func (h *AuthHandler) Require(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw := strings.TrimSpace(strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer "))
		id, err := h.verify(raw)
		if err != nil {
			Fail(w, http.StatusUnauthorized, CodeUnauthorized, "需要登录")
			return
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), authContextKey{}, id)))
	})
}

func (h *AuthHandler) issue(id int64) (string, error) {
	if len(h.secret) == 0 {
		return "", errors.New("AUTH_SECRET 不能为空")
	}
	exp := time.Now().Add(2 * time.Hour).Unix()
	body := strconv.FormatInt(id, 10) + "." + strconv.FormatInt(exp, 10)
	mac := hmac.New(sha256.New, h.secret)
	_, _ = mac.Write([]byte(body))
	return base64.RawURLEncoding.EncodeToString([]byte(body + "." + hex.EncodeToString(mac.Sum(nil)))), nil
}

func (h *AuthHandler) verify(token string) (int64, error) {
	raw, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil {
		return 0, err
	}
	parts := strings.Split(string(raw), ".")
	if len(parts) != 3 {
		return 0, errors.New("invalid token")
	}
	id, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return 0, err
	}
	exp, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil || time.Now().Unix() >= exp {
		return 0, errors.New("expired token")
	}
	mac := hmac.New(sha256.New, h.secret)
	_, _ = mac.Write([]byte(parts[0] + "." + parts[1]))
	if !hmac.Equal([]byte(parts[2]), []byte(hex.EncodeToString(mac.Sum(nil)))) {
		return 0, errors.New("invalid signature")
	}
	return id, nil
}

// shortcut: SHA-256 is only a temporary dependency-free hash; replace with bcrypt before production.
func passwordHash(password string) (string, error) {
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	sum := sha256.Sum256(append(salt, []byte(password)...))
	return "sha256$" + hex.EncodeToString(salt) + "$" + hex.EncodeToString(sum[:]), nil
}

func passwordMatches(encoded, password string) bool {
	parts := strings.Split(encoded, "$")
	if len(parts) != 3 || parts[0] != "sha256" {
		return false
	}
	salt, err := hex.DecodeString(parts[1])
	if err != nil {
		return false
	}
	sum := sha256.Sum256(append(salt, []byte(password)...))
	return hmac.Equal([]byte(parts[2]), []byte(hex.EncodeToString(sum[:])))
}

func userID(id int64) string { return fmt.Sprintf("u_%d", id) }

func decodeJSON(r *http.Request, dst any) error {
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	return dec.Decode(dst)
}
