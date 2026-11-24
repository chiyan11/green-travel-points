import { HttpResponse } from 'msw'

/**
 * 构造后端统一壳 `{code, message, data}` 的响应，字段名与
 * `apps/api/internal/httpserver/response.go` 保持一致（成功时 code 恒为 OK）。
 */
export function okJson<T>(data: T, init?: { status?: number; message?: string }) {
  return HttpResponse.json(
    { code: 'OK', message: init?.message ?? 'ok', data },
    { status: init?.status ?? 200 },
  )
}

/** 业务错误响应：`Fail(w, status, code, message)` 不带 data。 */
export function failJson(status: number, code: string, message: string) {
  return HttpResponse.json({ code, message }, { status })
}

/** 分页壳，对应后端 `app.Page[T]`。 */
export function pageOf<T>(items: T[], init?: { page?: number; pageSize?: number; total?: number }) {
  return {
    items,
    page: init?.page ?? 1,
    pageSize: init?.pageSize ?? items.length,
    total: init?.total ?? items.length,
  }
}
