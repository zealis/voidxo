<script lang="ts">
  import { invalidateAll } from '$app/navigation';
  import AdminPager from './AdminPager.svelte';
  import VxButton from '#lib/components/ui/VxButton.svelte';
  import { AdminRequestError, deleteContent, type AdminContentItem, type ContentKind } from '#lib/api/admin.js';

  let {
    kind,
    label,
    items,
    page,
    hasMore,
    query,
    deleted,
    loadError
  }: {
    kind: ContentKind;
    label: string;
    items: AdminContentItem[];
    page: number;
    hasMore: boolean;
    query: string;
    deleted: string;
    loadError: boolean;
  } = $props();

  let pendingId = $state(0);
  let error = $state('');

  const deletedLabels: Record<string, string> = {
    live: '仅正常内容',
    deleted: '仅已删除',
    all: '全部'
  };

  function target(item: AdminContentItem): string {
    switch (kind) {
      case 'question':
        return `/questions/${item.id}`;
      case 'article':
        return `/articles/${item.id}`;
      case 'answer':
        return item.parent_id ? `/questions/${item.parent_id}` : '';
      default:
        if (item.parent_type === 'question') return `/questions/${item.parent_id}`;
        if (item.parent_type === 'article') return `/articles/${item.parent_id}`;
        return '';
    }
  }

  async function remove(item: AdminContentItem) {
    if (pendingId) return;
    if (!confirm(`确定删除这条${label}？删除后作者会收到通知。`)) return;
    pendingId = item.id;
    error = '';
    try {
      await deleteContent(kind, item.id);
      await invalidateAll();
    } catch (cause) {
      error = cause instanceof AdminRequestError ? cause.message : '删除失败，请稍后重试';
    } finally {
      pendingId = 0;
    }
  }
</script>

<section class="section">
  <div class="head">
    <h1>{label}</h1>
    <form class="filters" method="get">
      <input type="search" name="q" value={query} placeholder="搜索标题或正文" aria-label={`搜索${label}`} />
      <select name="deleted" aria-label="删除状态">
        {#each Object.entries(deletedLabels) as [value, text] (value)}
          <option {value} selected={deleted === value}>{text}</option>
        {/each}
      </select>
      <VxButton type="submit" variant="outlined">筛选</VxButton>
    </form>
  </div>

  {#if error}<p class="error" role="alert">{error}</p>{/if}
  {#if loadError}<p class="error" role="alert">列表加载失败。</p>{/if}

  <div class="table-wrap">
    <table>
      <thead>
        <tr>
          <th>内容</th>
          <th>作者</th>
          <th>互动</th>
          <th>状态</th>
          <th>时间</th>
          <th>操作</th>
        </tr>
      </thead>
      <tbody>
        {#each items as item (item.id)}
          <tr class:removed={item.deleted}>
            <td>
              {#if target(item)}
                <a class="title" href={target(item)}>{item.title || item.excerpt.slice(0, 40)}</a>
              {:else}
                <span class="title">{item.title || item.excerpt.slice(0, 40)}</span>
              {/if}
              {#if item.context}<div class="muted">{item.context}</div>{/if}
              {#if item.title && item.excerpt}<div class="muted excerpt">{item.excerpt}</div>{/if}
            </td>
            <td class="muted">{item.username}</td>
            <td class="muted">
              {#if kind !== 'comment'}评论 {item.comment_count} · {/if}票数 {item.vote_count}
            </td>
            <td>
              {#if item.deleted}
                <span class="badge badge-off">已删除</span>
              {:else}
                <span class="badge">正常</span>
              {/if}
            </td>
            <td class="muted">{item.created_at.slice(0, 10)}</td>
            <td>
              {#if !item.deleted}
                <VxButton variant="text" disabled={pendingId === item.id} onclick={() => remove(item)}>删除</VxButton>
              {:else}
                <span class="muted">—</span>
              {/if}
            </td>
          </tr>
        {:else}
          <tr><td colspan="6" class="muted">没有匹配的内容。</td></tr>
        {/each}
      </tbody>
    </table>
  </div>

  <AdminPager current={page} {hasMore} />
</section>

<style>
  .section { display: flex; flex-direction: column; gap: var(--vx-space-3); }
  .head { display: flex; align-items: center; justify-content: space-between; gap: var(--vx-space-3); flex-wrap: wrap; }
  h1 { font-size: 18px; font-weight: 600; margin: 0; }
  .filters { display: flex; gap: var(--vx-space-2); align-items: center; flex-wrap: wrap; }
  .table-wrap { overflow-x: auto; background: var(--vx-color-surface); border: 1px solid var(--vx-color-border); border-radius: var(--vx-radius-card); }
  table { width: 100%; border-collapse: collapse; font-size: 13px; }
  th, td { padding: 10px 12px; text-align: left; vertical-align: top; }
  th { color: var(--vx-color-text-secondary); font-weight: 500; font-size: 12px; border-bottom: 1px solid var(--vx-color-border); }
  tbody tr + tr td { border-top: 1px solid var(--vx-color-border); }
  tr.removed td { opacity: 0.55; }
  .title { font-weight: 500; color: inherit; text-decoration: none; }
  a.title:hover { text-decoration: underline; }
  .muted { color: var(--vx-color-text-secondary); font-size: 12px; }
  .excerpt { max-width: 420px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
  .error { color: var(--vx-color-error); font-size: 13px; }
  .badge { font-size: 12px; padding: 1px 8px; border-radius: 999px; border: 1px solid var(--vx-color-border); }
  .badge-off { color: var(--vx-color-error); border-color: var(--vx-color-error); }
  input, select { height: 32px; padding: 0 8px; border: 1px solid var(--vx-color-border); border-radius: var(--vx-radius-control); background: var(--vx-color-surface); color: inherit; }
</style>
