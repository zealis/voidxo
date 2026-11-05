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
    { value: 'popular', label: '热门提问' },
    { value: 'new', label: '最新发布' }
  ];

  function meta(answers: number, votes: number, date: string) {
    return `${answers} 个回答 · ${votes} 赞同 · ${date.slice(0, 10)}`;
  }
</script>

<svelte:head>
  <title>问答 · VOIDXO</title>
  <meta name="description" content="VOIDXO 的提问与回答，按最近更新、热门与发布时间浏览。" />
</svelte:head>

<h1 class="vx-visually-hidden">问答</h1>

<TabBar items={tabs} active={data.sort} basePath="/questions" />

{#if data.topicId}
  <p class="filter-note">已按话题筛选 · <a href="/questions">查看全部</a></p>
{/if}

{#if data.loadError}
  <p class="state" role="status">提问加载失败，请刷新重试。</p>
{:else if data.questions.length === 0}
  <p class="state">还没有提问，来问第一个问题吧。</p>
{:else}
  <SectionCard title="提问列表" href="/questions" flush>
    {#each data.questions as question (question.id)}
      <ListRow
        href={`/questions/${question.id}`}
        title={question.title}
        username={question.username}
        avatarKey={question.avatar_key}
        meta={meta(question.answer_count, question.vote_count, question.updated_at)}
      />
    {/each}
  </SectionCard>
{/if}

<Pager page={data.page} hasMore={data.hasMore} basePath="/questions" keep={{...(data.topicId ? { topic_id: String(data.topicId) } : {}), ...(data.sort === 'recent' ? {} : { sort: data.sort })}} />

<Fab href="/questions/new" label="提问题" />

<style>
  .filter-note { padding: var(--vx-space-3) var(--vx-space-2) 0; color: var(--vx-color-text-secondary); font-size: var(--vx-font-size-small); }
  .filter-note a { color: var(--vx-color-primary); }
  .state { padding: var(--vx-space-8) 0; color: var(--vx-color-text-tertiary); font-size: var(--vx-font-size-small); text-align: center; }
</style>
