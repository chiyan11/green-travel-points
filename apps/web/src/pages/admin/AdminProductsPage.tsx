import { Fragment, useState, type FormEvent, type ReactNode } from 'react'
import { Link, useSearchParams } from 'react-router-dom'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { listAdminProducts, setSKUStock, updateAdminProductStatus } from '@/services/admin'
import { PRODUCT_STATUS_META, type AdminProduct } from '@/types/admin'
import type { ProductStatus } from '@/types/domain'
import { formatPoints } from '@/lib/format'
import { ApiError } from '@/lib/http'

const STATUS_FILTERS: Array<{ value: ProductStatus | ''; label: string }> = [
  { value: '', label: '全部' },
  { value: 'DRAFT', label: '草稿' },
  { value: 'ONLINE', label: '已上架' },
  { value: 'OFFLINE', label: '已下架' },
]

/** 商品管理：列表 + 上下架 + 展开改库存。 */
export function AdminProductsPage(): ReactNode {
  const queryClient = useQueryClient()
  const [searchParams] = useSearchParams()
  const [status, setStatus] = useState<ProductStatus | ''>('')
  const [keyword, setKeyword] = useState('')
  const [keywordInput, setKeywordInput] = useState('')
  const [expanded, setExpanded] = useState<string | null>(null)
  const [stockDraft, setStockDraft] = useState<Record<string, string>>({})
  const [error, setError] = useState('')
  const createdName = searchParams.get('created') ?? ''

  const { data, isPending, isError, error: queryError } = useQuery({
    queryKey: ['admin-products', status, keyword],
    queryFn: () => listAdminProducts({ status, keyword }),
  })

  const invalidate = () => void queryClient.invalidateQueries({ queryKey: ['admin-products'] })

  const statusMutation = useMutation({
    mutationFn: ({ id, next }: { id: string; next: ProductStatus }) =>
      updateAdminProductStatus(id, next),
    onSuccess: () => {
      setError('')
      invalidate()
    },
    onError: (err: unknown) => setError(err instanceof ApiError ? err.message : '上下架失败'),
  })

  const stockMutation = useMutation({
    mutationFn: ({ skuId, stock }: { skuId: string; stock: number }) => setSKUStock(skuId, stock),
    onSuccess: () => {
      setError('')
      invalidate()
    },
    onError: (err: unknown) => setError(err instanceof ApiError ? err.message : '改库存失败'),
  })

  const onSearch = (e: FormEvent) => {
    e.preventDefault()
    setKeyword(keywordInput.trim())
  }

  const onExpand = (p: AdminProduct) => {
    setError('')
    if (expanded === p.id) {
      setExpanded(null)
      return
    }
    setExpanded(p.id)
    const draft: Record<string, string> = {}
    for (const sku of p.skus) draft[sku.id] = String(sku.stock)
    setStockDraft(draft)
  }

  const onSaveStock = (skuId: string) => {
    const raw = stockDraft[skuId] ?? ''
    const value = Number(raw)
    if (raw.trim() === '' || !Number.isInteger(value) || value < 0) {
      setError('库存必须是非负整数')
      return
    }
    stockMutation.mutate({ skuId, stock: value })
  }

  const products = data?.items ?? []

  return (
    <>
      {createdName && <p className="badge badge--ok">已创建：{createdName}</p>}

      <div className="filter-bar">
        {STATUS_FILTERS.map((f) => (
          <button
            key={f.value || 'ALL'}
            className={status === f.value ? 'chip chip--active' : 'chip'}
            onClick={() => setStatus(f.value)}
          >
            {f.label}
          </button>
        ))}
        <form className="filter-bar filter-bar--inline" onSubmit={onSearch}>
          <input
            className="filter-input"
            value={keywordInput}
            onChange={(e) => setKeywordInput(e.target.value)}
            placeholder="商品名称 / 描述"
            aria-label="搜索商品"
          />
          <button type="submit" className="btn btn--ghost">
            搜索
          </button>
        </form>
        <Link className="btn btn--inline" to="/admin/products/new">
          新建商品
        </Link>
      </div>

      {error && <p className="field-error">{error}</p>}

      <section className="card">
        <div className="section-head">
          <h3>商品列表</h3>
          <span className="muted">共 {data?.total ?? 0} 件</span>
        </div>

        {isPending ? (
          <p className="empty-state">加载中…</p>
        ) : isError ? (
          <p className="field-error">
            {queryError instanceof ApiError && queryError.status === 403
              ? '当前账号没有管理员权限，请用 ADMIN 账号登录。'
              : queryError instanceof Error
                ? queryError.message
                : '加载失败'}
          </p>
        ) : products.length === 0 ? (
          <p className="empty-state">还没有商品，点「新建商品」建一件。</p>
        ) : (
          <table className="data-table">
            <thead>
              <tr>
                <th>ID</th>
                <th>名称</th>
                <th>分类</th>
                <th>状态</th>
                <th>最低价</th>
                <th>总库存</th>
                <th>操作</th>
              </tr>
            </thead>
            <tbody>
              {products.map((p) => {
                const meta = PRODUCT_STATUS_META[p.status]
                return (
                  <Fragment key={p.id}>
                    <tr>
                      <td><code>{p.id}</code></td>
                      <td>{p.name}</td>
                      <td>{p.category}</td>
                      <td>
                        <span className={`badge ${meta.badge}`}>{meta.label}</span>
                      </td>
                      <td>{formatPoints(p.pointsPrice)} 分</td>
                      <td>{p.stock}</td>
                      <td>
                        <div className="row-actions">
                          {p.status !== 'ONLINE' ? (
                            <button
                              className="btn btn--ghost"
                              disabled={statusMutation.isPending}
                              onClick={() => statusMutation.mutate({ id: p.id, next: 'ONLINE' })}
                            >
                              上架
                            </button>
                          ) : (
                            <button
                              className="btn btn--ghost"
                              disabled={statusMutation.isPending}
                              onClick={() => statusMutation.mutate({ id: p.id, next: 'OFFLINE' })}
                            >
                              下架
                            </button>
                          )}
                          <button className="btn btn--ghost" onClick={() => onExpand(p)}>
                            {expanded === p.id ? '收起库存' : '改库存'}
                          </button>
                        </div>
                      </td>
                    </tr>
                    {expanded === p.id && (
                      <tr>
                        <td colSpan={7}>
                          {p.skus.length === 0 ? (
                            <p className="muted">该商品没有 SKU。</p>
                          ) : (
                            <div className="sku-panel">
                              {p.skus.map((sku) => (
                                <div className="sku-row" key={sku.id}>
                                  <span className="sku-row__name">{sku.name}</span>
                                  <span className="muted">
                                    {formatPoints(sku.pointsPrice)} 分 · 已售 {sku.soldCount}
                                  </span>
                                  <label className="sku-row__stock">
                                    库存
                                    <input
                                      value={stockDraft[sku.id] ?? ''}
                                      onChange={(e) =>
                                        setStockDraft((prev) => ({ ...prev, [sku.id]: e.target.value }))
                                      }
                                      inputMode="numeric"
                                    />
                                  </label>
                                  <button
                                    className="btn"
                                    disabled={stockMutation.isPending}
                                    onClick={() => onSaveStock(sku.id)}
                                  >
                                    保存
                                  </button>
                                </div>
                              ))}
                            </div>
                          )}
                        </td>
                      </tr>
                    )}
                  </Fragment>
                )
              })}
            </tbody>
          </table>
        )}
      </section>
    </>
  )
}
