<script lang="ts">
  import { invalidateAll } from '$app/navigation';
  import Button from '#lib/components/ui/Button.svelte';
  import {
    EngagementRequestError,
    deleteNotifications,
    markNotificationsRead,
    type NotificationItem
  } from '#lib/api/engagement.js';
  import type { PageProps } from './$types';

  let { data }: PageProps = $props();
  let pending = $state(false);
  let error = $state('');

  const labels: Record<string, string> = {
    question_answered: '有人回答了你的问题',
    question_commented: '你的问题有新评论',
    question_deleted: '你的问题被删除',
    article_commented: '你的文章有新评论',
    article_deleted: '你的文章被删除',
    answer_commented: '你的回答有新评论',
    answer_deleted: '你的回答被删除',
    comment_replied: '有人回复了你的评论',
    comment_deleted: '你的评论被删除'
  };

  function target(item: NotificationItem): string {
    if (item.article_id) return `/articles/${item.article_id}`;
    if (item.question_id) return `/questions/${item.question_id}`;
    if (item.answer_id) return `/questions/${item.question_id || ''}`;
    return '/notifications';
  }

  async function run(action: () => Promise<unknown>) {
    if (pending) return;
    pending = true;
    error = '';
    try {
      await action();
      await invalidateAll();
    } catch (cause) {
      error = cause instanceof EngagementRequestError ? cause.message : '操作失败，请稍后重试';
    } finally {
      pending = false;
    }
  }

  function markAll() {
    const ids = data.notifications.filter((item) => !item.read).map((item) => item.id);
    if (ids.length === 0) return;
    run(() => markNotificationsRead(ids));
  }

  function removeAll() {
    if (!confirm('确定删除全部通知？')) return;
    const ids = data.notifications.map((item) => item.id);
    if (ids.length === 0) return;
    run(() => deleteNotifications(ids));
  }
</script>

<svelte:head>
  <title>通知 · VOIDXO</title>
</svelte:head>


  <div class="page-head">
    <h1>通知</h1>
    <div class="head-actions">
      <span class="unread">未读 {data.unread}</span>
      <Button variant="outlined" onclick={markAll} disabled={pending || data.unread === 0}>全部已读</Button>
      <Button variant="text" onclick={removeAll} disabled={pending || data.notifications.length === 0}>删除全部</Button>
    </div>
  </div>

  {#if error}<p class="form-error" role="alert">{error}</p>{/if}

  {#if data.notifications.length === 0}
    <p class="empty-text">暂无通知</p>
  {:else}
    <ul class="notification-list">
      {#each data.notifications as item}
        <li class="notification" class:unread={!item.read}>
          <a href={target(item)}>
            <strong>{labels[item.type] ?? item.type}</strong>
            <span class="meta">{item.sender_name || '系统'} · {item.created_at.slice(0, 16).replace('T', ' ')}</span>
          </a>
          {#if !item.read}<span class="dot" aria-label="未读"></span>{/if}
        </li>
      {/each}
    </ul>
  {/if}

<style>
  .page-head { display: flex; align-items: center; justify-content: space-between; gap: var(--vx-space-3); padding: var(--vx-space-4) 0; flex-wrap: wrap; }
  .page-head h1 { font-size: 24px; font-weight: 600; }
  .head-actions { display: flex; align-items: center; gap: var(--vx-space-2); }
  .unread { color: var(--vx-color-text-secondary); font-size: 12px; }
  .notification-list { margin: 0; padding: 0; list-style: none; }
  .notification { display: flex; align-items: center; gap: var(--vx-space-3); padding: var(--vx-space-3) 0; border-bottom: 1px solid var(--vx-color-border); }
  .notification a { display: grid; gap: 2px; flex: 1; }
  .notification strong { font-weight: 500; }
  .notification.unread strong { font-weight: 600; }
  .meta { color: var(--vx-color-text-secondary); font-size: 12px; }
  .dot { width: 8px; height: 8px; background: var(--vx-color-text-primary); border-radius: 50%; }
  .empty-text { padding: var(--vx-space-8) 0; color: var(--vx-color-text-secondary); }
  .form-error { color: var(--vx-color-error); font-size: 12px; }
</style>
