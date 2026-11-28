/**
 * 下单幂等键生成。
 *
 * `crypto.randomUUID()` 带 `[SecureContext]` 标记，只有 HTTPS 和 localhost 才有；
 * 部署是 nginx 80 端口，局域网演示（`http://<内网IP>:8088`）时它是 undefined，
 * 直接调用会在点击「确认兑换」的瞬间抛 TypeError——按钮看起来毫无反应。
 *
 * 所以这里按能力降级，统一产出 32 位十六进制字符串（后端只当不透明字符串用）：
 * 1. 有 `randomUUID`：取 UUID 去掉连字符；
 * 2. 只有 `getRandomValues`：取 16 字节转 hex；
 * 3. 两者都没有（极端老浏览器）：抛可读错误，由页面提示用户。
 */
const HEX_BYTE = Array.from({ length: 256 }, (_, i) => i.toString(16).padStart(2, '0'))

function toHex(bytes: Uint8Array): string {
  let out = ''
  for (const byte of bytes) {
    out += HEX_BYTE[byte]
  }
  return out
}

export function newIdempotencyKey(): string {
  const cryptoObj = globalThis.crypto
  if (cryptoObj && typeof cryptoObj.randomUUID === 'function') {
    return cryptoObj.randomUUID().replace(/-/g, '').toLowerCase()
  }
  if (cryptoObj && typeof cryptoObj.getRandomValues === 'function') {
    return toHex(cryptoObj.getRandomValues(new Uint8Array(16)))
  }
  throw new Error('当前浏览器不支持生成兑换凭证，请换用较新的浏览器，或用 HTTPS 访问后再试')
}
