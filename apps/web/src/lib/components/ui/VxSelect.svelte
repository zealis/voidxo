<script lang="ts">
  import { onMount } from 'svelte';
  import type { Select } from 'mdui/components/select.js';
  import { ensureMduiComponents, listenMdui } from '#lib/mdui/registry.js';
  import type { VxOption } from './types.js';

  // M6：Select 不再优先原生 select，改用 mdui-select + mdui-menu-item。
  type Props = {
    value?: string;
    options: VxOption[];
    label?: string;
    name?: string;
    placeholder?: string;
    helper?: string;
    variant?: 'filled' | 'outlined';
    disabled?: boolean;
    clearable?: boolean;
    onValueChange?: (value: string) => void;
  };

  let {
    value = $bindable(''),
    options,
    label,
    name,
    placeholder,
    helper,
    variant = 'outlined',
    disabled = false,
    clearable = false,
    onValueChange
  }: Props = $props();

  let element: Select | undefined = $state();

  onMount(() => {
    void ensureMduiComponents(['select', 'menu-item']);
  });

  function handleChange() {
    if (!element) return;
    value = typeof element.value === 'string' ? element.value : String(element.value ?? '');
    onValueChange?.(value);
  }

  $effect(() => {
    if (!element) return;
    if (element.value !== value) element.value = value;
    element.disabled = disabled;
    element.clearable = clearable;
  });
  $effect(() => listenMdui(element, 'change', handleChange));
</script>

<mdui-select bind:this={element} value={value} {name} {label} {placeholder} {helper} {variant}>
  {#each options as option (option.value)}
    <mdui-menu-item value={option.value} disabled={option.disabled === true ? true : undefined}>
      {option.label}
    </mdui-menu-item>
  {/each}
</mdui-select>
