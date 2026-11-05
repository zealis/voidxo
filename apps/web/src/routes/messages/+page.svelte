<script lang="ts">
  import { goto } from '$app/navigation';
  import Avatar from '#lib/components/Avatar.svelte';
  import VxButton from '#lib/components/ui/VxButton.svelte';
  import { findRecipient, InboxRequestError } from '#lib/api/inbox.js';
  import type { PageProps } from './$types';

  let { data }: PageProps = $props();

  let username = $state('');
  let pending = $state(false);
  let error = $state('');

  function timeLabel(value: string): string {
    const date = new Date(value);
    const now = new Date();
    const sameDay = date.toDateString() === now.toDateString();
    return sameDay
      ? date.toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit' })
      : date.toLocaleDateString('zh-CN');
  }

  async function startConversation(event: SubmitEvent) {
    event.preventDefault();
    const target = username.trim();
    if (!target || pending) return;
    pending = true;
    error = '';
    try {
      const recipient = await findRecipient(target);
      await goto(`/messages/${recipient.id}`);
    } catch (cause) {
      error = cause instanceof InboxRequestError ? cause.message : '无法找到该用户';
    } finally {
      pending = false;
    }
  }
</script>

<svelte:head>
  <title>私信 · VOIDXO</title>
</svelte:head>


  <div class="head">
    <h1>私信</h1>
    {#if data.unread > 0}<span class="unread-total">{data.unread} 条未读</span>{/if}
  </div>

  <form class="starter" onsubmit={startConversation}>
    <label for="inbox-username">发起私信</label>
    <input
      id="inbox-username"
      type="text"
      bind:value={username}
      maxlength="20"
      placeholder="输入对方用户名"
      autocomplete="off"
      required
    />
    <VxButton type="submit" variant="outlined" disabled={pending}>打开会话</VxButton>
  </form>
  {#if error}<p class="error" role="alert">{error}</p>{/if}
  {#if data.loadError}<p class="error" role="alert">会话列表加载失败，请稍后刷新。</p>{/if}

  {#if data.conversations.length === 0}
    <p class="empty">还没有私信。输入用户名即可开始一段对话。</p>
  {:else}
    <ul class="conversations">
      {#each data.conversations as item (item.user_id)}
        <li>
          <a class="conversation" href={`/messages/${item.user_id}`}>
            <Avatar name={item.username} avatarKey={item.avatar_key} size={40} variant="small" />
            <span class="body">
              <span class="line">
                <strong>{item.username}</strong>
                <time datetime={item.last_at}>{timeLabel(item.last_at)}</time>
              </span>
              <span class="excerpt">
                {#if item.last_sender_id === data.user.id}<span class="muted">我：</span>{/if}
                {item.last_excerpt}
              </span>
            </span>
            {#if item.unread > 0}<span class="badge">{item.unread}</span>{/if}
          </a>
        </li>
      {/each}
    </ul>
  {/if}

  {#if data.page > 1 || data.hasMore}
    <nav class="pager" aria-label="分页">
      {#if data.page > 1}
        <a href={`?page=${data.page - 1}`}>上一页</a>
      {:else}
        <span class="muted">上一页</span>
      {/if}
      <span class="muted">第 {data.page} 页</span>
      {#if data.hasMore}
        <a href={`?page=${data.page + 1}`}>下一页</a>
      {:else}
        <span class="muted">下一页</span>
      {/if}
    </nav>
  {/if}

<style>
  .head { display: flex; align-items: baseline; gap: var(--vx-space-3); }
  h1 { font-size: 20px; font-weight: 600; margin: 0; }
  .unread-total { color: var(--vx-color-text-secondary); font-size: 12px; }
  .starter {
    display: flex;
    align-items: center;
    gap: var(--vx-space-2);
    margin: var(--vx-space-4) 0;
    flex-wrap: wrap;
  }
  .starter label { color: var(--vx-color-text-secondary); font-size: 13px; }
  .starter input {
    flex: 1;
    min-width: 180px;
    height: 36px;
    padding: 0 10px;
    border: 1px solid var(--vx-color-border);
    border-radius: var(--vx-radius-control);
    background: var(--vx-color-surface);
    color: inherit;
  }
  .conversations { list-style: none; margin: 0; padding: 0; display: flex; flex-direction: column; gap: 1px; background: var(--vx-color-border); border: 1px solid var(--vx-color-border); border-radius: var(--vx-radius-card); overflow: hidden; }
  .conversation {
    display: flex;
    align-items: center;
    gap: var(--vx-space-3);
    padding: var(--vx-space-3);
    background: var(--vx-color-surface);
    text-decoration: none;
    color: inherit;
  }
  .conversation:hover { background: var(--vx-color-bg); }
  .body { flex: 1; min-width: 0; display: flex; flex-direction: column; gap: 2px; }
  .line { display: flex; align-items: baseline; justify-content: space-between; gap: var(--vx-space-2); }
  .line strong { font-size: 14px; }
  .line time { color: var(--vx-color-text-secondary); font-size: 12px; }
  .excerpt { color: var(--vx-color-text-secondary); font-size: 13px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
  .badge {
    flex: none;
    min-width: 20px;
    height: 20px;
    padding: 0 6px;
    border-radius: 10px;
    background: var(--vx-color-text-primary);
    color: var(--vx-color-surface);
    font-size: 11px;
    display: inline-flex;
    align-items: center;
    justify-content: center;
  }
  .muted { color: var(--vx-color-text-secondary); }
  .empty { color: var(--vx-color-text-secondary); font-size: 13px; padding: var(--vx-space-6) 0; }
  .error { color: var(--vx-color-error); font-size: 13px; }
  .pager { display: flex; align-items: center; gap: var(--vx-space-3); margin-top: var(--vx-space-4); font-size: 13px; }
  .pager a { color: var(--vx-color-text-primary); }
</style>
