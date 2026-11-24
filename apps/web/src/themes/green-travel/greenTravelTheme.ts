/**
 * 绿行积分品牌主题：在 Astryx Neutral 之上把强调色换成品牌绿，
 * 并给字体栈补上中文回退（Figtree 只含拉丁字形，中文继续下落到系统字体）。
 *
 * 改完必须重新生成同目录的 CSS 与 JS：
 *   npx astryx theme build src/themes/green-travel/greenTravelTheme.ts
 */
import {defineTheme} from '@astryxdesign/core/theme';
import {neutralTheme} from '@astryxdesign/theme-neutral';

const cjkFallbacks =
  '-apple-system, BlinkMacSystemFont, "Segoe UI", "PingFang SC", "Microsoft YaHei", "Noto Sans SC", Roboto, Helvetica, Arial, sans-serif';

export const greenTravelTheme = defineTheme({
  name: 'green-travel',
  extends: neutralTheme,

  // 只换字体回退：子主题的 typography 是整体替换，比例与字重必须与 Neutral 一致。
  typography: {
    scale: {base: 14, ratio: 1.2},
    body: {family: 'Figtree', fallbacks: cjkFallbacks},
    heading: {
      family: 'Figtree',
      fallbacks: cjkFallbacks,
      weights: {3: 'bold', 4: 'bold'},
    },
    code: {
      family: 'ui-monospace',
      fallbacks:
        '"SF Mono", Monaco, Consolas, "Liberation Mono", "Courier New", monospace',
    },
  },

  // 强调色走主题令牌，页面里不再写死绿色。
  tokens: {
    '--color-accent': ['#177a4d', '#4ad295'],
    '--color-on-accent': ['#ffffff', '#0b1f16'],
    '--color-accent-muted': ['#e4f2ea', '#123227'],
    '--color-text-accent': ['#105f3a', '#7fe0b4'],
    '--color-icon-accent': ['#177a4d', '#7fe0b4'],
  },
});
