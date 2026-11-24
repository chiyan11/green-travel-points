/**
 * `npm run gen:api` 的收尾步骤。
 *
 * swagger-typescript-api 生成的 data contract 默认带整文件 `// @ts-nocheck`，
 * 等于让 tsc 跳过这份文件——那样「契约类型能不能编译」就永远进不了门禁。
 * 这里只做两件事：去掉 @ts-nocheck、补一行来源说明，不改任何类型内容。
 */
import { readFileSync, writeFileSync } from 'node:fs'

const target = new URL('../src/types/api.ts', import.meta.url)
const source = '// 由 docs/api/openapi.yaml 生成（npm run gen:api），勿手改；契约变更请改 yaml 后重新生成。'

const original = readFileSync(target, 'utf8')
const patched = original.replace('// @ts-nocheck\n', '').replace('/* eslint-disable */\n', `/* eslint-disable */\n${source}\n`)

if (patched === original) {
  console.error('fix-generated: 未找到 @ts-nocheck 标记，生成物格式可能变化，请检查生成器版本')
  process.exit(1)
}

writeFileSync(target, patched)
