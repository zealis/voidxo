<script lang="ts">
  import { page } from '$app/state';
  import Icon from '#lib/components/Icon.svelte';
  import ListRow from '#lib/components/ListRow.svelte';
  import SectionCard from '#lib/components/SectionCard.svelte';
  import FollowButton from '#lib/components/FollowButton.svelte';
  import type { AuthUser } from '#lib/api/auth.js';
  import type { PageProps } from './$types';

  let { data }: PageProps = $props();

  const currentUser = $derived((page.data.siteUser ?? null) as AuthUser | null);
  const hue = $derived(
    Array.from(data.topic.name).reduce((acc, char) => (acc * 31 + char.codePointAt(0)!) % 360, 11)
  );
  const coverStyle = $derived(
    data.topic.cover_key
      ? `background-image:url(/media/${data.topic.cover_key}/middle)`
      : `background-image:linear-gradient(135deg,hsl(${hue} 62% 88%),hsl(${(hue + 38) % 360} 58% 78%))`
  );
</script>

<svelte:head>
  <title>{data.topic.name} · 话题 · VOIDXO</title>
  <meta name="description" content={data.topic.description || `${data.topic.name} 话题下的提问与文章`} />
</svelte:head>

<nav class="page-nav" aria-label="返回">
  <a class="back-link" href="/topics"><Icon name="arrow-left" size={18} />返回话题</a>
</nav>

<header class="topic-header vx-card">
  <div class="topic-cover" style={coverStyle}></div>
  <div class="topic-info">
    <div class="topic-main">
      <h1>{data.topic.name}</h1>
      <p class="topic-desc">{data.topic.description || '这个话痨很懒，还没有写简介。'}</p>
      <p class="topic-counts">
        {data.topic.follower_count} 人关注 · {data.topic.question_count} 个提问 · {data.topic.article_count} 篇文章
      </p>
    </div>
    <FollowButton
      kind="topic"
      id={data.topic.id}
      initial={data.follow}
      canFollow={Boolean(currentUser)}
    />
  </div>
</header>

{#if data.loadError}
  <p class="state" role="status">部分内容加载失败，请刷新重试。</p>
{/if}

<div class="sections">
  <SectionCard title="提问" href={`/questions?topic_id=${data.topic.id}`}>
    {#if data.questions.length === 0}
      <p class="vx-empty">该话题下还没有提问。</p>
    {:else}
      {#each data.questions as question (question.id)}
        <ListRow
          href={`/questions/${question.id}`}
          title={question.title}
          username={question.username}
          avatarKey={question.avatar_key}
          meta={`${question.answer_count} 个回答 · ${question.updated_at.slice(0, 10)}`}
        />
      {/each}
    {/if}
  </SectionCard>

  <SectionCard title="文章" href={`/articles?topic_id=${data.topic.id}`}>
    {#if data.articles.length === 0}
      <p class="vx-empty">该话题下还没有文章。</p>
    {:else}
      {#each data.articles as article (article.id)}
        <ListRow
          href={`/articles/${article.id}`}
          title={article.title}
          username={article.username}
          avatarKey={article.avatar_key}
          meta={`${article.comment_count} 条评论 · ${article.created_at.slice(0, 10)}`}
        />
      {/each}
    {/if}
  </SectionCard>
</div>

<style>
  .page-nav { padding: var(--vx-space-2) 0 var(--vx-space-3); }
  .back-link {
    display: inline-flex;
    align-items: center;
    gap: var(--vx-space-2);
    padding: var(--vx-space-2) var(--vx-space-3);
    color: var(--vx-color-primary);
    font-size: var(--vx-font-size-small);
    font-weight: 500;
    background: var(--vx-color-primary-soft);
    border-radius: var(--vx-radius-control);
  }
  .back-link:hover { background: var(--vx-color-surface-hover); }

  .topic-header { overflow: hidden; }
  .topic-cover { aspect-ratio: 16 / 4; background-position: center; background-size: cover; }
  .topic-info { display: flex; align-items: flex-start; justify-content: space-between; gap: var(--vx-space-4); padding: var(--vx-space-5) var(--vx-space-6); }
  .topic-main { min-width: 0; }
  .topic-main h1 { font-size: var(--vx-font-size-headline); font-weight: 400; }
  .topic-desc { margin-top: var(--vx-space-2); color: var(--vx-color-text-secondary); font-size: var(--vx-font-size-body); }
  .topic-counts { margin-top: var(--vx-space-2); color: var(--vx-color-text-tertiary); font-size: var(--vx-font-size-small); }

  .sections { display: grid; grid-template-columns: 1fr; gap: var(--vx-space-6); padding-top: var(--vx-space-6); }
  @media (min-width: 900px) { .sections { grid-template-columns: repeat(2, minmax(0, 1fr)); } }
  .state { padding: var(--vx-space-4) 0; color: var(--vx-color-error); font-size: var(--vx-font-size-caption); }

  @media (max-width: 599.9px) {
    .topic-cover { aspect-ratio: 16 / 6; }
    .topic-info { flex-direction: column; padding: var(--vx-space-4); }
  }
</style>
