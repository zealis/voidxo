<script lang="ts">
  import ListRow from '#lib/components/ListRow.svelte';
  import SectionCard from '#lib/components/SectionCard.svelte';
  import TabBar from '#lib/components/TabBar.svelte';
  import Pager from '#lib/components/Pager.svelte';
  import Fab from '#lib/components/Fab.svelte';
  import type { PageProps } from './$types';

  let { data }: PageProps = $props();

  const tabs = [
    { value: 'recent', label: '最近更新' },
    { value: 'popular', label: '热门文章' },
    { value: 'new', label: '最新发布' }
  ];

  function meta(comments: number, votes: number, date: string) {
    return `${comments} 条评论 · ${votes} 赞同 · ${date.slice(0, 10)}`;
  }
</script>

<svelte:head>
  <title>文章 · VOIDXO</title>
  <meta name="description" content="VOIDXO 的文章与分享，按最近更新、热门与发布时间浏览。" />
</svelte:head>

<h1 class="vx-visually-hidden">文章</h1>

<TabBar items={tabs} active={data.sort} basePath="/articles" />

{#if data.topicId}
  <p class="filter-note">已按话题筛选 · <a href="/articles">查看全部</a></p>
{/if}

{#if data.loadError}
  <p class="state" role="status">文章加载失败，请刷新重试。</p>
{:else if data.articles.length === 0}
  <p class="state">还没有文章，来写第一篇吧。</p>
{:else}
  <SectionCard title="文章列表" href="/articles" flush>
    {#each data.articles as article (article.id)}
      <ListRow
        href={`/articles/${article.id}`}
        title={article.title}
        username={article.username}
        avatarKey={article.avatar_key}
        meta={meta(article.comment_count, article.vote_count, article.created_at)}
      />
    {/each}
  </SectionCard>
{/if}

<Pager page={data.page} hasMore={data.hasMore} basePath="/articles" keep={{...(data.topicId ? { topic_id: String(data.topicId) } : {}), ...(data.sort === 'recent' ? {} : { sort: data.sort })}} />

<Fab href="/articles/new" label="写文章" />

<style>
  .filter-note { padding: var(--vx-space-3) var(--vx-space-2) 0; color: var(--vx-color-text-secondary); font-size: var(--vx-font-size-small); }
  .filter-note a { color: var(--vx-color-primary); }
  .state { padding: var(--vx-space-8) 0; color: var(--vx-color-text-tertiary); font-size: var(--vx-font-size-small); text-align: center; }
</style>
