<script lang="ts">
  import { page } from '$app/state';
  import UserCard from '#lib/components/UserCard.svelte';
  import TabBar from '#lib/components/TabBar.svelte';
  import Pager from '#lib/components/Pager.svelte';
  import type { AuthUser } from '#lib/api/auth.js';
  import type { PageProps } from './$types';

  let { data }: PageProps = $props();

  const currentUser = $derived((page.data.siteUser ?? null) as AuthUser | null);
  const tabs = [
    { value: 'recommended', label: '人员推荐' },
    { value: 'new', label: '最新加入' }
  ];
</script>

<svelte:head>
  <title>人脉 · VOIDXO</title>
  <meta name="description" content="VOIDXO 的成员列表，关注感兴趣的作者。" />
</svelte:head>

<h1 class="vx-visually-hidden">人脉</h1>

<TabBar items={tabs} active={data.sort} basePath="/users" />

{#if data.loadError}
  <p class="state" role="status">成员加载失败，请刷新重试。</p>
{:else if data.users.length === 0}
  <p class="state">暂无成员。</p>
{:else}
  <div class="user-grid">
    {#each data.users as user (user.id)}
      <UserCard {user} canFollow={Boolean(currentUser)} following={data.following[String(user.id)] ?? false} />
    {/each}
  </div>
{/if}

<Pager page={data.page} hasMore={data.hasMore} basePath="/users" keep={data.sort === 'new' ? { sort: 'new' } : {}} />

<style>
  .user-grid {
    display: grid;
    grid-template-columns: 1fr;
    gap: var(--vx-space-4);
    padding-top: var(--vx-space-4);
  }
  @media (min-width: 600px) { .user-grid { grid-template-columns: repeat(2, minmax(0, 1fr)); } }
  @media (min-width: 1024px) { .user-grid { grid-template-columns: repeat(3, minmax(0, 1fr)); } }
  .state { padding: var(--vx-space-8) 0; color: var(--vx-color-text-tertiary); font-size: var(--vx-font-size-small); text-align: center; }
</style>
