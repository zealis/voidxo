<script lang="ts">
  import type { Snippet } from 'svelte';
  import { page } from '$app/state';
  import { goto } from '$app/navigation';
  import Icon from './Icon.svelte';
  import Avatar from './Avatar.svelte';
  import { logout, type AuthUser } from '#lib/api/auth.js';

  type Props = {
    siteName?: string;
    user?: AuthUser | null;
    width?: 'content' | 'list' | 'detail';
    children: Snippet;
  };

  let { siteName = 'VOIDXO', user = null, width = 'content', children }: Props = $props();

  let drawerOpen = $state(false);
  // 桌面端（>=1024px）抽屉常驻展开，汉堡按钮切换收起；CSS 默认按展开渲染。
  let drawerPinned = $state(true);
  let menuOpen = $state(false);
  let loggingOut = $state(false);
  let logoutError = $state('');
  let drawerElement = $state<HTMLDivElement | null>(null);

  const navItems = [
    { href: '/', label: '首页', icon: 'home' as const },
    { href: '/topics', label: '话题', icon: 'tag' as const },
    { href: '/questions', label: '问答', icon: 'forum' as const },
    { href: '/articles', label: '文章', icon: 'article' as const }
  ];
  const peopleItem = { href: '/users', label: '人脉', icon: 'users' as const };

  const sectionTitles: Record<string, string> = {
    '': '首页',
    topics: '话题',
    questions: '问答',
    articles: '文章',
    users: '人脉',
    messages: '私信',
    notifications: '通知',
    search: '搜索',
    admin: '管理',
    user: '个人资料'
  };
  const section = $derived(page.url.pathname.split('/').filter(Boolean)[0] ?? '');
  const sectionTitle = $derived(sectionTitles[section] ?? siteName);

  function isActive(href: string) {
    const path = page.url.pathname;
    return href === '/' ? path === '/' : path === href || path.startsWith(`${href}/`);
  }

  function toggleDrawer() {
    // 桌面端切换常驻抽屉的展开/收起；窄屏切换覆盖式抽屉。
    if (typeof window !== 'undefined' && window.matchMedia('(min-width: 1024px)').matches) {
      drawerPinned = !drawerPinned;
      return;
    }
    drawerOpen = !drawerOpen;
    if (drawerOpen) {
      // 打开后把焦点移到第一个导航项，键盘用户可以继续 Tab。
      queueMicrotask(() => drawerElement?.querySelector<HTMLAnchorElement>('a')?.focus());
    }
  }

  function closeDrawer() {
    drawerOpen = false;
  }

  function handleKeydown(event: KeyboardEvent) {
    if (event.key !== 'Escape') return;
    if (menuOpen) menuOpen = false;
    else if (drawerOpen) drawerOpen = false;
  }

  function handleWindowClick(event: MouseEvent) {
    if (!menuOpen) return;
    const target = event.target as HTMLElement | null;
    if (!target?.closest('.user-menu')) menuOpen = false;
  }

  // 路由变化后收起抽屉与菜单，避免遮挡新页面。
  $effect(() => {
    page.url.pathname;
    drawerOpen = false;
    menuOpen = false;
  });

  // 跟随视口同步抽屉形态：桌面常驻展开，窄屏回到覆盖式。
  // resize 兜底：部分环境（如内嵌浏览器视口模拟）不派发 matchMedia change 事件。
  $effect(() => {
    const mq = window.matchMedia('(min-width: 1024px)');
    const sync = () => {
      drawerPinned = mq.matches;
    };
    sync();
    mq.addEventListener('change', sync);
    window.addEventListener('resize', sync);
    return () => {
      mq.removeEventListener('change', sync);
      window.removeEventListener('resize', sync);
    };
  });

  async function handleLogout() {
    if (loggingOut) return;
    loggingOut = true;
    logoutError = '';
    try {
      await logout();
      await goto('/login');
    } catch {
      logoutError = '退出失败，请稍后重试';
    } finally {
      loggingOut = false;
      menuOpen = false;
    }
  }
</script>

<svelte:window onkeydown={handleKeydown} onclick={handleWindowClick} />

<div class="vx-shell" class:drawer-collapsed={!drawerPinned}>
  <header class="vx-appbar">
    <button
      type="button"
      class="vx-icon-button appbar-toggle"
      aria-label={drawerOpen || drawerPinned ? '收起导航' : '展开导航'}
      aria-expanded={drawerOpen || drawerPinned}
      onclick={toggleDrawer}
    >
      <Icon name="menu" />
    </button>

    <a class="appbar-brand" href="/" aria-label={`${siteName} 首页`}>{siteName}</a>
    <span class="appbar-title" aria-hidden="true">{sectionTitle}</span>

    <div class="appbar-spacer"></div>

    <!-- 收起态仅剩搜索图标；label 兼作点击触发区，点击或键盘聚焦即展开。 -->
    <form class="appbar-search" action="/search" method="GET" role="search">
      <label class="search-trigger" for="site-search">
        <Icon name="search" size={20} />
        <span class="vx-visually-hidden">搜索</span>
      </label>
      <input id="site-search" name="q" maxlength="50" placeholder="搜索文章与问答" required />
    </form>

    {#if user}
      <div class="user-menu">
        <button
          type="button"
          class="user-trigger"
          aria-haspopup="menu"
          aria-expanded={menuOpen}
          onclick={() => (menuOpen = !menuOpen)}
        >
          <Avatar name={user.username} avatarKey={user.avatar_key} size={32} />
          <span class="vx-visually-hidden">{user.username} 的账户菜单</span>
        </button>
        {#if menuOpen}
          <div class="menu-panel" role="menu">
            <div class="menu-head">
              <strong>{user.username}</strong>
              <span>{user.email}</span>
            </div>
            <a role="menuitem" href={`/users/${user.id}`}>我的主页</a>
            <a role="menuitem" href="/user">账户设置</a>
            <a role="menuitem" href="/messages">私信</a>
            <a role="menuitem" href="/notifications">通知</a>
            {#if user.permissions.includes('admin.access')}
              <a role="menuitem" href="/admin">管理后台</a>
            {/if}
            <button type="button" role="menuitem" disabled={loggingOut} onclick={handleLogout}>
              {loggingOut ? '退出中…' : '退出登录'}
            </button>
            {#if logoutError}<p class="menu-error" role="alert">{logoutError}</p>{/if}
          </div>
        {/if}
      </div>
    {:else}
      <a class="appbar-action" href="/login">登录</a>
      <a class="appbar-action" href="/register">注册</a>
    {/if}
  </header>

  <div class="drawer-scrim" class:open={drawerOpen} aria-hidden="true" onclick={closeDrawer}></div>

  <div
    class="vx-drawer"
    class:open={drawerOpen}
    class:collapsed={!drawerPinned}
    bind:this={drawerElement}
    aria-label="站点导航"
  >
    <nav class="drawer-nav">
      {#each navItems as item}
        <a class="drawer-item" class:active={isActive(item.href)} href={item.href} aria-current={isActive(item.href) ? 'page' : undefined}>
          <Icon name={item.icon} size={20} />
          <span>{item.label}</span>
        </a>
      {/each}
      <hr class="vx-divider drawer-divider" />
      <a class="drawer-item" class:active={isActive(peopleItem.href)} href={peopleItem.href}>
        <Icon name={peopleItem.icon} size={20} />
        <span>{peopleItem.label}</span>
      </a>
      {#if user}
        <hr class="vx-divider drawer-divider" />
        <a class="drawer-item" class:active={isActive('/messages')} href="/messages">
          <Icon name="mail" size={20} />
          <span>私信</span>
        </a>
        <a class="drawer-item" class:active={isActive('/notifications')} href="/notifications">
          <Icon name="bell" size={20} />
          <span>通知</span>
        </a>
        {#if user.permissions.includes('admin.access')}
          <a class="drawer-item" class:active={isActive('/admin')} href="/admin">
            <Icon name="shield" size={20} />
            <span>管理后台</span>
          </a>
        {/if}
      {/if}
    </nav>
    <div class="drawer-footer">
      <p>Powered by {siteName}</p>
    </div>
  </div>

  <main
    class="vx-container"
    class:vx-container--list={width === 'list'}
    class:vx-container--detail={width === 'detail'}
  >
    {@render children()}
  </main>
</div>

<style>
  .vx-shell { padding-top: var(--vx-layout-appbar); }

  .vx-appbar {
    position: fixed;
    inset: 0 0 auto;
    z-index: 40;
    display: flex;
    align-items: center;
    height: var(--vx-layout-appbar);
    padding: 0 var(--vx-space-4) 0 var(--vx-space-2);
    /* 液态玻璃：半透明表面 + 背后模糊提饱和，取代投影分层。 */
    background: var(--vx-color-surface-glass);
    -webkit-backdrop-filter: blur(var(--vx-blur-glass)) saturate(1.6);
    backdrop-filter: blur(var(--vx-blur-glass)) saturate(1.6);
  }
  /* 减弱材质降级：不支持 backdrop-filter、系统要求减少透明度/增强对比、
     或产品「减弱材质」开关（html[data-vm-material='solid']）时回退实色表面。
     降级后保留抽屉阴影与层级，交互不受影响。 */
  @supports not ((backdrop-filter: blur(1px)) or (-webkit-backdrop-filter: blur(1px))) {
    .vx-appbar,
    .vx-drawer {
      background: var(--vx-color-surface);
    }
    .appbar-search:focus-within {
      background: var(--vx-color-surface-subtle);
    }
  }
  @media (prefers-reduced-transparency: reduce), (prefers-contrast: more), (forced-colors: active) {
    .vx-appbar,
    .vx-drawer {
      background: var(--vx-color-surface);
      -webkit-backdrop-filter: none;
      backdrop-filter: none;
    }
    .appbar-search:focus-within {
      background: var(--vx-color-surface-subtle);
      -webkit-backdrop-filter: none;
      backdrop-filter: none;
    }
  }
  :root[data-vm-material='solid'] .vx-appbar,
  :root[data-vm-material='solid'] .vx-drawer {
    background: var(--vx-color-surface);
    -webkit-backdrop-filter: none;
    backdrop-filter: none;
  }
  :root[data-vm-material='solid'] .appbar-search:focus-within {
    background: var(--vx-color-surface-subtle);
    -webkit-backdrop-filter: none;
    backdrop-filter: none;
  }

  .appbar-toggle { margin-right: var(--vx-space-1); }
  .appbar-brand {
    margin-right: var(--vx-space-4);
    color: var(--vx-color-text-primary);
    font-size: var(--vx-font-size-headline);
    font-weight: 400;
    letter-spacing: 0.01em;
    white-space: nowrap;
  }
  .appbar-title {
    color: var(--vx-color-text-secondary);
    font-size: var(--vx-font-size-title);
    letter-spacing: 0.02em;
    white-space: nowrap;
  }
  .appbar-spacer { flex: 1; }

  /* 搜索框收起态仅剩图标；点击或键盘聚焦展开为旧站 .search-bar（最大 722，46 高，浅灰底）。 */
  .appbar-search {
    display: flex;
    align-items: center;
    gap: 0;
    width: 40px;
    height: 40px;
    margin-right: var(--vx-space-4);
    color: var(--vx-color-text-tertiary);
    background: transparent;
    border-radius: var(--vx-radius-pill);
    cursor: pointer;
    transition: width var(--vx-motion-standard) var(--vx-motion-easing),
      height var(--vx-motion-standard) var(--vx-motion-easing),
      gap var(--vx-motion-standard) var(--vx-motion-easing),
      padding var(--vx-motion-standard) var(--vx-motion-easing),
      background-color var(--vx-motion-standard) var(--vx-motion-easing),
      box-shadow var(--vx-motion-standard) var(--vx-motion-easing);
  }
  .search-trigger {
    display: flex;
    align-items: center;
    justify-content: center;
    flex: none;
    width: 40px;
    height: 40px;
    border-radius: var(--vx-radius-pill);
  }
  .appbar-search:hover { background: var(--vx-color-surface-hover); }
  .appbar-search:focus-within {
    gap: var(--vx-space-2);
    width: 100%;
    max-width: 722px;
    height: 46px;
    padding: 0 var(--vx-space-4);
    /* 展开态用控件级玻璃：tint + 背后模糊，与顶栏玻璃区分层次。 */
    background: var(--vx-color-surface-glass-control);
    -webkit-backdrop-filter: blur(var(--vx-blur-glass)) saturate(1.6);
    backdrop-filter: blur(var(--vx-blur-glass)) saturate(1.6);
    border-radius: var(--vx-radius-card);
    cursor: text;
    box-shadow: var(--vx-shadow-card);
  }
  .appbar-search input {
    flex: 1;
    min-width: 0;
    width: 0;
    height: 100%;
    padding: 0;
    color: var(--vx-color-text-primary);
    background: transparent;
    border: 0;
    outline: none;
    opacity: 0;
    transition: opacity 150ms var(--vx-motion-easing);
  }
  .appbar-search:focus-within input { opacity: 1; }

  .appbar-action {
    margin-left: var(--vx-space-2);
    padding: var(--vx-space-1) var(--vx-space-3);
    color: var(--vx-color-text-secondary);
    font-size: var(--vx-font-size-small);
    font-weight: 500;
    border-radius: var(--vx-radius-xs);
  }
  .appbar-action:hover { background: var(--vx-color-surface-hover); color: var(--vx-color-text-primary); }

  .user-menu { position: relative; margin-left: var(--vx-space-2); }
  .user-trigger {
    display: flex;
    padding: 0;
    background: none;
    border: 0;
    border-radius: var(--vx-radius-pill);
    cursor: pointer;
  }
  .user-trigger:hover { box-shadow: var(--vx-shadow-card); }
  .menu-panel {
    position: absolute;
    top: calc(100% + var(--vx-space-2));
    right: 0;
    display: flex;
    flex-direction: column;
    min-width: 200px;
    padding: var(--vx-space-2) 0;
    background: var(--vx-color-surface);
    border-radius: var(--vx-radius-card);
    box-shadow: var(--vx-shadow-overlay);
  }
  .menu-head {
    display: flex;
    flex-direction: column;
    gap: 2px;
    padding: var(--vx-space-2) var(--vx-space-4) var(--vx-space-3);
    border-bottom: 1px solid var(--vx-color-divider);
  }
  .menu-head span { color: var(--vx-color-text-secondary); font-size: var(--vx-font-size-caption); }
  .menu-panel a,
  .menu-panel button {
    padding: var(--vx-space-2) var(--vx-space-4);
    color: var(--vx-color-text-primary);
    font-size: var(--vx-font-size-small);
    text-align: left;
    background: none;
    border: 0;
    cursor: pointer;
  }
  .menu-panel a:hover,
  .menu-panel button:hover { background: var(--vx-color-surface-hover); }
  .menu-panel button:disabled { color: var(--vx-color-text-tertiary); cursor: default; }
  .menu-error { padding: 0 var(--vx-space-4) var(--vx-space-2); color: var(--vx-color-error); font-size: var(--vx-font-size-caption); }

  .drawer-scrim {
    position: fixed;
    inset: var(--vx-layout-appbar) 0 0;
    z-index: 30;
    background: rgba(0, 0, 0, 0.32);
    opacity: 0;
    visibility: hidden;
    transition: opacity var(--vx-motion-standard) var(--vx-motion-easing),
      visibility var(--vx-motion-standard) var(--vx-motion-easing);
  }
  .drawer-scrim.open { opacity: 1; visibility: visible; }

  .vx-drawer {
    position: fixed;
    top: var(--vx-layout-appbar);
    bottom: 0;
    left: 0;
    z-index: 35;
    display: flex;
    flex-direction: column;
    width: var(--vx-layout-drawer);
    /* 覆盖态与顶栏同款液态玻璃；桌面常驻态见下方媒体查询，恢复实色省去无意义模糊。 */
    background: var(--vx-color-surface-glass);
    -webkit-backdrop-filter: blur(var(--vx-blur-glass)) saturate(1.6);
    backdrop-filter: blur(var(--vx-blur-glass)) saturate(1.6);
    box-shadow: var(--vx-shadow-overlay);
    transform: translateX(-102%);
    transition: transform var(--vx-motion-standard) var(--vx-motion-easing);
  }
  /* 液态感来自静态绘制的顶部弱高光：无滤镜动画、无实时折射、无描边线条。 */
  .vx-drawer::before {
    content: '';
    position: absolute;
    inset: 0;
    z-index: -1;
    pointer-events: none;
    border-radius: inherit;
    background: linear-gradient(to bottom, hsla(0, 0%, 100%, 0.1), hsla(0, 0%, 100%, 0) 36%);
  }
  .vx-drawer.open { transform: translateX(0); }

  .drawer-nav { flex: 1; padding: var(--vx-space-2) 0; overflow-y: auto; }
  .drawer-item {
    display: flex;
    align-items: center;
    gap: var(--vx-space-4);
    height: 48px;
    margin-right: var(--vx-space-3);
    padding: 0 var(--vx-space-5) 0 var(--vx-space-5);
    color: var(--vx-color-text-primary);
    font-size: var(--vx-font-size-body);
    border-radius: 0 50px 50px 0;
    transition: background-color var(--vx-motion-standard) var(--vx-motion-easing),
      transform 120ms var(--vx-motion-easing);
  }
  .drawer-item:hover { background: var(--vx-color-surface-hover); }
  /* 按下轻微缩放、松开快速恢复；reduced-motion 下由全局覆盖即时切换。 */
  .drawer-item:active { transform: scale(0.98); }
  /* 选中项：中性灰底，文字保持正文色不另行染色。 */
  .drawer-item.active {
    background: var(--vx-color-nav-active-bg);
    font-weight: 500;
  }
  /* 分组分隔线只留间距不画线。 */
  .drawer-divider { margin: var(--vx-space-2) 0; border: 0; }
  .drawer-footer { padding: var(--vx-space-5) var(--vx-space-4); }
  .drawer-footer p {
    overflow: hidden;
    color: var(--vx-color-text-tertiary);
    font-size: var(--vx-font-size-small);
    white-space: nowrap;
    text-overflow: ellipsis;
  }

  /* 桌面端抽屉常驻：旧站 mdui-drawer-body-left，1024px 起贴左展开、内容右移。 */
  @media (min-width: 1024px) {
    .vx-shell:not(.drawer-collapsed) { padding-left: var(--vx-layout-drawer); }
    .vx-drawer {
      z-index: 20;
      background: var(--vx-color-surface);
      -webkit-backdrop-filter: none;
      backdrop-filter: none;
      box-shadow: none;
      transform: translateX(0);
    }
    .vx-drawer.collapsed { transform: translateX(-102%); }
    .drawer-scrim { display: none; }
  }

  @media (max-width: 1023.9px) {
    .appbar-title { display: none; }
  }
  @media (max-width: 599.9px) {
    .vx-shell { padding-top: 56px; }
    .vx-appbar { height: 56px; padding-right: var(--vx-space-2); }
    .drawer-scrim { inset: 56px 0 0; }
    .vx-drawer { top: 56px; }
    .appbar-brand { font-size: var(--vx-font-size-title); }
    .appbar-search { display: none; }
  }
</style>
