<script lang="ts">
  import Icon from './Icon.svelte';

  type Props = {
    href: string;
    label: string;
  };

  let { href, label }: Props = $props();
</script>

<a class="fab" {href}>
  <Icon name="plus" size={24} />
  <span>{label}</span>
</a>

<style>
  /* 旧站 .mdui-fab-extended：48 高、24 圆角、右下 16。 */
  .fab {
    position: fixed;
    right: var(--vx-space-4);
    bottom: var(--vx-space-4);
    z-index: 20;
    display: inline-flex;
    align-items: center;
    gap: var(--vx-space-3);
    height: 48px;
    padding: 0 var(--vx-space-5) 0 var(--vx-space-3);
    color: #fff;
    font-size: var(--vx-font-size-body);
    font-weight: 500;
    background: var(--vx-color-primary);
    border-radius: var(--vx-radius-pill);
    box-shadow: var(--vx-shadow-overlay);
    transition: box-shadow var(--vx-motion-standard) var(--vx-motion-easing),
      transform var(--vx-motion-standard) var(--vx-motion-easing);
  }
  .fab:hover { box-shadow: 0 5px 5px -3px rgba(0, 0, 0, 0.2), 0 8px 10px 1px rgba(0, 0, 0, 0.14), 0 3px 14px 2px rgba(0, 0, 0, 0.12); }
  .fab:active { transform: scale(0.98); }
</style>
