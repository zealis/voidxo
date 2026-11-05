// M3：深浅色与品牌色都走 mdui 官方函数，本项目不再自研主题生成。
// 首屏的 mdui-theme-* class 由 app.html 的内联脚本设置，避免升级前闪烁。

export type VxTheme = 'light' | 'dark' | 'auto';

export const THEME_STORAGE_KEY = 'voidxo-theme';

/** 品牌原始值，来自旧站实测；作为 setColorScheme 的输入。 */
export const BRAND_COLOR = '#1a73e8';

const THEME_CLASSES = ['mdui-theme-light', 'mdui-theme-dark', 'mdui-theme-auto'] as const;

export function themeClass(theme: VxTheme): string {
  return `mdui-theme-${theme}`;
}

export function isVxTheme(value: unknown): value is VxTheme {
  return value === 'light' || value === 'dark' || value === 'auto';
}

/** 切换主题：先落 class（立即生效），再让 mdui 同步内部状态。 */
export function applyTheme(theme: VxTheme): void {
  if (typeof document === 'undefined') return;
  const root = document.documentElement;
  root.classList.remove(...THEME_CLASSES);
  root.classList.add(themeClass(theme));
  try {
    localStorage.setItem(THEME_STORAGE_KEY, theme);
  } catch {
    // 隐私模式下不持久化，仅本次会话生效。
  }
  void import('mdui/functions/setTheme.js').then(({ setTheme }) => setTheme(theme));
}

/** 按品牌色生成整套 MD3 色板；组件 JS 与函数均为按需导入。 */
export function applyBrandColor(hex: string): void {
  if (typeof document === 'undefined') return;
  void import('mdui/functions/setColorScheme.js').then(({ setColorScheme }) => setColorScheme(hex));
}
