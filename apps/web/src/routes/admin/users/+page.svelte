<script lang="ts">
  import { invalidateAll } from '$app/navigation';
  import AdminPager from '#lib/components/admin/AdminPager.svelte';
  import Button from '#lib/components/ui/Button.svelte';
  import { AdminRequestError, updateUser, type AdminUser } from '#lib/api/admin.js';
  import type { PageProps } from './$types';

  let { data }: PageProps = $props();

  let pendingId = $state(0);
  let error = $state('');

  const roleLabels: Record<string, string> = {
    admin: '管理员',
    moderator: '版主',
    user: '普通用户'
  };

  function primaryRole(user: AdminUser): string {
    if (user.roles.includes('admin')) return 'admin';
    if (user.roles.includes('moderator')) return 'moderator';
    return 'user';
  }

  async function run(id: number, action: () => Promise<unknown>) {
    if (pendingId) return;
    pendingId = id;
    error = '';
    try {
      await action();
      await invalidateAll();
    } catch (cause) {
      error = cause instanceof AdminRequestError ? cause.message : '操作失败，请稍后重试';
    } finally {
      pendingId = 0;
    }
  }

  function changeRole(user: AdminUser, event: Event) {
    const select = event.currentTarget as HTMLSelectElement;
    const role = select.value;
    if (role === primaryRole(user)) return;
    run(user.id, async () => {
      try {
        await updateUser(user.id, { role });
      } catch (cause) {
        select.value = primaryRole(user);
        throw cause;
      }
    });
  }

  function toggleStatus(user: AdminUser) {
    const disable = user.status === 1 && !user.disabled_at;
    if (disable && !confirm(`确定停用 ${user.username}？停用后该用户的会话会立即失效。`)) return;
    run(user.id, () => updateUser(user.id, { status: disable ? 0 : 1 }));
  }
</script>

<svelte:head>
  <title>用户 · 管理后台 · VOIDXO</title>
</svelte:head>

<section class="section">
  <div class="head">
    <h1>用户</h1>
    <form class="filters" method="get">
      <input type="search" name="q" value={data.query} placeholder="用户名或邮箱" aria-label="搜索用户" />
      <select name="role" aria-label="角色筛选">
        <option value="" selected={data.role === ''}>全部角色</option>
        <option value="admin" selected={data.role === 'admin'}>管理员</option>
        <option value="moderator" selected={data.role === 'moderator'}>版主</option>
        <option value="user" selected={data.role === 'user'}>普通用户</option>
      </select>
      <select name="status" aria-label="状态筛选">
        <option value="" selected={data.status === ''}>全部状态</option>
        <option value="active" selected={data.status === 'active'}>正常</option>
        <option value="disabled" selected={data.status === 'disabled'}>已停用</option>
      </select>
      <Button type="submit" variant="outlined">筛选</Button>
    </form>
  </div>

  {#if error}<p class="error" role="alert">{error}</p>{/if}
  {#if data.loadError}<p class="error" role="alert">用户列表加载失败。</p>{/if}

  <div class="table-wrap">
    <table>
      <thead>
        <tr>
          <th>用户</th>
          <th>角色</th>
          <th>内容</th>
          <th>状态</th>
          <th>注册时间</th>
          <th>操作</th>
        </tr>
      </thead>
      <tbody>
        {#each data.users as user (user.id)}
          <tr class:disabled={user.status !== 1 || !!user.disabled_at}>
            <td>
              <strong>{user.username}</strong>
              <div class="muted">{user.email}</div>
            </td>
            <td>
              <select
                value={primaryRole(user)}
                disabled={pendingId === user.id}
                onchange={(event) => changeRole(user, event)}
                aria-label={`${user.username}的角色`}
              >
                {#each Object.entries(roleLabels) as [value, label] (value)}
                  <option {value}>{label}</option>
                {/each}
              </select>
            </td>
            <td class="muted">
              问题 {user.question_count} · 回答 {user.answer_count} · 文章 {user.article_count} · 评论 {user.comment_count}
            </td>
            <td>
              {#if user.status === 1 && !user.disabled_at}
                <span class="badge">正常</span>
              {:else}
                <span class="badge badge-off">已停用</span>
              {/if}
            </td>
            <td class="muted">{user.created_at.slice(0, 10)}</td>
            <td>
              <Button
                variant="text"
                disabled={pendingId === user.id}
                onclick={() => toggleStatus(user)}
              >
                {user.status === 1 && !user.disabled_at ? '停用' : '启用'}
              </Button>
            </td>
          </tr>
        {:else}
          <tr><td colspan="6" class="muted">没有匹配的用户。</td></tr>
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
  .filters { display: flex; gap: var(--vx-space-2); align-items: center; flex-wrap: wrap; }
  .table-wrap { overflow-x: auto; background: var(--vx-color-surface); border: 1px solid var(--vx-color-border); border-radius: var(--vx-radius-card); }
  table { width: 100%; border-collapse: collapse; font-size: 13px; }
  th, td { padding: 10px 12px; text-align: left; vertical-align: top; }
  th { color: var(--vx-color-text-secondary); font-weight: 500; font-size: 12px; border-bottom: 1px solid var(--vx-color-border); }
  tbody tr + tr td { border-top: 1px solid var(--vx-color-border); }
  tr.disabled td { opacity: 0.55; }
  .muted { color: var(--vx-color-text-secondary); font-size: 12px; }
  .error { color: var(--vx-color-error); font-size: 13px; }
  .badge { font-size: 12px; padding: 1px 8px; border-radius: 999px; border: 1px solid var(--vx-color-border); }
  .badge-off { color: var(--vx-color-error); border-color: var(--vx-color-error); }
  input, select { height: 32px; padding: 0 8px; border: 1px solid var(--vx-color-border); border-radius: var(--vx-radius-control); background: var(--vx-color-surface); color: inherit; }
</style>
