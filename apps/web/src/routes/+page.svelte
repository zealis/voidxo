<script lang="ts">
  import { page } from '$app/state';
  import Icon from '#lib/components/Icon.svelte';
  import Avatar from '#lib/components/Avatar.svelte';
  import TopicCard from '#lib/components/TopicCard.svelte';
  import SectionCard from '#lib/components/SectionCard.svelte';
  import type { AuthUser } from '#lib/api/auth.js';
  import type { PageProps } from './$types';

  let { data }: PageProps = $props();

  const currentUser = $derived((page.data.siteUser ?? null) as AuthUser | null);
  const siteName = $derived((page.data.siteOptions as { site_name?: string } | null)?.site_name ?? 'VOIDXO');
  const siteDescription = $derived(
    (page.data.siteOptions as { site_description?: string } | null)?.site_description ?? ''
  );
</script>

<svelte:head>
  <title>{siteName}</title>
  <meta name="description" content={siteDescription} />
</svelte:head>

<h1 class="vx-visually-hidden">{siteName} 社区首页</h1>

{#if data.loadError}
  <p class="load-hint" role="status">部分内容加载失败，刷新可重试。</p>
{/if}

{#if data.topics.length > 0}
  <section class="topics-block" aria-labelledby="topics-title">
    <a class="block-head" href="/topics">
      <span id="topics-title">推荐话题</span>
      <Icon name="arrow-right" size={18} />
    </a>
    <div class="topic-grid">
      {#each data.topics as topic (topic.id)}
        <TopicCard {topic} canFollow={Boolean(currentUser)} />
      {/each}
    </div>
  </section>
{/if}

<div class="items-wrapper">
  <div class="items-col">
    <SectionCard title="最近更新提问" href="/questions">
      {#if data.questionsRecent.length === 0}
        <p class="vx-empty">暂无提问</p>
      {:else}
        {#each data.questionsRecent as question (question.id)}
          <a class="home-row" href={`/questions/${question.id}`}>
            <span class="row-avatar"><Avatar name={question.username} avatarKey={question.avatar_key} size={20} /></span>
            <span class="row-title vx-clamp-1">{question.title}</span>
          </a>
        {/each}
      {/if}
    </SectionCard>
  </div>
  <div class="items-col">
    <SectionCard title="最近热门提问" href="/questions?sort=popular">
      {#if data.questionsPopular.length === 0}
        <p class="vx-empty">暂无提问</p>
      {:else}
        {#each data.questionsPopular as question (question.id)}
          <a class="home-row" href={`/questions/${question.id}`}>
            <span class="row-avatar"><Avatar name={question.username} avatarKey={question.avatar_key} size={20} /></span>
            <span class="row-title vx-clamp-1">{question.title}</span>
          </a>
        {/each}
      {/if}
    </SectionCard>
  </div>
</div>

<div class="items-wrapper">
  <div class="items-col">
    <SectionCard title="最新文章" href="/articles">
      {#if data.articlesRecent.length === 0}
        <p class="vx-empty">暂无文章</p>
      {:else}
        {#each data.articlesRecent as article (article.id)}
          <a class="home-row" href={`/articles/${article.id}`}>
            <span class="row-avatar"><Avatar name={article.username} avatarKey={article.avatar_key} size={20} /></span>
            <span class="row-title vx-clamp-1">{article.title}</span>
          </a>
        {/each}
      {/if}
    </SectionCard>
  </div>
  <div class="items-col">
    <SectionCard title="最近热门文章" href="/articles?sort=popular">
      {#if data.articlesPopular.length === 0}
        <p class="vx-empty">暂无文章</p>
      {:else}
        {#each data.articlesPopular as article (article.id)}
          <a class="home-row" href={`/articles/${article.id}`}>
            <span class="row-avatar"><Avatar name={article.username} avatarKey={article.avatar_key} size={20} /></span>
            <span class="row-title vx-clamp-1">{article.title}</span>
          </a>
        {/each}
      {/if}
    </SectionCard>
  </div>
</div>

{#if data.stats}
  <footer class="site-stats" aria-label="站点数据">
    <span>{data.stats.users} 位成员</span>
    <span>{data.stats.questions} 个问题</span>
    <span>{data.stats.answers} 个回答</span>
    <span>{data.stats.articles} 篇文章</span>
    <span>{data.stats.comments} 条评论</span>
  </footer>
{/if}

<style>
  .load-hint { padding: var(--vx-space-2) var(--vx-space-4); color: var(--vx-color-error); font-size: var(--vx-font-size-caption); }

  .topics-block { margin-bottom: var(--vx-space-6); }
  .block-head {
    display: flex;
    align-items: center;
    justify-content: space-between;
    height: 52px;
    padding: 0 var(--vx-space-4);
    color: var(--vx-color-text-secondary);
    font-size: var(--vx-font-size-subtitle);
    font-weight: 500;
  }
  .block-head:hover { color: var(--vx-color-text-primary); }
  .block-head :global(svg) { color: var(--vx-color-primary); }

  /* 旧站话题网格：默认 2 列，600 起 3 列，1024 起 4 列。 */
  .topic-grid { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: var(--vx-space-4); }
  @media (min-width: 600px) { .topic-grid { grid-template-columns: repeat(3, minmax(0, 1fr)); } }
  @media (min-width: 1024px) { .topic-grid { grid-template-columns: repeat(4, minmax(0, 1fr)); } }

  /* 旧站 items-container：1024 起两列，以下单列堆叠。 */
  .items-wrapper { display: grid; grid-template-columns: 1fr; gap: var(--vx-space-6); margin-bottom: var(--vx-space-6); }
  @media (min-width: 1024px) { .items-wrapper { grid-template-columns: repeat(2, minmax(0, 1fr)); } }
  .items-col { min-width: 0; }

  /* 旧站首页条目：52 高、左内边距 64、头像 20。 */
  .home-row {
    position: relative;
    display: flex;
    align-items: center;
    gap: var(--vx-space-3);
    height: 52px;
    padding: 0 var(--vx-space-6) 0 64px;
    font-size: 15px;
    transition: background-color var(--vx-motion-standard) var(--vx-motion-easing);
  }
  .home-row:hover { background: var(--vx-color-surface-hover); }
  .row-avatar { position: absolute; left: 24px; display: flex; }
  .row-title { flex: 1; min-width: 0; }

  .site-stats {
    display: flex;
    flex-wrap: wrap;
    gap: var(--vx-space-4);
    padding: var(--vx-space-4) 0 var(--vx-space-2);
    color: var(--vx-color-text-tertiary);
    font-size: var(--vx-font-size-caption);
  }

  @media (max-width: 599.9px) {
    .home-row { padding: 0 var(--vx-space-4) 0 48px; }
    .row-avatar { left: 16px; }
  }
</style>
