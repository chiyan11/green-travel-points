# AGENTS.md

Project-specific guidance for AI coding agents.

## 本项目的约束优先于 Astryx 官方约定

`apps/web` 是绿行积分的前端（React 19 + Vite + TanStack Query，仓库根 `AGENTS.md` 是总纲）。下文的 ASTRYX 区块由 `npx astryx init` 生成，是 Astryx 的官方用法说明；两者冲突时以仓库根 `AGENTS.md` 为准，具体是：

- **样式分层不变**：全站业务样式只写在 `src/styles.css`，它自带设计 token（第 1 节 `:root` 的 `--green-*` / `--amber-*` / `--ink-*` / `--fs-*` / `--sp-*` / `--r-*` / `--shadow`），页面类名只用这些变量、不写字面量。不要新建按页面拆分的 css；第 18 节「兼容层」是给运营后台 `/admin` 的旧类名用的，新页面不要再往这一层加东西。
- **不引入 StyleX / Tailwind 编译器**：禁止 `stylex.create()`、`xstyle`、utility class。
- **用户端页面用语义 HTML + 自有类名**：用户端 8 个页面（首页 / 行程 / 积分 / 商城 / 详情 / 兑换记录 / 我的 / 登录）+ 后台 3 屏（商品列表 / 新建商品 / 订单）已经是「语义标签 + `styles.css` 类名」的形态，只有少量控件（如 `Theme`、`LinkProvider`）来自 Astryx；后台 `/admin` 仍用 Astryx 的 `Button` / `Badge` / `Table` 等组件与旧类名。改用户端页面时保持这个分工，不要把整页换回 Astryx 组件、也不要为了一个控件新建依赖。
- **图标手写、图表用 CSS**：`src/components/icons.tsx` 存 24×24 单色线性 SVG（`currentColor` / `strokeWidth=1.75`）；近 7 日碳减排是 CSS 柱状图。**不引入图标库、图表库**。
- **测试契约是硬约束**：页面测试（`src/pages/*.test.tsx`）与 E2E（`e2e/main-flow.spec.ts`、`e2e/admin-console.e2e.ts`）依赖的 label、文案、`data-testid`、`.stat-card` / `.product-card` 等选择器，改页面时必须保住；改文案要同步改测试。
- **主题只有一个**：品牌子主题在 `src/themes/green-travel/greenTravelTheme.ts`（`defineTheme({ extends: neutralTheme })`，品牌绿 `#177a4d` + 中文字体回退），产物 `green-travel.js` 与 `greenTravelTheme.css` 一起提交；改完必须重跑 `npx astryx theme build src/themes/green-travel/greenTravelTheme.ts`。`src/main.tsx` 里 `<Theme theme={greenTravelTheme} mode="light">`，`mode` 必须显式 `light`（默认 `system` 会跟随系统渲染成暗色）；样式表导入顺序（fonts → reset → astryx → 主题 css → styles.css）不要改。

已验证的组件用法（踩过的坑，别再踩一遍）：

| 组件 | 关键约束 |
| --- | --- |
| `Button` | 文案走 `label` 不是 children；`variant` 只有 `primary / secondary / ghost / destructive`；`isDisabled` 只渲染 `aria-disabled`，表单提交处仍需自己挡一道 |
| `TextInput` | 必传 `label`；`onChange: (value, e) => void`；`type` 不支持 `number` / `datetime-local`，数字与时间用 `Field` + 原生 `<input className="native-input">` |
| `Badge` | `label` + `variant`（`neutral / info / success / warning / error`），没有 `color` |
| `Card` | `padding` 用 spacing step（数字），`elevation: none / low / med / high` |
| `Field` | `label` + 必填 `inputID`，子节点是自己写的原生控件 |
<!-- ASTRYX:START -->
Astryx v0.7.17 · 174 components
CLI: run every command as `npx astryx <cmd>` (shown below as `astryx ...`).

SETUP (once, in app entry e.g. main.tsx) — without these, components render unstyled:
  import "@astryxdesign/core/reset.css";
  import "@astryxdesign/core/astryx.css";

WORKFLOW — start every page from a template, never from scratch:
1. `astryx build "<idea>"` — names the template to scaffold from and the parts it lacks.
2. `astryx template <name> <path>` — scaffold it; keep its frame, gap and padding; replace the content.
3. `astryx component <Name>` to read props before using one; `astryx template <Block>` prints a block example to read for a part it lacks. Astryx names differ from other libraries: buttons, badges, tokens and tabs take their text in `label` (Text, Heading and Link take children), booleans start with `is` or `has` (`isOpen`, `isDisabled`), and every input takes `value` and `onChange`.
Changing an existing page? Keep it: skip step 2 and add components inside its sections; read a block example when it helps.
Before you deliver a build, run `astryx doctor --built <dir>`; don't deliver it if it fails.

RULES:
- No <div> — components handle all layout and spacing.
- Read `astryx docs layout` before changing a template's frame.
- Dense data = rows (Table, List/Item), not Card-wrapped lists. Badge = counts only.
- Style with component props first, then tokens (var(--color-*|--spacing-*|--radius-*)). No raw hex/px. No StyleX/Tailwind compiler here — don't use xstyle/utility classes directly. Some templates use StyleX and need a compiler plugin; see `astryx docs styling-overview`.
- THEMES: palettes and custom colors go through the theme, never :root overrides; so does a denser or roomier app (`editor set <name> spacing tight`), never gap and padding changes page by page. A change to one kind of part (the sidebar, its selected item, the gaps in a form) is a theme change too, not props on one page: `editor how "<ask>"` names the `set … --on <part>` line. No app yet: `astryx experimental theme preview app <dir>` makes a starter app that shows the theme. To make or change a theme (colors, fonts, corners, spacing, a look from a picture), prefer the experimental theme editor over hand-written theme files: `astryx experimental theme editor new <name> --like neutral`, then `editor set <name> <thing> <value>` or `editor how "<what you want>"`; wire it with `astryx theme add <name>`; after each change look at it with `astryx experimental theme editor show <name> --png` (takes pictures of real components) and open the PNGs it writes; a theme change is not done until you have looked at the pictures. In code, extend an imported theme to customize it; `theme eject <slug>` only forks source. See `astryx docs theme`.
- SELF-CHECK: re-read the file; replace any raw <div>/<span>, imported .css/@apply, or hardcoded #hex/px with the component or a token.

`astryx help` lists every command. Key ones beyond the workflow:
  search "<query>"   find any component / hook / doc / template
  discover <words>   integrations you could add
  docs <topic>       getting-started, principles, tokens, theme … (`astryx docs` lists all)
  docs cli           commands and API reference
  docs authoring     build an integration package, and every field it holds
  upgrade --from <old version> --apply   run after a dependency bump
To remove this block, run `astryx init --remove-agents`, which removes it from the agent files at the project root and in the folder you run it in; from any other file it leaves, delete everything from ASTRYX:START to ASTRYX:END.
<!-- ASTRYX:END -->
