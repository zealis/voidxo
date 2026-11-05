// mdui 是基于 Lit 的 Web Components，依赖 customElements 与 Shadow DOM。
// M1/M2：组件 JS 一律按需导入，且只在客户端注册；SSR 输出的是原生 <mdui-*>
// 标签，服务端模块图里不应出现任何 mdui 组件导入。

const loaders = {
  button: () => import('mdui/components/button.js'),
  'button-icon': () => import('mdui/components/button-icon.js'),
  'text-field': () => import('mdui/components/text-field.js'),
  checkbox: () => import('mdui/components/checkbox.js'),
  radio: () => import('mdui/components/radio.js'),
  'radio-group': () => import('mdui/components/radio-group.js'),
  switch: () => import('mdui/components/switch.js'),
  select: () => import('mdui/components/select.js'),
  'menu-item': () => import('mdui/components/menu-item.js'),
  dialog: () => import('mdui/components/dialog.js'),
  snackbar: () => import('mdui/components/snackbar.js'),
  tooltip: () => import('mdui/components/tooltip.js'),
  tabs: () => import('mdui/components/tabs.js'),
  tab: () => import('mdui/components/tab.js'),
  'tab-panel': () => import('mdui/components/tab-panel.js')
} as const;

export type MduiComponentName = keyof typeof loaders;

const isBrowser = typeof window !== 'undefined' && typeof customElements !== 'undefined';

const pending = new Map<MduiComponentName, Promise<unknown>>();

/** 按需注册一个组件；重复调用复用同一个 Promise。SSR 期间不执行导入。 */
export function ensureMdui(name: MduiComponentName): Promise<unknown> {
  if (!isBrowser) return Promise.resolve();
  let task = pending.get(name);
  if (!task) {
    task = loaders[name]();
    pending.set(name, task);
  }
  return task;
}

/** 一次注册多个组件，供封装层与试点页共用。 */
export function ensureMduiComponents(names: MduiComponentName[]): Promise<unknown[]> {
  return Promise.all(names.map(ensureMdui));
}

/**
 * M5：在封装层监听 mdui 的 CustomEvent，返回清理函数供 $effect 使用。
 * 不把事件写成模板属性，避免 Svelte 把自定义元素当作静态元素做 a11y 检查。
 */
export function listenMdui(
  element: EventTarget | undefined,
  type: string,
  handler: (event: Event) => void
): (() => void) | undefined {
  if (!element) return undefined;
  element.addEventListener(type, handler);
  return () => element.removeEventListener(type, handler);
}

/** M8：真实浏览器断言或截图前，等待自定义元素升级完成。 */
export function whenDefined(tag: `mdui-${string}`): Promise<CustomElementConstructor | void> {
  if (!isBrowser) return Promise.resolve();
  return customElements.whenDefined(tag);
}
