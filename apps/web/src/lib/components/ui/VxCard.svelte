<script lang="ts">
  import { onMount } from 'svelte';
  import type { Snippet } from 'svelte';
  import type { Card } from 'mdui/components/card.js';
  import { ensureMdui, listenMdui } from '#lib/mdui/registry.js';

  // 为 SectionCard / TopicCard 等业务卡片提供统一的 mdui 容器；
  // 标题、头部与信息密度仍由业务组件控制，不在这里改变页面结构。
  type Props = {
    variant?: 'elevated' | 'filled' | 'outlined';
    clickable?: boolean;
    disabled?: boolean;
    onclick?: (event: MouseEvent) => void;
    children: Snippet;
  };

  let {
    variant = 'outlined',
    clickable = false,
    disabled = false,
    onclick,
    children
  }: Props = $props();

  let element: Card | undefined = $state();

  onMount(() => {
    void ensureMdui('card');
  });

  $effect(() => {
    if (!element) return;
    element.variant = variant;
    element.clickable = clickable;
    element.disabled = disabled;
  });
  $effect(() => listenMdui(element, 'click', (event) => onclick?.(event as MouseEvent)));
</script>

<mdui-card bind:this={element} {variant}>{@render children()}</mdui-card>
