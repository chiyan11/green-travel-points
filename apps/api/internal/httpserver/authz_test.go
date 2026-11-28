package httpserver

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-sql-driver/mysql"

	"greentravelpoints/internal/app"
)

type registerErrorConnector struct{ err error }

func (c registerErrorConnector) Connect(context.Context) (driver.Conn, error) {
	return registerErrorConn{err: c.err}, nil
}
func (c registerErrorConnector) Driver() driver.Driver { return registerErrorDriver{err: c.err} }

type registerErrorDriver struct{ err error }

func (d registerErrorDriver) Open(string) (driver.Conn, error) {
	return registerErrorConn{err: d.err}, nil
}

type registerErrorConn struct{ err error }

func (c registerErrorConn) Prepare(string) (driver.Stmt, error) {
	return nil, errors.New("unexpected prepare")
}
func (registerErrorConn) Close() error              { return nil }
func (registerErrorConn) Begin() (driver.Tx, error) { return registerErrorTx{}, nil }
func (c registerErrorConn) ExecContext(context.Context, string, []driver.NamedValue) (driver.Result, error) {
	return nil, c.err
}

type registerErrorTx struct{}

func (registerErrorTx) Commit() error   { return nil }
func (registerErrorTx) Rollback() error { return nil }

func TestRegisterDuplicateUsesMySQLErrorCode(t *testing.T) {
	for _, tt := range []struct {
		name string
		err  error
		want int
		code string
	}{
		{"wrapped duplicate code", fmt.Errorf("insert: %w", &mysql.MySQLError{Number: 1062, Message: "other wording"}), http.StatusConflict, CodeConflict},
		{"duplicate text only", errors.New("duplicate network request"), http.StatusInternalServerError, CodeInternal},
		{"other mysql error", &mysql.MySQLError{Number: 1205, Message: "duplicate"}, http.StatusInternalServerError, CodeInternal},
	} {
		t.Run(tt.name, func(t *testing.T) {
			db := sql.OpenDB(registerErrorConnector{err: tt.err})
			defer db.Close()
			handler := NewAuthHandler(db, "test-secret", false, discardLogger())
			req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register",
				strings.NewReader(`{"phone":"13800000001","password":"demo123456"}`))
			rec := httptest.NewRecorder()
			handler.Register(rec, req)
			if rec.Code != tt.want || !strings.Contains(rec.Body.String(), `"code":"`+tt.code+`"`) {
				t.Fatalf("status=%d body=%s, want status=%d code=%s", rec.Code, rec.Body.String(), tt.want, tt.code)
			}
		})
	}
}

// 这一组用例锁死「身份从哪来」的约定：
//   - 默认只认 Bearer，匿名一律 401；
//   - X-User-Id 是开发开关，只有 AllowDevHeader=true 时才生效；
//   - 登录态永远优先于身份头，令牌里的 userId 必须 > 0。
// 起因：审查 P0-1——X-User-Id 在所有环境生效、前端默认注入 1，等于全站越权。

func secureRouter(allowDevHeader bool) http.Handler {
	return NewRouter(Options{
		Logger:         discardLogger(),
		Env:            "test",
		AuthSecret:     "test-secret",
		Points:         fakePoints{},
		Orders:         fakeOrders{list: func(app.OrderQuery) (app.Page[app.Order], error) { return app.Page[app.Order]{}, nil }},
		Trips:          fakeTrips{list: func(app.TripQuery) (app.Page[app.Trip], error) { return app.Page[app.Trip]{}, nil }},
		Admin:          fakeAdmin{},
		Roles:          fakeRoles{role: "ADMIN"},
		AllowDevHeader: allowDevHeader,
	})
}

func getWithHeaders(h http.Handler, path string, headers map[string]string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, path, nil)
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestAnonymousRequestRejected(t *testing.T) {
	h := secureRouter(false)
	for _, path := range []string{
		"/api/v1/points/summary",
		"/api/v1/trips",
		"/api/v1/orders",
		"/api/v1/admin/config",
	} {
		if rec := getWithHeaders(h, path, nil); rec.Code != http.StatusUnauthorized {
			t.Errorf("%s 匿名访问：期望 401，实际 %d", path, rec.Code)
		}
	}
}

func TestDevHeaderIgnoredWhenDisabled(t *testing.T) {
	h := secureRouter(false)
	for _, id := range []string{"1", "1002"} {
		rec := getWithHeaders(h, "/api/v1/admin/config", map[string]string{"X-User-Id": id})
		if rec.Code != http.StatusUnauthorized {
			t.Errorf("开关关闭时 X-User-Id=%s：期望 401，实际 %d", id, rec.Code)
		}
	}
}

func TestDevHeaderAcceptedWhenEnabled(t *testing.T) {
	h := secureRouter(true)
	if rec := getWithHeaders(h, "/api/v1/points/summary", map[string]string{"X-User-Id": "1002"}); rec.Code != http.StatusOK {
		t.Errorf("开关开启时 X-User-Id=1002：期望 200，实际 %d", rec.Code)
	}
	if rec := getWithHeaders(h, "/api/v1/admin/config", map[string]string{"X-User-Id": "1002"}); rec.Code != http.StatusOK {
		t.Errorf("开关开启时管理员访问 /admin/config：期望 200，实际 %d", rec.Code)
	}
}

func TestDevHeaderRequiresPositiveInteger(t *testing.T) {
	h := secureRouter(true)
	for _, id := range []string{"0", "-5", "abc", "", "1.5"} {
		rec := getWithHeaders(h, "/api/v1/points/summary", map[string]string{"X-User-Id": id})
		if rec.Code != http.StatusUnauthorized {
			t.Errorf("X-User-Id=%q：期望 401，实际 %d", id, rec.Code)
		}
	}
}

func TestBearerTakesPrecedenceOverDevHeader(t *testing.T) {
	h := NewAuthHandler(nil, "test-secret", true, discardLogger())
	token, err := h.issue(7)
	if err != nil {
		t.Fatalf("签发令牌失败: %v", err)
	}
	var got int64
	next := http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		got = UserID(r.Context())
	})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/points/summary", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("X-User-Id", "1002") // 故意与令牌冲突
	rec := httptest.NewRecorder()
	h.Require(next).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("状态码 = %d，期望 200", rec.Code)
	}
	if got != 7 {
		t.Fatalf("上下文 userId = %d，期望取令牌里的 7（身份头不得覆盖登录态）", got)
	}
}

func TestBearerSchemeCaseInsensitive(t *testing.T) {
	h := NewAuthHandler(nil, "test-secret", false, discardLogger())
	token, err := h.issue(7)
	if err != nil {
		t.Fatal(err)
	}
	for _, scheme := range []string{"Bearer", "bearer", "BEARER", "Basic"} {
		t.Run(scheme, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.Header.Set("Authorization", scheme+" "+token)
			rec := httptest.NewRecorder()
			h.Require(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
			})).ServeHTTP(rec, req)
			want := http.StatusOK
			if scheme == "Basic" {
				want = http.StatusUnauthorized
			}
			if rec.Code != want {
				t.Fatalf("status = %d, want %d", rec.Code, want)
			}
		})
	}
}

func TestTokenWithZeroUserIDRejected(t *testing.T) {
	h := NewAuthHandler(nil, "test-secret", false, discardLogger())
	token, err := h.issue(0)
	if err != nil {
		t.Fatalf("签发令牌失败: %v", err)
	}
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("userId=0 的令牌不应进入业务处理，实际 userId=%d", UserID(r.Context()))
	})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/points/summary", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	h.Require(next).ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("状态码 = %d，期望 401", rec.Code)
	}
}

func TestVerifyRejectsNonPositiveUserID(t *testing.T) {
	h := NewAuthHandler(nil, "test-secret", false, discardLogger())
	for _, id := range []int64{0, -1} {
		token, err := h.issue(id)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := h.verify(token); err == nil {
			t.Fatalf("verify accepted user ID %d", id)
		}
	}
}

func TestRequirePutsUserInContext(t *testing.T) {
	h := NewAuthHandler(nil, "test-secret", false, discardLogger())
	token, err := h.issue(42)
	if err != nil {
		t.Fatalf("签发令牌失败: %v", err)
	}
	var got int64
	next := http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		got = UserID(r.Context())
	})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/points/summary", nil).WithContext(context.Background())
	req.Header.Set("Authorization", "Bearer "+token)
	h.Require(next).ServeHTTP(httptest.NewRecorder(), req)
	if got != 42 {
		t.Fatalf("上下文 userId = %d，期望 42", got)
	}
}
