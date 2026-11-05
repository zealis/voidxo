<script lang="ts">
  import { invalidateAll } from '$app/navigation';
  import Button from '#lib/components/ui/Button.svelte';
  import { InboxRequestError, sendMessage, type InboxMessage } from '#lib/api/inbox.js';
  import type { PageProps } from './$types';

  let { data }: PageProps = $props();

  let draft = $state('');
  let sending = $state(false);
  let error = $state('');
  let listElement = $state<HTMLDivElement | null>(null);

  // 接口按新到旧返回，界面需要从旧到新阅读。
  const ordered = $derived([...data.messages].reverse());

  $effect(() => {
    // 依赖消息条数：发送或切换会话后自动滚到底部。
    ordered.length;
    listElement?.scrollTo({ top: listElement.scrollHeight });
  });

  function timeLabel(value: string): string {
    const date = new Date(value);
    const now = new Date();
    const sameDay = date.toDateString() === now.toDateString();
    return sameDay
      ? date.toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit' })
      : date.toLocaleString('zh-CN', { month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit' });
  }

  async function send(event: SubmitEvent) {
    event.preventDefault();
    if (!data.counterpart || sending) return;
    const content = draft.trim();
    if (!content) {
      error = '请先输入内容';
      return;
    }
    if ([...content].length > 1000) {
      error = '私信内容不能超过 1000 个字';
      return;
    }
    sending = true;
    error = '';
    try {
      await sendMessage(data.counterpart.id, content);
      draft = '';
      await invalidateAll();
    } catch (cause) {
      error = cause instanceof InboxRequestError ? cause.message : '发送失败，请稍后重试';
    } finally {
      sending = false;
    }
  }

  function keydown(event: KeyboardEvent) {
    // Enter 发送，Shift+Enter 换行。
    if (event.key === 'Enter' && !event.shiftKey && !event.isComposing) {
      event.preventDefault();
      const form = (event.currentTarget as HTMLTextAreaElement).closest('form');
      form?.requestSubmit();
    }
  }

  function own(message: InboxMessage): boolean {
    return message.sender_id === data.user.id;
  }
</script>

<svelte:head>
  <title>{data.counterpart ? `与 ${data.counterpart.username} 的私信` : '私信'} · VOIDXO</title>
</svelte:head>


  <div class="head">
    <a class="back" href="/messages">← 私信</a>
    <h1>{data.counterpart?.username ?? '会话'}</h1>
  </div>

  {#if data.notice}
    <p class="empty">{data.notice}</p>
  {:else if data.loadError}
    <p class="error" role="alert">会话加载失败，请稍后刷新。</p>
  {:else}
    {#if data.hasMore}
      <p class="more"><a href={`?page=${data.page + 1}`}>查看更早的消息</a></p>
    {/if}

    <div class="list" bind:this={listElement}>
      {#each ordered as message (message.id)}
        <article class="bubble" class:own={own(message)}>
          <div class="content">{@html message.content_rendered}</div>
          <time datetime={message.created_at}>{timeLabel(message.created_at)}</time>
        </article>
      {:else}
        <p class="empty">还没有消息，打个招呼吧。</p>
      {/each}
    </div>

    <form class="composer" onsubmit={send}>
      <textarea
        bind:value={draft}
        rows="3"
        maxlength="1000"
        placeholder="输入私信内容，Enter 发送，Shift+Enter 换行"
        aria-label="私信内容"
        onkeydown={keydown}
      ></textarea>
      <div class="actions">
        <span class="counter">{[...draft].length} / 1000</span>
        <Button type="submit" disabled={sending || !draft.trim()}>发送</Button>
      </div>
    </form>
    {#if error}<p class="error" role="alert">{error}</p>{/if}
  {/if}

<style>
  .head { display: flex; align-items: baseline; gap: var(--vx-space-3); margin-bottom: var(--vx-space-3); }
  .back { color: var(--vx-color-text-secondary); font-size: 13px; text-decoration: none; }
  .back:hover { color: var(--vx-color-text-primary); }
  h1 { font-size: 18px; font-weight: 600; margin: 0; }
  .more { font-size: 13px; text-align: center; margin: 0 0 var(--vx-space-3); }
  .more a { color: var(--vx-color-text-secondary); }
  .list {
    display: flex;
    flex-direction: column;
    gap: var(--vx-space-3);
    max-height: 60vh;
    overflow-y: auto;
    padding: var(--vx-space-3);
    background: var(--vx-color-surface);
    border: 1px solid var(--vx-color-border);
    border-radius: var(--vx-radius-card);
  }
  .bubble {
    max-width: 78%;
    align-self: flex-start;
    padding: 8px 12px;
    border-radius: 12px;
    border: 1px solid var(--vx-color-border);
    background: var(--vx-color-bg);
  }
  .bubble.own { align-self: flex-end; }
  .content { font-size: 14px; line-height: 1.55; word-break: break-word; }
  .content :global(p) { margin: 0; }
  .content :global(p + p) { margin-top: 6px; }
  .content :global(a) { color: inherit; }
  .content :global(code) { font-size: 12px; }
  .bubble time { display: block; margin-top: 4px; color: var(--vx-color-text-secondary); font-size: 11px; text-align: right; }
  .composer {
    margin-top: var(--vx-space-3);
    display: flex;
    flex-direction: column;
    gap: var(--vx-space-2);
    padding: var(--vx-space-3);
    background: var(--vx-color-surface);
    border: 1px solid var(--vx-color-border);
    border-radius: var(--vx-radius-card);
  }
  .composer textarea {
    width: 100%;
    resize: vertical;
    padding: 8px 10px;
    border: 1px solid var(--vx-color-border);
    border-radius: var(--vx-radius-control);
    background: var(--vx-color-bg);
    color: inherit;
    font: inherit;
    font-size: 14px;
  }
  .actions { display: flex; align-items: center; justify-content: space-between; }
  .counter { color: var(--vx-color-text-secondary); font-size: 11px; }
  .empty { color: var(--vx-color-text-secondary); font-size: 13px; text-align: center; padding: var(--vx-space-6) 0; }
  .error { color: var(--vx-color-error); font-size: 13px; }
</style>
