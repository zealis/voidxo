<script lang="ts">
  import { onMount } from 'svelte';
  import type { TextField } from 'mdui/components/text-field.js';
  import { ensureMdui } from '#lib/mdui/registry.js';

  // M5：bind:value 不能直接绑到 mdui 自定义元素。
  // 这里对业务层提供标准的 Svelte bind:value，内部用 input/change 事件读回
  // 元素的 value property，并在外部值变化时反向写回。
  type FieldType =
    | 'text' | 'number' | 'password' | 'url' | 'email' | 'search' | 'tel'
    | 'hidden' | 'date' | 'datetime-local' | 'month' | 'time' | 'week';

  type Props = {
    value?: string;
    label?: string;
    name?: string;
    type?: FieldType;
    variant?: 'filled' | 'outlined';
    placeholder?: string;
    autocomplete?: string;
    maxlength?: number;
    minlength?: number;
    inputmode?: 'none' | 'text' | 'decimal' | 'numeric' | 'tel' | 'search' | 'email' | 'url';
    pattern?: string;
    // 多行文本框的固定行数；设置后 mdui-text-field 渲染为 textarea。
    rows?: number;
    helper?: string;
    error?: string;
    disabled?: boolean;
    required?: boolean;
    readonly?: boolean;
    clearable?: boolean;
    onValueChange?: (value: string) => void;
  };

  let {
    value = $bindable(''),
    label,
    name,
    type = 'text',
    variant = 'outlined',
    placeholder,
    autocomplete,
    maxlength,
    minlength,
    inputmode,
    pattern,
    rows,
    helper,
    error,
    disabled = false,
    required = false,
    readonly = false,
    clearable = false,
    onValueChange
  }: Props = $props();

  // mdui 在 HTMLElementTagNameMap 里注册了 TextField；import type 会在编译期擦除，
  // 不会让组件 JS 进入服务端模块图。
  let element: TextField | undefined = $state();

  onMount(() => {
    void ensureMdui('text-field');
  });

  function syncFromEvent(event: Event) {
    const target = event.currentTarget as HTMLElement & { value: string };
    value = target.value;
    onValueChange?.(value);
  }

  // 布尔与表单值以 property 同步，避免 Lit 把 attribute="false" 判成 true。
  $effect(() => {
    if (!element) return;
    if (element.value !== value) element.value = value;
    element.disabled = disabled;
    element.required = required;
    element.readonly = readonly;
    element.clearable = clearable;
    if (rows != null) element.rows = rows;
  });
</script>

<mdui-text-field
  bind:this={element}
  value={value}
  {label}
  {name}
  {type}
  {variant}
  {placeholder}
  {autocomplete}
  {maxlength}
  {minlength}
  {inputmode}
  {pattern}
  {helper}
  {error}
  oninput={syncFromEvent}
  onchange={syncFromEvent}
></mdui-text-field>
