<script lang="ts">
  import { onMount } from 'svelte';
  import type { Snippet } from 'svelte';
  import type { Switch } from 'mdui/components/switch.js';
  import { ensureMdui, listenMdui } from '#lib/mdui/registry.js';

  type Props = {
    checked?: boolean;
    disabled?: boolean;
    required?: boolean;
    name?: string;
    value?: string;
    onCheckedChange?: (checked: boolean) => void;
    children?: Snippet;
  };

  let {
    checked = $bindable(false),
    disabled = false,
    required = false,
    name,
    value,
    onCheckedChange,
    children
  }: Props = $props();

  let element: Switch | undefined = $state();

  onMount(() => {
    void ensureMdui('switch');
  });

  function handleChange() {
    if (!element) return;
    checked = element.checked;
    onCheckedChange?.(checked);
  }

  $effect(() => {
    if (!element) return;
    element.checked = checked;
    element.disabled = disabled;
    element.required = required;
  });
  $effect(() => listenMdui(element, 'change', handleChange));
</script>

<mdui-switch bind:this={element} {name} {value}>
  {#if children}{@render children()}{/if}
</mdui-switch>
