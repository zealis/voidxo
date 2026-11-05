<script lang="ts">
  import { onMount } from 'svelte';
  import type { Snippet } from 'svelte';
  import type { Tooltip } from 'mdui/components/tooltip.js';
  import { ensureMdui } from '#lib/mdui/registry.js';

  type Props = {
    content: string;
    headline?: string;
    variant?: 'plain' | 'rich';
    placement?: string;
    trigger?: 'click' | 'hover' | 'focus';
    disabled?: boolean;
    children: Snippet;
  };

  let {
    content,
    headline,
    variant = 'plain',
    placement = 'auto',
    trigger = 'hover',
    disabled = false,
    children
  }: Props = $props();

  let element: Tooltip | undefined = $state();

  onMount(() => {
    void ensureMdui('tooltip');
  });

  $effect(() => {
    if (!element) return;
    element.disabled = disabled;
  });
</script>

<!-- default slot 的第一个元素即为 tooltip 的触发目标。 -->
<mdui-tooltip bind:this={element} {content} {headline} {variant} {placement} {trigger}>
  {@render children()}
</mdui-tooltip>
