<script lang="ts">
  import { onMount } from 'svelte';
  import type { Tabs } from 'mdui/components/tabs.js';
  import { ensureMduiComponents, listenMdui } from '#lib/mdui/registry.js';
  import type { VxTabItem } from './types.js';

  type Props = {
    value?: string;
    items: VxTabItem[];
    variant?: 'primary' | 'secondary';
    placement?: string;
    onValueChange?: (value: string) => void;
  };

  let {
    value = $bindable(''),
    items,
    variant = 'secondary',
    placement = 'top-start',
    onValueChange
  }: Props = $props();

  let element: Tabs | undefined = $state();

  onMount(() => {
    void ensureMduiComponents(['tabs', 'tab', 'tab-panel']);
  });

  function handleChange() {
    if (!element) return;
    value = element.value ?? '';
    onValueChange?.(value);
  }

  $effect(() => {
    if (!element) return;
    if (element.value !== value) element.value = value;
  });
  $effect(() => listenMdui(element, 'change', handleChange));
</script>

<mdui-tabs bind:this={element} value={value} {variant} {placement}>
  {#each items as item (item.value)}
    <mdui-tab value={item.value}>{item.label}</mdui-tab>
  {/each}
  {#each items as item (item.value)}
    <mdui-tab-panel value={item.value} slot="panel">{item.content}</mdui-tab-panel>
  {/each}
</mdui-tabs>
