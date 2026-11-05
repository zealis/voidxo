<script lang="ts">
  import { onMount } from 'svelte';
  import type { Snippet } from 'svelte';
  import { ensureMdui } from '#lib/mdui/registry.js';

  type Variant = 'standard' | 'filled' | 'tonal' | 'outlined';

  type Props = {
    variant?: Variant;
    disabled?: boolean;
    loading?: boolean;
    href?: string;
    ariaLabel: string;
    onclick?: (event: MouseEvent) => void;
    children: Snippet;
  };

  let {
    variant = 'standard',
    disabled = false,
    loading = false,
    href,
    ariaLabel,
    onclick,
    children
  }: Props = $props();

  let element: HTMLElement | undefined = $state();

  onMount(() => {
    void ensureMdui('button-icon');
  });

  // 图标沿用项目现有 SVG 经 default slot 传入，不使用 mdui 的 icon 属性（那需要 Material Icons）。
  $effect(() => {
    if (!element) return;
    const node = element as HTMLElement & { variant: string; disabled: boolean; loading: boolean };
    node.variant = variant;
    node.disabled = disabled || loading;
    node.loading = loading;
  });

  $effect(() => {
    const node = element;
    const handler = onclick;
    if (!node || !handler) return;
    const listener = (event: Event) => handler(event as MouseEvent);
    node.addEventListener('click', listener);
    return () => node.removeEventListener('click', listener);
  });
</script>

<mdui-button-icon bind:this={element} {variant} {href} aria-label={ariaLabel}>
  {@render children()}
</mdui-button-icon>
