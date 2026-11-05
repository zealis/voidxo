<script lang="ts">
  import Icon from './Icon.svelte';

  type Props = {
    page: number;
    hasMore: boolean;
    basePath: string;
    /** 保留当前排序等查询参数。 */
    keep?: Record<string, string>;
  };

  let { page, hasMore, basePath, keep = {} }: Props = $props();

  function hrefFor(target: number) {
    const query = new URLSearchParams(keep);
    if (target > 1) query.set('page', String(target));
    const suffix = query.toString();
    return suffix ? `${basePath}?${suffix}` : basePath;
  }
</script>

{#if page > 1 || hasMore}
  <nav class="pager" aria-label="分页">
    {#if page > 1}
      <a class="pager-link" href={hrefFor(page - 1)}><Icon name="arrow-left" size={16} />上一页</a>
    {:else}
      <span class="pager-link disabled"><Icon name="arrow-left" size={16} />上一页</span>
    {/if}
    <span class="pager-info">第 {page} 页</span>
    {#if hasMore}
      <a class="pager-link" href={hrefFor(page + 1)}>下一页<Icon name="arrow-right" size={16} /></a>
    {:else}
      <span class="pager-link disabled">下一页<Icon name="arrow-right" size={16} /></span>
    {/if}
  </nav>
{/if}

<style>
  .pager { display: flex; align-items: center; justify-content: center; gap: var(--vx-space-4); padding: var(--vx-space-6) 0; }
  .pager-link { display: inline-flex; align-items: center; gap: var(--vx-space-1); padding: var(--vx-space-2) var(--vx-space-3); color: var(--vx-color-primary); font-size: var(--vx-font-size-small); border-radius: var(--vx-radius-control); }
  .pager-link:hover { background: var(--vx-color-surface-hover); }
  .pager-link.disabled { color: var(--vx-color-text-tertiary); }
  .pager-info { color: var(--vx-color-text-secondary); font-size: var(--vx-font-size-small); }
</style>
