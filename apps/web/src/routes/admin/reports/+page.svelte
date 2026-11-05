<script lang="ts">
  import { invalidateAll } from '$app/navigation';
  import AdminPager from '#lib/components/admin/AdminPager.svelte';
  import Button from '#lib/components/ui/Button.svelte';
  import { AdminRequestError, resolveReport, type AdminReport } from '#lib/api/admin.js';
  import type { PageProps } from './$types';

  let { data }: PageProps = $props();

  let pendingId = $state(0);
  let error = $state('');

  const typeLabels: Record<string, string> = {
    question: '问题',
    answer: '回答',
    article: '文章',
    comment: '评论',
    user: '用户'
  };

  const statusLabels: Record<string, string> = {
    pending: '待处理',
    resolved: '已处理',
    rejected: '已驳回'
  };

  function target(report: AdminReport): string {
    switch (report.reportable_type) {
      case 'question':
        return `/questions/${report.reportable_id}`;
      case 'article':
        return `/articles/${report.reportable_id}`;
      default:
        return '';
    }
  }

  async function handle(report: AdminReport, status: 'resolved' | 'rejected') {
    if (pendingId) return;
    pendingId = report.id;
    error = '';
    try {
      await resolveReport(report.id, status);
      await invalidateAll();
    } catch (cause) {
      error = cause instanceof AdminRequestError ? cause.message : '处理失败，请稍后重试';
    } finally {
      pendingId = 0;
    }
  }
</script>

<svelte:head>
  <title>举报 · 管理后台 · VOIDXO</title>
</svelte:head>

<section class="section">
  <div class="head">
    <h1>举报</h1>
    <form class="filters" method="get">
      <select name="status" aria-label="举报状态">
        {#each Object.entries(statusLabels) as [value, label] (value)}
          <option {value} selected={data.status === value}>{label}</option>
        {/each}
        <option value="" selected={data.status === ''}>全部</option>
      </select>
      <Button type="submit" variant="outlined">筛选</Button>
    </form>
  </div>

  {#if error}<p class="error" role="alert">{error}</p>{/if}
  {#if data.loadError}<p class="error" role="alert">举报列表加载失败。</p>{/if}

  <div class="table-wrap">
    <table>
      <thead>
        <tr>
          <th>目标</th>
          <th>举报人</th>
          <th>理由</th>
          <th>状态</th>
          <th>时间</th>
          <th>操作</th>
        </tr>
      </thead>
      <tbody>
        {#each data.reports as report (report.id)}
          <tr>
            <td>
              {#if target(report)}
                <a href={target(report)}>{typeLabels[report.reportable_type] ?? report.reportable_type} #{report.reportable_id}</a>
              {:else}
                <span>{typeLabels[report.reportable_type] ?? report.reportable_type} #{report.reportable_id}</span>
              {/if}
            </td>
            <td class="muted">{report.username}</td>
            <td>{report.reason}</td>
            <td><span class="badge">{statusLabels[report.status] ?? report.status}</span></td>
            <td class="muted">{report.created_at.slice(0, 10)}</td>
            <td class="actions">
              {#if report.status === 'pending'}
                <Button variant="text" disabled={pendingId === report.id} onclick={() => handle(report, 'resolved')}>处理</Button>
                <Button variant="text" disabled={pendingId === report.id} onclick={() => handle(report, 'rejected')}>驳回</Button>
              {:else}
                <span class="muted">—</span>
              {/if}
            </td>
          </tr>
        {:else}
          <tr><td colspan="6" class="muted">没有符合条件的举报。</td></tr>
        {/each}
      </tbody>
    </table>
  </div>

  <AdminPager current={data.page} hasMore={data.hasMore} />
</section>

<style>
  .section { display: flex; flex-direction: column; gap: var(--vx-space-3); }
  .head { display: flex; align-items: center; justify-content: space-between; gap: var(--vx-space-3); flex-wrap: wrap; }
  h1 { font-size: 18px; font-weight: 600; margin: 0; }
  .filters { display: flex; gap: var(--vx-space-2); align-items: center; }
  .table-wrap { overflow-x: auto; background: var(--vx-color-surface); border: 1px solid var(--vx-color-border); border-radius: var(--vx-radius-card); }
  table { width: 100%; border-collapse: collapse; font-size: 13px; }
  th, td { padding: 10px 12px; text-align: left; vertical-align: top; }
  th { color: var(--vx-color-text-secondary); font-weight: 500; font-size: 12px; border-bottom: 1px solid var(--vx-color-border); }
  tbody tr + tr td { border-top: 1px solid var(--vx-color-border); }
  .muted { color: var(--vx-color-text-secondary); font-size: 12px; }
  .error { color: var(--vx-color-error); font-size: 13px; }
  .badge { font-size: 12px; padding: 1px 8px; border-radius: 999px; border: 1px solid var(--vx-color-border); }
  .actions { display: flex; gap: var(--vx-space-1); }
  select { height: 32px; padding: 0 8px; border: 1px solid var(--vx-color-border); border-radius: var(--vx-radius-control); background: var(--vx-color-surface); color: inherit; }
  a { color: inherit; }
</style>
