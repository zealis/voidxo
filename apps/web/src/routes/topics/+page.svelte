<script lang="ts">
  import { page } from '$app/state';
  import TopicCard from '#lib/components/TopicCard.svelte';
  import TabBar from '#lib/components/TabBar.svelte';
  import Pager from '#lib/components/Pager.svelte';
  import type { AuthUser } from '#lib/api/auth.js';
  import type { PageProps } from './$types';

  let { data }: PageProps = $props();

  const currentUser = $derived((page.data.siteUser ?? null) as AuthUser | null);
  const tabs = [
    { value: 'recommended', label: '推荐话题' },
    { value: 'new', label: '最新话题' }
  ];
</script>

<svelte:head>
  <title>话题 · VOIDXO</title>
  <meta name="description" content="浏览 VOIDXO 的全部话题，关注感兴趣的方向。" />
</svelte:head>

<h1 class="vx-visually-hidden">话题</h1>

<TabBar items={tabs} active={data.sort} basePath="/topics" />

{#if data.loadError}
  <p class="state" role="status">话题加载失败，请刷新重试。</p>
{:else if data.topics.length === 0}
  <p class="state">暂无话题。</p>
{:else}
  <div class="topic-grid">
    {#each data.topics as topic (topic.id)}
      <TopicCard {topic} canFollow={Boolean(currentUser)} following={data.following[String(topic.id)] ?? false} />
    {/each}
  </div>
{/if}

<Pager page={data.page} hasMore={data.hasMore} basePath="/topics" keep={data.sort === 'new' ? { sort: 'new' } : {}} />

<style>
  .topic-grid {
    display: grid;
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: var(--vx-space-4);
    padding-top: var(--vx-space-4);
  }
  @media (min-width: 600px) { .topic-grid { grid-template-columns: repeat(3, minmax(0, 1fr)); } }
  @media (min-width: 1024px) { .topic-grid { grid-template-columns: repeat(4, minmax(0, 1fr)); } }
  .state { padding: var(--vx-space-8) 0; color: var(--vx-color-text-tertiary); font-size: var(--vx-font-size-small); text-align: center; }
</style>
