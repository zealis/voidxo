<script lang="ts">
  import { goto, invalidateAll } from '$app/navigation';
  import VxButton from '#lib/components/ui/VxButton.svelte';
  import Avatar from '#lib/components/Avatar.svelte';
  import Icon from '#lib/components/Icon.svelte';
  import CommentSection from '#lib/components/CommentSection.svelte';
  import FollowButton from '#lib/components/FollowButton.svelte';
  import VoteButtons from '#lib/components/VoteButtons.svelte';
  import { ContentRequestError, deleteArticle } from '#lib/api/content.js';
  import type { PageProps } from './$types';

  let { data }: PageProps = $props();
  let deleting = $state(false);
  let deleteError = $state('');

  const canManage = $derived(
    !!data.user &&
      !!data.article &&
      (data.user.permissions.includes('content.manage') ||
        data.user.permissions.includes('content.moderate') ||
        data.user.id === data.article.user_id)
  );

  async function removeArticle() {
    if (!data.article || deleting) return;
    if (!confirm('确定删除这篇文章？')) return;
    deleting = true;
    deleteError = '';
    try {
      await deleteArticle(data.article.id);
      await goto('/articles');
    } catch (cause) {
      if (cause instanceof ContentRequestError) {
        deleteError = cause.message;
      } else {
        deleteError = '删除失败，请稍后重试';
      }
      await invalidateAll();
    } finally {
      deleting = false;
    }
  }
</script>

<svelte:head>
  <title>{data.article?.title ?? '文章'} · VOIDXO</title>
  <meta name="description" content={data.article?.title ?? ''} />
</svelte:head>

<nav class="page-nav" aria-label="返回">
  <a class="back-link" href="/articles"><Icon name="arrow-left" size={18} />返回文章</a>
</nav>

{#if data.loadError}
  <div class="error-message" role="alert">
    <span>文章加载失败，请刷新重试。</span>
    <VxButton variant="outlined" onclick={() => location.reload()}>重试</VxButton>
  </div>
{:else if data.article}
  <article class="vx-card vx-card--shadow article" aria-labelledby="article-title">
    <h1 class="article-title" id="article-title">{data.article.title}</h1>

    <div class="user-line">
      <a class="user-avatar" href={`/users/${data.article.user_id}`} aria-label={`${data.article.username} 的主页`}>
        <Avatar name={data.article.username} avatarKey={data.article.avatar_key} size={40} />
      </a>
      <div class="user-meta">
        <a class="user-name" href={`/users/${data.article.user_id}`}>{data.article.username}</a>
        <span class="user-time">{data.article.updated_at.slice(0, 16).replace('T', ' ')}</span>
      </div>
    </div>

    <div class="prose">{@html data.article.content_rendered}</div>

    {#if data.article.topics.length > 0}
      <ul class="topic-bar" aria-label="所属话题">
        {#each data.article.topics as topic (topic.id)}
          <li><a class="chip" href={`/topics/${topic.id}`}>{topic.name}</a></li>
        {/each}
      </ul>
    {/if}

    <div class="article-actions">
      <VoteButtons kind="article" id={data.article.id} initial={data.vote} canVote={!!data.user} />
      <FollowButton kind="article" id={data.article.id} initial={data.follow} canFollow={!!data.user} />
      <div class="action-spacer"></div>
      {#if canManage}
        <a class="text-link" href={`/articles/${data.article.id}/edit`}>编辑</a>
        <button type="button" class="text-link danger" disabled={deleting} onclick={removeArticle}>删除</button>
      {/if}
    </div>
  </article>

  {#if deleteError}<p class="form-error" role="alert">{deleteError}</p>{/if}

  <CommentSection kind="article" id={data.article.id} comments={data.comments} user={data.user} />
{/if}

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

  .article { padding: 0 var(--vx-space-12); }
  .article-title {
    padding: 36px 0 var(--vx-space-6);
    font-size: var(--vx-font-size-headline);
    font-weight: 400;
    line-height: 36px;
  }

  .user-line { display: flex; align-items: center; gap: var(--vx-space-3); padding-bottom: var(--vx-space-4); }
  .user-avatar { display: flex; }
  .user-meta { display: flex; flex-direction: column; gap: 2px; min-width: 0; }
  .user-name { font-size: 17px; line-height: 24px; }
  .user-name:hover { color: var(--vx-color-primary); }
  .user-time { color: var(--vx-color-text-secondary); font-size: var(--vx-font-size-small); }

  .prose { padding: var(--vx-space-6) 0; line-height: 1.75; overflow-wrap: anywhere; }
  .prose :global(h1), .prose :global(h2), .prose :global(h3) { margin: 1.2em 0 0.5em; line-height: 1.35; }
  .prose :global(p) { margin: 0 0 1em; }
  .prose :global(pre) { padding: var(--vx-space-4); overflow-x: auto; background: var(--vx-source-code-surface); border-radius: var(--vx-radius-control); }
  .prose :global(code) { font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace; }
  .prose :global(blockquote) { margin: 1em 0; padding-left: var(--vx-space-4); color: var(--vx-color-text-secondary); border-left: 2px solid var(--vx-color-border); }
  .prose :global(img) { max-width: 100%; height: auto; }

  .topic-bar { display: flex; flex-wrap: wrap; gap: var(--vx-space-2); padding-bottom: var(--vx-space-4); }
  .chip {
    display: inline-flex;
    align-items: center;
    height: 32px;
    padding: 0 var(--vx-space-3);
    color: var(--vx-color-text-secondary);
    font-size: var(--vx-font-size-small);
    background: var(--vx-color-surface-subtle);
    border-radius: var(--vx-radius-pill);
  }
  .chip:hover { color: var(--vx-color-text-primary); }

  .article-actions {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    gap: var(--vx-space-3);
    margin: 0 calc(var(--vx-space-12) * -1);
    padding: var(--vx-space-2) var(--vx-space-12);
    background: var(--vx-color-surface-hover);
  }
  .action-spacer { flex: 1; }
  .text-link { padding: 0; color: var(--vx-color-text-secondary); font-size: var(--vx-font-size-small); background: none; border: 0; cursor: pointer; }
  .text-link:hover { color: var(--vx-color-text-primary); }
  .text-link.danger:hover { color: var(--vx-color-error); }
  .form-error { padding-top: var(--vx-space-2); color: var(--vx-color-error); font-size: var(--vx-font-size-caption); }

  @media (max-width: 599.9px) {
    .article { padding: 0 var(--vx-space-4); border-right: 0; border-left: 0; border-radius: 0; }
    .article-title { padding: var(--vx-space-4) 0; font-size: 22px; line-height: 32px; }
    .article-actions { margin: 0 calc(var(--vx-space-4) * -1); padding: var(--vx-space-2) var(--vx-space-4); }
  }
</style>
