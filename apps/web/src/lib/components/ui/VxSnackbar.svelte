<script lang="ts">
  import { onMount } from 'svelte';
  import type { Snackbar } from 'mdui/components/snackbar.js';
  import { ensureMdui, listenMdui } from '#lib/mdui/registry.js';

  // M6：原自研 Toast 改名为 Snackbar，底层走 mdui-snackbar。
  type Props = {
    open?: boolean;
    message: string;
    action?: string;
    placement?: 'top' | 'top-start' | 'top-end' | 'bottom' | 'bottom-start' | 'bottom-end';
    autoCloseDelay?: number;
    closeable?: boolean;
    onActionClick?: () => void;
  };

  let {
    open = $bindable(false),
    message,
    action,
    placement = 'bottom',
    autoCloseDelay = 5000,
    closeable = false,
    onActionClick
  }: Props = $props();

  let element: Snackbar | undefined = $state();

  onMount(() => {
    void ensureMdui('snackbar');
  });

  function handleClose() {
    open = false;
  }

  function handleAction() {
    onActionClick?.();
  }

  $effect(() => {
    if (!element) return;
    element.open = open;
    element.placement = placement;
    element.autoCloseDelay = autoCloseDelay;
    element.closeable = closeable;
  });
  $effect(() => listenMdui(element, 'close', handleClose));
  $effect(() => listenMdui(element, 'action-click', handleAction));
</script>

<mdui-snackbar bind:this={element} {action}>{message}</mdui-snackbar>
