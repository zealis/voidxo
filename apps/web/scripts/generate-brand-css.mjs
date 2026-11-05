// 构建期生成品牌色板 CSS：让首屏直接呈现品牌色，不必等客户端 onMount 调用
// setColorScheme 后重绘，避免「默认紫 → 品牌蓝」的跳动。
//
// 算法与 mdui 的 internal/colorScheme.js 保持一致（Scheme + CorePalette 补齐
// 8 个 surface token）。只覆盖 --mdui-color-*-light / -dark 的具体取值，
// mdui.css 自身已把 --mdui-color-* 指向这两组变量，因此深浅色切换仍然生效。
//
// 用法：node ./scripts/generate-brand-css.mjs   （改品牌色后重新运行并提交产物）
import { writeFileSync } from 'node:fs';
import { fileURLToPath } from 'node:url';
import { dirname, resolve } from 'node:path';
import {
  argbFromHex,
  blueFromArgb,
  CorePalette,
  greenFromArgb,
  redFromArgb,
  Scheme
} from '@material/material-color-utilities';

const BRAND_COLOR = '#1a73e8';
const source = argbFromHex(BRAND_COLOR);
const rgb = (argb) => [redFromArgb(argb), greenFromArgb(argb), blueFromArgb(argb)].join(', ');
const kebab = (key) => key.replace(/([A-Z])/g, '-$1').toLowerCase();

const light = Scheme.light(source).toJSON();
const dark = Scheme.dark(source).toJSON();
const palette = CorePalette.of(source);

// material-color-utilities 0.3.0 尚未提供这 8 个 surface token，mdui 内部用 CorePalette 补齐。
Object.assign(light, {
  'surface-dim': palette.n1.tone(87),
  'surface-bright': palette.n1.tone(98),
  'surface-container-lowest': palette.n1.tone(100),
  'surface-container-low': palette.n1.tone(96),
  'surface-container': palette.n1.tone(94),
  'surface-container-high': palette.n1.tone(92),
  'surface-container-highest': palette.n1.tone(90),
  'surface-tint-color': light.primary
});
Object.assign(dark, {
  'surface-dim': palette.n1.tone(6),
  'surface-bright': palette.n1.tone(24),
  'surface-container-lowest': palette.n1.tone(4),
  'surface-container-low': palette.n1.tone(10),
  'surface-container': palette.n1.tone(12),
  'surface-container-high': palette.n1.tone(17),
  'surface-container-highest': palette.n1.tone(22),
  'surface-tint-color': dark.primary
});

const lines = [
  `/* 由 scripts/generate-brand-css.mjs 生成，品牌色 ${BRAND_COLOR}。请勿手改；改品牌色后重跑脚本。 */`,
  '/* 覆盖 mdui.css 的默认紫色板；--mdui-color-* 仍由 mdui.css 按主题指向 -light / -dark。 */',
  ':root {'
];
for (const [key, value] of Object.entries(light)) {
  lines.push(`  --mdui-color-${kebab(key)}-light: ${rgb(value)};`);
}
for (const [key, value] of Object.entries(dark)) {
  lines.push(`  --mdui-color-${kebab(key)}-dark: ${rgb(value)};`);
}
lines.push('}', '');

const here = dirname(fileURLToPath(import.meta.url));
const target = resolve(here, '../src/lib/styles/mdui-brand.css');
writeFileSync(target, lines.join('\n'), 'utf8');
console.log(`已生成 ${target}（${Object.keys(light).length * 2} 个变量，品牌色 ${BRAND_COLOR}）`);
