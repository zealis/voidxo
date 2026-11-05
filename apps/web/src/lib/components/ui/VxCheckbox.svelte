<script lang="ts">
  import { onMount } from 'svelte';
  import type { Snippet } from 'svelte';
  import type { Checkbox } from 'mdui/components/checkbox.js';
  import { ensureMdui, listenMdui } from '#lib/mdui/registry.js';

  type Props = {
    checked?: boolean;
    indeterminate?: boolean;
    disabled?: boolean;
    required?: boolean;
    name?: string;
    value?: string;
    onCheckedChange?: (checked: boolean) => void;
    children?: Snippet;
  };

  let {
    checked = $bindable(false),
    indeterminate = false,
    disabled = false,
    required = false,
    name,
    value,
    onCheckedChange,
    children
  }: Props = $props();

  let element: Checkbox | undefined = $state();

  onMount(() => {
    void ensureMdui('checkbox');
  });

  // M5：mdui 的 change 是 CustomEvent，在封装层转成 Svelte 的 bind:checked。
  function handleChange() {
    if (!element) return;
    checked = element.checked;
    onCheckedChange?.(checked);
  }

  // 布尔值一律设 property：Lit 会把 attribute="false" 判成 true。
  $effect(() => {
    if (!element) return;
    element.checked = checked;
    element.indeterminate = indeterminate;
    element.disabled = disabled;
    element.required = required;
  });
  $effect(() => listenMdui(element, 'change', handleChange));
</script>

<mdui-checkbox bind:this={element} {name} {value}>
  {#if children}{@render children()}{/if}
</mdui-checkbox>
