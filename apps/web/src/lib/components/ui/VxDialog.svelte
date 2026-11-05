<script lang="ts">
  import { onMount } from 'svelte';
  import type { Snippet } from 'svelte';
  import type { Dialog } from 'mdui/components/dialog.js';
  import { ensureMdui, listenMdui } from '#lib/mdui/registry.js';

  // M6：不再基于原生 <dialog> 自研，改用 mdui-dialog。
  type Props = {
    open?: boolean;
    headline?: string;
    description?: string;
    fullscreen?: boolean;
    closeOnEsc?: boolean;
    closeOnOverlayClick?: boolean;
    children?: Snippet;
    actions?: Snippet;
  };

  let {
    open = $bindable(false),
    headline,
    description,
    fullscreen = false,
    closeOnEsc = true,
    closeOnOverlayClick = false,
    children,
    actions
  }: Props = $props();

  let element: Dialog | undefined = $state();

  onMount(() => {
    void ensureMdui('dialog');
  });

  function handleClose() {
    open = false;
  }

  $effect(() => {
    if (!element) return;
    element.open = open;
    element.fullscreen = fullscreen;
    element.closeOnEsc = closeOnEsc;
    element.closeOnOverlayClick = closeOnOverlayClick;
  });
  $effect(() => listenMdui(element, 'close', handleClose));
</script>

<mdui-dialog bind:this={element} {headline} {description}>
  {#if children}{@render children()}{/if}
  {#if actions}
    <div slot="action" class="vx-dialog-actions">{@render actions()}</div>
  {/if}
</mdui-dialog>

<style>
  .vx-dialog-actions { display: flex; justify-content: flex-end; gap: var(--vx-space-2); }
</style>
