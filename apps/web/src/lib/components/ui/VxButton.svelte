<script lang="ts">
  import { onMount } from 'svelte';
  import type { Snippet } from 'svelte';
  import { ensureMdui } from '#lib/mdui/registry.js';

  // M5：业务页面只面对 Vx* 封装，不直接散落裸 <mdui-*>。
  type Variant = 'elevated' | 'filled' | 'tonal' | 'outlined' | 'text';

  type Props = {
    variant?: Variant;
    type?: 'button' | 'submit' | 'reset';
    disabled?: boolean;
    loading?: boolean;
    href?: string;
    name?: string;
    value?: string;
    ariaLabel?: string;
    onclick?: (event: MouseEvent) => void;
    children: Snippet;
  };

  let {
    variant = 'filled',
    type = 'button',
    disabled = false,
    loading = false,
    href,
    name,
    value,
    ariaLabel,
    onclick,
    children
  }: Props = $props();

  let element: HTMLElement | undefined = $state();

  // 组件 JS 只在客户端按需注册，SSR 输出的是原生标签 + slot 文本。
  onMount(() => {
    void ensureMdui('button');
  });

  // Lit 的布尔属性转换器会把 attribute="false" 判成 true，
  // 因此布尔与枚举值一律在客户端以 property 设置，模板只输出字符串 attribute。
  $effect(() => {
    if (!element) return;
    const node = element as HTMLElement & {
      variant: string;
      disabled: boolean;
      loading: boolean;
      type: string;
      name: string;
      value: string;
    };
    node.variant = variant;
    node.disabled = disabled || loading;
    node.loading = loading;
    // 表单相关值同样以 property 写入，确保 submit / reset 在升级后生效。
    if (!href) {
      node.type = type;
      if (name !== undefined) node.name = name;
      if (value !== undefined) node.value = value;
    }
  });

  // M5：mdui 的事件在封装层用 DOM 监听接住再转给 Svelte 回调。
  // 不把 onclick 写成自定义元素的模板属性，避免 Svelte a11y 把它当静态元素，
  // 也让键盘触发（由 mdui 内部转成 click）走同一条路径。
  $effect(() => {
    const node = element;
    const handler = onclick;
    if (!node || !handler) return;
    const listener = (event: Event) => handler(event as MouseEvent);
    node.addEventListener('click', listener);
    return () => node.removeEventListener('click', listener);
  });
</script>

{#if href}
  <mdui-button bind:this={element} {variant} {href} aria-label={ariaLabel}>
    {@render children()}
  </mdui-button>
{:else}
  <mdui-button bind:this={element} {variant} {type} {name} {value} aria-label={ariaLabel}>
    {@render children()}
  </mdui-button>
{/if}
