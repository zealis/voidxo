<script lang="ts">
  import { page } from '$app/state';

  let { current, hasMore }: { current: number; hasMore: boolean } = $props();

  // 只读的 searchParams 不能直接改，先复制成可变副本。
  const params = $derived(new URLSearchParams(page.url.searchParams.toString()));

  function href(target: number): string {
    const next = new URLSearchParams(params);
    next.set('page', String(target));
    return `?${next.toString()}`;
  }
</script>

{#if current > 1 || hasMore}
  <nav class="pager" aria-label="分页">
    {#if current > 1}
      <a href={href(current - 1)} rel="prev">上一页</a>
    {:else}
      <span class="disabled">上一页</span>
    {/if}
    <span class="state">第 {current} 页</span>
    {#if hasMore}
      <a href={href(current + 1)} rel="next">下一页</a>
    {:else}
      <span class="disabled">下一页</span>
    {/if}
  </nav>
{/if}

<style>
  .pager { display: flex; align-items: center; gap: var(--vx-space-3); font-size: 13px; }
  .pager a { color: var(--vx-color-text-primary); text-decoration: none; }
  .pager a:hover { text-decoration: underline; }
  .disabled { color: var(--vx-color-text-secondary); }
  .state { color: var(--vx-color-text-secondary); font-size: 12px; }
</style>
