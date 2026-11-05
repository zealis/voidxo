<script lang="ts">
  import { onMount } from 'svelte';
  import type { RadioGroup } from 'mdui/components/radio-group.js';
  import { ensureMduiComponents, listenMdui } from '#lib/mdui/registry.js';
  import type { VxOption } from './types.js';

  type Props = {
    value?: string;
    options: VxOption[];
    name?: string;
    disabled?: boolean;
    required?: boolean;
    onValueChange?: (value: string) => void;
  };

  let {
    value = $bindable(''),
    options,
    name,
    disabled = false,
    required = false,
    onValueChange
  }: Props = $props();

  let element: RadioGroup | undefined = $state();

  onMount(() => {
    void ensureMduiComponents(['radio-group', 'radio']);
  });

  function handleChange() {
    if (!element) return;
    value = element.value;
    onValueChange?.(value);
  }

  $effect(() => {
    if (!element) return;
    if (element.value !== value) element.value = value;
    element.disabled = disabled;
    element.required = required;
  });
  $effect(() => listenMdui(element, 'change', handleChange));
</script>

<mdui-radio-group bind:this={element} value={value} {name}>
  {#each options as option (option.value)}
    <!-- disabled 只在 true 时输出 attribute，避免 Lit 把 "false" 判成 true。 -->
    <mdui-radio value={option.value} disabled={option.disabled === true ? true : undefined}>
      {option.label}
    </mdui-radio>
  {/each}
</mdui-radio-group>
