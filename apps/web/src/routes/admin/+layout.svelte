<script lang="ts">
  import { page } from '$app/state';
  import type { LayoutProps } from './$types';

  let { data, children }: LayoutProps = $props();

  type Item = { href: string; label: string; permission?: string };

  // 10 个后台分区，与旧系统保持一致；无对应权限的分区不显示。
  const sections: Item[] = [
    { href: '/admin', label: '仪表盘' },
    { href: '/admin/users', label: '用户', permission: 'user.manage' },
    { href: '/admin/questions', label: '问题', permission: 'content.moderate' },
    { href: '/admin/answers', label: '回答', permission: 'content.moderate' },
    { href: '/admin/articles', label: '文章', permission: 'content.moderate' },
    { href: '/admin/comments', label: '评论', permission: 'content.moderate' },
    { href: '/admin/topics', label: '话题', permission: 'content.manage' },
    { href: '/admin/reports', label: '举报', permission: 'report.manage' },
    { href: '/admin/options', label: '配置', permission: 'option.manage' },
    { href: '/admin/images', label: '图片', permission: 'file.manage' }
  ];

  const visible = $derived(
    sections.filter((item) => !item.permission || data.user.permissions.includes(item.permission))
  );

  const current = $derived(page.url.pathname);

  function isActive(href: string): boolean {
    if (href === '/admin') return current === '/admin';
    return current === href || current.startsWith(`${href}/`);
  }
</script>

<svelte:head>
  <title>管理后台 · VOIDXO</title>
</svelte:head>

<div class="admin-shell">
  <header class="admin-bar">
    <a class="brand" href="/">VOIDXO</a>
    <span class="divider" aria-hidden="true">/</span>
    <span class="title">管理后台</span>
    <span class="spacer"></span>
    <a class="back" href="/">返回前台</a>
    <span class="user">{data.user.username}</span>
  </header>

  <div class="admin-body">
    <nav class="admin-nav" aria-label="后台分区">
      {#each visible as item (item.href)}
        <a href={item.href} class:active={isActive(item.href)} aria-current={isActive(item.href) ? 'page' : undefined}>
          {item.label}
        </a>
      {/each}
    </nav>
    <main class="admin-main">
      {@render children()}
    </main>
  </div>
</div>

<style>
  .admin-shell { min-height: 100vh; background: var(--vx-color-bg); }
  .admin-bar {
    display: flex;
    align-items: center;
    gap: var(--vx-space-2);
    height: 56px;
    padding: 0 var(--vx-space-4);
    background: var(--vx-color-surface);
    box-shadow: inset 0 -1px 0 var(--vx-color-border);
  }
  .brand { font-weight: 600; letter-spacing: 0.04em; color: var(--vx-color-text-primary); text-decoration: none; }
  .divider { color: var(--vx-color-border); }
  .title { color: var(--vx-color-text-secondary); font-size: 13px; }
  .spacer { flex: 1; }
  .back { color: var(--vx-color-text-secondary); font-size: 13px; text-decoration: none; }
  .back:hover { color: var(--vx-color-text-primary); }
  .user { color: var(--vx-color-text-secondary); font-size: 13px; }
  .admin-body { display: grid; grid-template-columns: 176px minmax(0, 1fr); gap: var(--vx-space-4); padding: var(--vx-space-4); }
  .admin-nav { display: flex; flex-direction: column; gap: 2px; align-self: start; position: sticky; top: var(--vx-space-4); }
  .admin-nav a {
    padding: 8px 12px;
    border-radius: var(--vx-radius-control);
    color: var(--vx-color-text-secondary);
    font-size: 13px;
    text-decoration: none;
  }
  .admin-nav a:hover { background: var(--vx-color-surface); color: var(--vx-color-text-primary); }
  .admin-nav a.active { background: var(--vx-color-surface); color: var(--vx-color-text-primary); font-weight: 600; }
  .admin-main { min-width: 0; }
  @media (max-width: 720px) {
    .admin-body { grid-template-columns: minmax(0, 1fr); }
    .admin-nav { flex-direction: row; flex-wrap: wrap; position: static; }
  }
</style>
