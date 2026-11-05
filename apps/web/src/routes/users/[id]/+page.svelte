<script lang="ts">
  import Avatar from '#lib/components/Avatar.svelte';
  import FollowButton from '#lib/components/FollowButton.svelte';
  import type { UserContentItem } from '#lib/api/users.js';
  import type { PageProps } from './$types';

  let { data }: PageProps = $props();

  const tabs = $derived([
    { key: 'questions', label: '问题', count: data.profile?.question_count ?? 0 },
    { key: 'answers', label: '回答', count: data.profile?.answer_count ?? 0 },
    { key: 'articles', label: '文章', count: data.profile?.article_count ?? 0 },
    { key: 'comments', label: '评论', count: data.profile?.comment_count ?? 0 }
  ] as const);

  function target(item: UserContentItem): string {
    switch (item.kind) {
      case 'question':
        return `/questions/${item.id}`;
      case 'article':
        return `/articles/${item.id}`;
      case 'answer':
        return item.target_id ? `/questions/${item.target_id}` : '';
      default:
        if (item.target_type === 'question') return `/questions/${item.target_id}`;
        if (item.target_type === 'article') return `/articles/${item.target_id}`;
        return '';
    }
  }

  function pageHref(page: number): string {
    return `/users/${data.profile?.id}?tab=${data.tab}&page=${page}`;
  }
</script>

<svelte:head>
  <title>{data.profile ? `${data.profile.username} · VOIDXO` : '用户 · VOIDXO'}</title>
  {#if data.profile?.bio}<meta name="description" content={data.profile.bio} />{/if}
</svelte:head>


  {#if data.notice}
    <p class="empty">{data.notice}</p>
  {:else if data.loadError && !data.profile}
    <p class="error" role="alert">资料加载失败，请稍后刷新。</p>
  {:else if data.profile}
    <header class="profile">
      {#if data.profile.cover_key}
        <div class="cover"><img src={`/media/${data.profile.cover_key}/middle`} alt="" /></div>
      {/if}
      <div class="identity">
        <Avatar
          name={data.profile.username}
          avatarKey={data.profile.avatar_key}
          size={72}
          alt={`${data.profile.username} 的头像`}
        />
        <div class="intro">
          <h1>{data.profile.username}</h1>
          {#if data.profile.headline}<p class="headline">{data.profile.headline}</p>{/if}
          {#if data.profile.bio}<p class="bio">{data.profile.bio}</p>{/if}
          <p class="meta">
            <span>加入于 {data.profile.created_at.slice(0, 10)}</span>
            {#if data.profile.company}<span>· {data.profile.company}</span>{/if}
            {#if data.profile.location}<span>· {data.profile.location}</span>{/if}
            {#if data.profile.blog}<a href={data.profile.blog} rel="nofollow noopener" target="_blank">{data.profile.blog}</a>{/if}
          </p>
        </div>
        {#if data.user && data.user.id !== data.profile.id}
          <FollowButton kind="user" id={data.profile.id} initial={data.follow} canFollow={true} />
        {/if}
      </div>
      <dl class="counters">
        <div><dt>关注</dt><dd>{data.profile.followee_count}</dd></div>
        <div><dt>粉丝</dt><dd>{data.profile.follower_count}</dd></div>
        <div><dt>问题</dt><dd>{data.profile.question_count}</dd></div>
        <div><dt>回答</dt><dd>{data.profile.answer_count}</dd></div>
        <div><dt>文章</dt><dd>{data.profile.article_count}</dd></div>
        <div><dt>评论</dt><dd>{data.profile.comment_count}</dd></div>
      </dl>
    </header>

    <nav class="tabs" aria-label="内容分类">
      {#each tabs as tab (tab.key)}
        <a
          href={`/users/${data.profile.id}?tab=${tab.key}`}
          class:active={data.tab === tab.key}
          aria-current={data.tab === tab.key ? 'page' : undefined}
        >{tab.label}<span>{tab.count}</span></a>
      {/each}
    </nav>

    {#if data.loadError}
      <p class="error" role="alert">内容加载失败，请稍后刷新。</p>
    {/if}

    <ul class="items">
      {#each data.items as item (item.kind + '-' + item.id)}
        <li>
          {#if target(item)}
            <a class="title" href={target(item)}>{item.title || item.excerpt}</a>
          {:else}
            <span class="title">{item.title || item.excerpt}</span>
          {/if}
          {#if item.title && item.excerpt}<p class="excerpt">{item.excerpt}</p>{/if}
          <p class="meta">
            <time datetime={item.created_at}>{item.created_at.slice(0, 10)}</time>
            {#if item.kind !== 'comment'}<span>· 评论 {item.comment_count}</span>{/if}
            <span>· 票数 {item.vote_count}</span>
          </p>
        </li>
      {:else}
        <li class="empty">还没有内容。</li>
      {/each}
    </ul>

    {#if data.page > 1 || data.hasMore}
      <nav class="pager" aria-label="分页">
        {#if data.page > 1}<a href={pageHref(data.page - 1)}>上一页</a>{:else}<span class="muted">上一页</span>{/if}
        <span class="muted">第 {data.page} 页</span>
        {#if data.hasMore}<a href={pageHref(data.page + 1)}>下一页</a>{:else}<span class="muted">下一页</span>{/if}
      </nav>
    {/if}
  {/if}

<style>
  .profile { border: 1px solid var(--vx-color-border); border-radius: var(--vx-radius-card); background: var(--vx-color-surface); overflow: hidden; }
  .cover { height: 120px; overflow: hidden; background: var(--vx-color-bg); }
  .cover img { width: 100%; height: 100%; object-fit: cover; }
  .identity { display: flex; gap: var(--vx-space-4); padding: var(--vx-space-4); align-items: flex-start; }
  .intro { flex: 1; min-width: 0; }
  h1 { margin: 0; font-size: 20px; font-weight: 600; }
  .headline { margin: 2px 0 0; color: var(--vx-color-text-secondary); font-size: 13px; }
  .bio { margin: var(--vx-space-2) 0 0; font-size: 14px; }
  .meta { margin: 4px 0 0; color: var(--vx-color-text-secondary); font-size: 12px; display: flex; flex-wrap: wrap; gap: 6px; }
  .meta a { color: var(--vx-color-text-secondary); }
  .counters { display: flex; flex-wrap: wrap; gap: var(--vx-space-6); margin: 0; padding: 0 var(--vx-space-4) var(--vx-space-4); }
  .counters div { display: flex; flex-direction: column; }
  .counters dt { color: var(--vx-color-text-secondary); font-size: 12px; }
  .counters dd { margin: 0; font-size: 16px; font-weight: 600; }
  .tabs { display: flex; gap: var(--vx-space-1); margin: var(--vx-space-4) 0 var(--vx-space-3); border-bottom: 1px solid var(--vx-color-border); }
  .tabs a { padding: 8px 12px; color: var(--vx-color-text-secondary); font-size: 13px; text-decoration: none; border-bottom: 2px solid transparent; }
  .tabs a.active { color: var(--vx-color-text-primary); border-bottom-color: var(--vx-color-text-primary); }
  .tabs span { margin-left: 4px; font-size: 11px; }
  .items { list-style: none; margin: 0; padding: 0; display: flex; flex-direction: column; gap: 1px; background: var(--vx-color-border); border: 1px solid var(--vx-color-border); border-radius: var(--vx-radius-card); overflow: hidden; }
  .items li { background: var(--vx-color-surface); padding: var(--vx-space-3); }
  .title { font-size: 14px; font-weight: 500; color: inherit; text-decoration: none; display: -webkit-box; -webkit-line-clamp: 2; line-clamp: 2; -webkit-box-orient: vertical; overflow: hidden; }
  a.title:hover { text-decoration: underline; }
  .excerpt { margin: 4px 0 0; color: var(--vx-color-text-secondary); font-size: 13px; overflow: hidden; text-overflow: ellipsis; display: -webkit-box; -webkit-line-clamp: 2; line-clamp: 2; -webkit-box-orient: vertical; }
  .meta { color: var(--vx-color-text-secondary); font-size: 12px; }
  .empty { color: var(--vx-color-text-secondary); font-size: 13px; padding: var(--vx-space-6); text-align: center; background: var(--vx-color-surface); }
  .error { color: var(--vx-color-error); font-size: 13px; }
  .pager { display: flex; gap: var(--vx-space-3); align-items: center; margin-top: var(--vx-space-4); font-size: 13px; }
  .pager a { color: var(--vx-color-text-primary); }
  .muted { color: var(--vx-color-text-secondary); }
</style>
