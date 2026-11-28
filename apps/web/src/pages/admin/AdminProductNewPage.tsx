import { useState, type FormEvent, type ReactNode } from 'react'
import { useNavigate } from 'react-router-dom'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { createAdminProduct } from '@/services/admin'
import type { ProductStatus } from '@/types/domain'
import { ApiError } from '@/lib/http'

/** 种子数据里当前只有一家商户（id=1001），新建商品默认挂它名下。 */
const DEFAULT_MERCHANT_ID = 1001

/** 新建商品：单 SKU（标准款），状态默认草稿，提交后回列表。 */
export function AdminProductNewPage(): ReactNode {
  const navigate = useNavigate()
  const queryClient = useQueryClient()
  const [name, setName] = useState('')
  const [category, setCategory] = useState('')
  const [description, setDescription] = useState('')
  const [pointsPrice, setPointsPrice] = useState('100')
  const [stock, setStock] = useState('10')
  const [limitPerUser, setLimitPerUser] = useState('0')
  const [status, setStatus] = useState<ProductStatus>('DRAFT')
  const [error, setError] = useState('')

  const mutation = useMutation({
    mutationFn: createAdminProduct,
    onSuccess: (product) => {
      // 列表 staleTime 30s：不主动失效的话，返回列表会看到「已创建：X」但表格里没有 X。
      // 用前缀 key 覆盖所有筛选/分页组合。
      void queryClient.invalidateQueries({ queryKey: ['admin-products'] })
      navigate(`/admin/products?created=${encodeURIComponent(product.name)}`)
    },
    onError: (err: unknown) => {
      setError(err instanceof ApiError ? err.message : '创建失败')
    },
  })

  const onSubmit = (e: FormEvent) => {
    e.preventDefault()
    setError('')
    const price = Number(pointsPrice)
    const stockValue = Number(stock)
    const limit = Number(limitPerUser)
    if (name.trim() === '' || category.trim() === '') {
      setError('名称与分类必填')
      return
    }
    if (!Number.isInteger(price) || price <= 0) {
      setError('积分价必须是正整数')
      return
    }
    if (!Number.isInteger(stockValue) || stockValue < 0) {
      setError('库存必须是非负整数')
      return
    }
    if (!Number.isInteger(limit) || limit < 0) {
      setError('每人限兑必须是非负整数')
      return
    }
    mutation.mutate({
      merchantId: DEFAULT_MERCHANT_ID,
      name: name.trim(),
      category: category.trim(),
      description: description.trim(),
      status,
      limitPerUser: limit,
      skus: [{ name: '标准款', pointsPrice: price, stock: stockValue }],
    })
  }

  return (
    <section className="card">
      <div className="section-head">
        <h3>新建商品</h3>
      </div>

      <form onSubmit={onSubmit}>
        <div className="form-field">
          <label htmlFor="name">名称</label>
          <input id="name" value={name} onChange={(e) => setName(e.target.value)} />
        </div>
        <div className="form-field">
          <label htmlFor="category">分类</label>
          <input
            id="category"
            value={category}
            onChange={(e) => setCategory(e.target.value)}
            placeholder="文创 / 生活 / 出行"
          />
        </div>
        <div className="form-field">
          <label htmlFor="description">描述</label>
          <input
            id="description"
            value={description}
            onChange={(e) => setDescription(e.target.value)}
          />
        </div>
        <div className="form-field">
          <label htmlFor="pointsPrice">积分价（标准款 SKU）</label>
          <input
            id="pointsPrice"
            value={pointsPrice}
            onChange={(e) => setPointsPrice(e.target.value)}
            inputMode="numeric"
          />
        </div>
        <div className="form-field">
          <label htmlFor="stock">库存</label>
          <input
            id="stock"
            value={stock}
            onChange={(e) => setStock(e.target.value)}
            inputMode="numeric"
          />
        </div>
        <div className="form-field">
          <label htmlFor="limitPerUser">每人限兑（0 表示不限）</label>
          <input
            id="limitPerUser"
            value={limitPerUser}
            onChange={(e) => setLimitPerUser(e.target.value)}
            inputMode="numeric"
          />
        </div>
        <div className="form-field">
          <label htmlFor="status">初始状态</label>
          <select
            id="status"
            value={status}
            onChange={(e) => setStatus(e.target.value as ProductStatus)}
          >
            <option value="DRAFT">草稿（DRAFT）</option>
            <option value="ONLINE">直接上架（ONLINE）</option>
          </select>
        </div>

        {error && <p className="field-error">{error}</p>}
        <button type="submit" className="btn" disabled={mutation.isPending}>
          {mutation.isPending ? '提交中…' : '创建商品'}
        </button>
      </form>
    </section>
  )
}
