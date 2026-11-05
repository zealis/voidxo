<script lang="ts">
  import { onMount } from 'svelte';
  import type { Snippet } from 'svelte';
  import type { Chip } from 'mdui/components/chip.js';
  import { ensureMdui, listenMdui } from '#lib/mdui/registry.js';

  // 话题标签场景：链接仍用原生 <a>，保证 SSR 可读、SEO 可索引；
  // mdui-chip 只负责标签的视觉与交互形态。
  type Props = {
    variant?: 'assist' | 'filter' | 'input' | 'suggestion';
    selected?: boolean;
    selectable?: boolean;
    deletable?: boolean;
    elevated?: boolean;
    href?: string;
    onSelectedChange?: (selected: boolean) => void;
    onDelete?: () => void;
    children: Snippet;
  };

  let {
    variant = 'assist',
    selected = $bindable(false),
    selectable = false,
    deletable = false,
    elevated = false,
    href,
    onSelectedChange,
    onDelete,
    children
  }: Props = $props();

  let element: Chip | undefined = $state();

  onMount(() => {
    void ensureMdui('chip');
  });

  function handleChange() {
    if (!element) return;
    selected = element.selected;
    onSelectedChange?.(selected);
  }

  $effect(() => {
    if (!element) return;
    element.variant = variant;
    element.selected = selected;
    element.selectable = selectable;
    element.deletable = deletable;
    element.elevated = elevated;
  });
  $effect(() => listenMdui(element, 'change', handleChange));
  $effect(() => listenMdui(element, 'delete', () => onDelete?.()));
</script>

{#if href}
  <a {href} class="vx-chip-link"><mdui-chip bind:this={element} {variant}>{@render children()}</mdui-chip></a>
{:else}
  <mdui-chip bind:this={element} {variant}>{@render children()}</mdui-chip>
{/if}

<style>
  .vx-chip-link { display: inline-flex; color: inherit; }
</style>
