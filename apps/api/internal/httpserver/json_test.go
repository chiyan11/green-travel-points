package httpserver

import (
	"net/http"
	"strings"
	"testing"

	"greentravelpoints/internal/app"
)

// 审查 P2-19：请求体解码必须统一 —— 1MiB 上限、拒绝未知字段、拒绝尾随内容；
// cancel / transition 这类「body 可省」的接口也不能再静默吞掉解码错误。

func TestDecodeRejectsOversizedBody(t *testing.T) {
	h := secureRouter(true)
	// 1MiB 上限之外再给 1KB 的字符串内容，让解码器在字符串中途触到限长。
	body := `{"reason":"` + strings.Repeat("a", maxJSONBodyBytes+1024) + `"}`
	rec, env := doJSON(t, h, http.MethodPut, "/api/v1/admin/config", body,
		map[string]string{"X-User-Id": "1002"})
	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("超长请求体：期望 413，实际 %d（%s）", rec.Code, rec.Body.String())
	}
	if env.Code != CodeInvalidArgument {
		t.Fatalf("错误码 = %q，期望 %q", env.Code, CodeInvalidArgument)
	}
}

func TestDecodeRejectsTrailingJSON(t *testing.T) {
	h := secureRouter(true)
	for _, body := range []string{
		`{"reason":"x"}{"reason":"y"}`,
		`{"reason":"x"} null`,
	} {
		rec, env := doJSON(t, h, http.MethodPut, "/api/v1/admin/config", body,
			map[string]string{"X-User-Id": "1002"})
		if rec.Code != http.StatusBadRequest {
			t.Errorf("body=%s：期望 400，实际 %d", body, rec.Code)
			continue
		}
		if env.Code != CodeInvalidArgument {
			t.Errorf("body=%s：错误码 = %q，期望 %q", body, env.Code, CodeInvalidArgument)
		}
	}
}

func TestDecodeRejectsUnknownFields(t *testing.T) {
	h := secureRouter(true)
	body := `{"productId":1,"receiver":"本人","unexpected":true}`
	rec, env := doJSON(t, h, http.MethodPost, "/api/v1/orders", body,
		map[string]string{"X-User-Id": "1002"})
	if rec.Code != http.StatusBadRequest || env.Code != CodeInvalidArgument {
		t.Fatalf("未知字段：期望 400/%s，实际 %d/%q", CodeInvalidArgument, rec.Code, env.Code)
	}
}

func TestCancelOrderRejectsMalformedBody(t *testing.T) {
	h := secureRouter(true)
	rec, env := doJSON(t, h, http.MethodPost, "/api/v1/orders/1/cancel", "not-json",
		map[string]string{"X-User-Id": "1002"})
	if rec.Code != http.StatusBadRequest || env.Code != CodeInvalidArgument {
		t.Fatalf("坏 body 的取消请求：期望 400/%s，实际 %d/%q", CodeInvalidArgument, rec.Code, env.Code)
	}
}

func TestCancelOrderAllowsEmptyBody(t *testing.T) {
	h := NewRouter(Options{
		Logger:     discardLogger(),
		Env:        "test",
		AuthSecret: "test-secret",
		Orders: fakeOrders{cancel: func(userID, orderID int64, reason string) (app.Order, error) {
			if reason != "" {
				t.Errorf("没带 body 时 reason 应为空，实际 %q", reason)
			}
			return app.Order{ID: orderID, Status: "CANCELLED"}, nil
		}},
		AllowDevHeader: true,
	})
	rec, env := doJSON(t, h, http.MethodPost, "/api/v1/orders/1/cancel", "",
		map[string]string{"X-User-Id": "1002"})
	if rec.Code != http.StatusOK || env.Code != CodeOK {
		t.Fatalf("不带 body 的取消请求应照常受理：实际 %d/%q", rec.Code, env.Code)
	}
}

func TestAdminTransitionRejectsMalformedBody(t *testing.T) {
	h := secureRouter(true)
	rec, env := doJSON(t, h, http.MethodPost, "/api/v1/admin/orders/1/confirm", "{oops",
		map[string]string{"X-User-Id": "1002"})
	if rec.Code != http.StatusBadRequest || env.Code != CodeInvalidArgument {
		t.Fatalf("坏 body 的状态流转：期望 400/%s，实际 %d/%q", CodeInvalidArgument, rec.Code, env.Code)
	}
}
