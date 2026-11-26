package httpserver

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
)

// maxJSONBodyBytes 是单个请求体的上限。本项目最大的合法请求是后台批量改配置，
// 1MiB 足够；超过直接 413，避免客户端用超大 body 把内存吃穿。
const maxJSONBodyBytes = 1 << 20

// errTrailingJSON 表示请求体里 JSON 之后还跟着内容（多发了一个对象、或者
// 拼接了别的字节）。原实现只解码第一个值，后面的内容被静默丢弃。
var errTrailingJSON = errors.New("json: unexpected trailing data")

// decodeJSONBody 统一请求体解码入口：限长 1MiB、拒绝未知字段、拒绝尾随内容。
func decodeJSONBody(w http.ResponseWriter, r *http.Request, dst any) error {
	if r.Body == nil {
		return io.EOF
	}
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxJSONBodyBytes))
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		return err
	}
	return ensureNoTrailing(dec)
}

// decodeJSONOptional 用于「请求体可以省略」的接口（取消订单、状态流转）。
// 空 body 视为没带参数，其余解码错误照常上抛。
func decodeJSONOptional(w http.ResponseWriter, r *http.Request, dst any) error {
	if err := decodeJSONBody(w, r, dst); err != nil && !errors.Is(err, io.EOF) {
		return err
	}
	return nil
}

func ensureNoTrailing(dec *json.Decoder) error {
	var extra json.RawMessage
	switch err := dec.Decode(&extra); {
	case errors.Is(err, io.EOF):
		return nil
	case err != nil:
		return err
	default:
		return errTrailingJSON
	}
}

// failDecode 把解码错误映射成统一响应：超长 413，其余 400。
func failDecode(w http.ResponseWriter, err error) {
	var tooLarge *http.MaxBytesError
	if errors.As(err, &tooLarge) {
		Fail(w, http.StatusRequestEntityTooLarge, CodeInvalidArgument,
			fmt.Sprintf("请求体超过 %d KiB 上限", maxJSONBodyBytes/1024))
		return
	}
	msg := "请求体不是合法 JSON"
	if errors.Is(err, errTrailingJSON) {
		msg = "请求体只能包含一个 JSON 对象"
	}
	Fail(w, http.StatusBadRequest, CodeInvalidArgument, msg)
}

// decodeJSON 是给 handler 用的布尔包装：失败时已经把响应写出去了。
func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) bool {
	if err := decodeJSONBody(w, r, dst); err != nil {
		failDecode(w, err)
		return false
	}
	return true
}
