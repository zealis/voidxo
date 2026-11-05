<script lang="ts">
  import type { Snippet } from 'svelte';
  import Icon from './Icon.svelte';

  type Props = {
    title: string;
    href: string;
    children: Snippet;
    /** 列表页使用无边框卡片，与旧站 #page-questions .item-list 一致。 */
    flush?: boolean;
  };

  let { title, href, children, flush = false }: Props = $props();
</script>

<section class="section-card" class:section-card--flush={flush}>
  <a class="section-head" href={href}>
    <span class="section-title">{title}</span>
    <Icon name="arrow-right" size={18} />
  </a>
  <div class="section-body">
    {@render children()}
  </div>
</section>

<style>
  .section-card {
    background: var(--vx-color-surface);
    border: 1px solid var(--vx-color-border);
    border-radius: var(--vx-radius-card);
    overflow: hidden;
  }
  .section-card--flush { border-right: 0; border-left: 0; border-radius: 0; }
  .section-head {
    display: flex;
    align-items: center;
    justify-content: space-between;
    height: 52px;
    padding: 0 var(--vx-space-6);
    color: var(--vx-color-text-secondary);
    font-size: var(--vx-font-size-subtitle);
    font-weight: 500;
  }
  .section-head:hover { background: var(--vx-color-surface-hover); color: var(--vx-color-text-primary); }
  .section-head :global(svg) { color: var(--vx-color-primary); }
  .section-title { overflow: hidden; white-space: nowrap; text-overflow: ellipsis; }
  .section-body { display: flex; flex-direction: column; }
  .section-body :global(> * + *) { border-top: 1px solid var(--vx-color-divider); }
  @media (max-width: 599.9px) {
    .section-head { padding: 0 var(--vx-space-4); }
  }
</style>
