<script lang="ts">
  import { onMount } from 'svelte';
  import type { Snippet } from 'svelte';
  import type { Badge } from 'mdui/components/badge.js';
  import { ensureMdui } from '#lib/mdui/registry.js';

  // 未读数 / 状态徽标场景，替换各页面手写的 .badge 样式类。
  type Props = {
    variant?: 'small' | 'large';
    children: Snippet;
  };

  let { variant = 'large', children }: Props = $props();
  let element: Badge | undefined = $state();

  onMount(() => {
    void ensureMdui('badge');
  });

  $effect(() => {
    if (!element) return;
    element.variant = variant;
  });
</script>

<mdui-badge bind:this={element} {variant}>{@render children()}</mdui-badge>
