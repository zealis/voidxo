<script lang="ts">
  import Avatar from './Avatar.svelte';

  type Props = {
    href: string;
    title: string;
    username?: string;
    avatarKey?: string;
    snippet?: string;
    /** 元信息用 ` · ` 连接，例如「3 个回答 · 2026-11-05」。 */
    meta?: string;
  };

  let { href, title, username = '', avatarKey = '', snippet = '', meta = '' }: Props = $props();
</script>

<a class="list-row" {href}>
  <span class="row-avatar">
    <Avatar name={username || title} {avatarKey} size={20} />
  </span>
  <span class="row-title vx-clamp-1">{title}</span>
  <span class="row-content">
    {#if snippet}<span class="row-snippet vx-clamp-1">{snippet}</span>{/if}
    {#if meta}<span class="row-meta">{meta}</span>{/if}
  </span>
</a>

<style>
  /* 旧站 .mc-list-item：左侧 20px 头像，标题 16，摘要 14，元信息 13。 */
  .list-row {
    position: relative;
    display: block;
    padding: var(--vx-space-4) var(--vx-space-6) var(--vx-space-4) 64px;
    transition: background-color var(--vx-motion-standard) var(--vx-motion-easing);
  }
  .list-row:hover { background: var(--vx-color-surface-hover); }
  .list-row:active { background: var(--vx-color-surface-active); }
  .row-avatar { position: absolute; top: 18px; left: 26px; display: flex; }
  .row-title { display: block; font-size: var(--vx-font-size-subtitle); line-height: 20px; }
  .row-content { display: flex; align-items: center; gap: var(--vx-space-2); margin-top: var(--vx-space-2); }
  .row-snippet { flex: 1; color: var(--vx-color-text-secondary); font-size: var(--vx-font-size-body); }
  .row-meta { flex: none; color: var(--vx-color-text-tertiary); font-size: var(--vx-font-size-small); white-space: nowrap; }
  @media (max-width: 599.9px) {
    .list-row { padding: var(--vx-space-3) var(--vx-space-4) var(--vx-space-3) 48px; }
    .row-avatar { top: 15px; left: 16px; }
    .row-content { flex-direction: column; align-items: flex-start; gap: var(--vx-space-1); }
    .row-meta { margin-top: 2px; }
  }
</style>
