<script lang="ts">
  import { goto, invalidateAll } from '$app/navigation';
  import VxButton from '#lib/components/ui/VxButton.svelte';
  import type { AuthUser } from '#lib/api/auth.js';
  import { ContentRequestError, createComment, deleteComment, type CommentItem } from '#lib/api/content.js';

  type Props = {
    kind: string;
    id: number;
    comments: CommentItem[];
    user: AuthUser | null;
  };

  let { kind, id, comments, user }: Props = $props();
  let content = $state('');
  let replyTo = $state<CommentItem | null>(null);
  let replyContent = $state('');
  let pending = $state(false);
  let error = $state('');

  const roots = $derived(comments.filter((item) => !item.root_id));
  const replies = $derived((rootID: number) => comments.filter((item) => item.root_id === rootID));

  function canDelete(item: CommentItem): boolean {
    if (!user) return false;
    return user.id === item.user_id || user.permissions.includes('content.moderate') || user.permissions.includes('content.manage');
  }

  async function submit(event: SubmitEvent, rootID?: number) {
    event.preventDefault();
    if (pending) return;
    const text = rootID ? replyContent : content;
    if (!text.trim()) return;
    error = '';
    pending = true;
    try {
      await createComment({ commentable_type: kind, commentable_id: id, content: text, root_id: rootID });
      if (rootID) {
        replyContent = '';
        replyTo = null;
      } else {
        content = '';
      }
      await invalidateAll();
    } catch (cause) {
      if (cause instanceof ContentRequestError && cause.status === 401) {
        await goto('/login');
        return;
      }
      error = cause instanceof ContentRequestError ? cause.message : '评论失败，请稍后重试';
    } finally {
      pending = false;
    }
  }

  async function remove(item: CommentItem) {
    if (pending) return;
    if (!confirm('确定删除这条评论？相关回复会一并删除。')) return;
    error = '';
    pending = true;
    try {
      await deleteComment(item.id);
      await invalidateAll();
    } catch (cause) {
      error = cause instanceof ContentRequestError ? cause.message : '删除失败';
    } finally {
      pending = false;
    }
  }
</script>

<section class="comments" aria-labelledby="comments-title">
  <h2 id="comments-title">{comments.length} 条评论</h2>
  {#if error}<p class="form-error" role="alert">{error}</p>{/if}

  {#if roots.length === 0}
    <p class="empty-text">暂无评论</p>
  {:else}
    <ul class="comment-list">
      {#each roots as item}
        <li class="comment">
          <header class="meta">
            <a href={`/users/${item.user_id}`}>{item.username}</a>
            <time datetime={item.created_at}>{item.created_at.slice(0, 10)}</time>
          </header>
          <p class="comment-content">{item.content}</p>
          <div class="comment-actions">
            {#if user}
              <button type="button" class="link" onclick={() => (replyTo = replyTo?.id === item.id ? null : item)}>回复</button>
            {/if}
            {#if canDelete(item)}
              <button type="button" class="link danger" disabled={pending} onclick={() => remove(item)}>删除</button>
            {/if}
          </div>

          {#if replies(item.id).length > 0}
            <ul class="reply-list">
              {#each replies(item.id) as reply}
                <li class="reply">
                  <header class="meta">
                    <a href={`/users/${reply.user_id}`}>{reply.username}</a>
                    <time datetime={reply.created_at}>{reply.created_at.slice(0, 10)}</time>
                  </header>
                  <p class="comment-content">{reply.content}</p>
                  {#if canDelete(reply)}
                    <button type="button" class="link danger" disabled={pending} onclick={() => remove(reply)}>删除</button>
                  {/if}
                </li>
              {/each}
            </ul>
          {/if}

          {#if replyTo?.id === item.id}
            <form class="reply-form" onsubmit={(event) => submit(event, item.id)} novalidate>
              <label class="sr-only" for={`reply-${item.id}`}>回复 {item.username}</label>
              <input id={`reply-${item.id}`} maxlength="1000" bind:value={replyContent} required placeholder={`回复 ${item.username}`} />
              <VxButton type="submit" variant="outlined" disabled={pending}>发送</VxButton>
            </form>
          {/if}
        </li>
      {/each}
    </ul>
  {/if}

  {#if user}
    <form class="comment-form" onsubmit={(event) => submit(event)} novalidate>
      <label for="comment-content">写评论</label>
      <textarea id="comment-content" rows="3" maxlength="1000" bind:value={content} required></textarea>
      <div class="form-actions">
        <VxButton type="submit" disabled={pending}>发表评论</VxButton>
      </div>
    </form>
  {:else}
    <p class="empty-text"><a class="login-link" href="/login">登录</a>后可以评论</p>
  {/if}
</section>

<style>
  .comments { max-width: 760px; margin-top: var(--vx-space-6); padding-top: var(--vx-space-6); border-top: 1px solid var(--vx-color-border); }
  .comments h2 { margin: 0 0 var(--vx-space-4); font-size: 18px; font-weight: 500; }
  .comment-list, .reply-list { margin: 0; padding: 0; list-style: none; }
  .comment { padding: var(--vx-space-3) 0; border-bottom: 1px solid var(--vx-color-border); }
  .reply-list { margin-top: var(--vx-space-3); padding-left: var(--vx-space-4); border-left: 2px solid var(--vx-color-border); }
  .reply { padding: var(--vx-space-2) 0; }
  .meta { display: flex; gap: var(--vx-space-2); color: var(--vx-color-text-secondary); font-size: 12px; }
  .comment-content { margin: var(--vx-space-1) 0; line-height: 1.7; white-space: pre-wrap; overflow-wrap: anywhere; }
  .comment-actions { display: flex; gap: var(--vx-space-3); }
  .link { padding: 0; color: var(--vx-color-text-secondary); font-size: 12px; background: none; border: 0; cursor: pointer; }
  .link:hover { color: var(--vx-color-text-primary); }
  .link.danger:hover { color: var(--vx-color-error); }
  .reply-form { display: flex; gap: var(--vx-space-2); margin-top: var(--vx-space-2); }
  .reply-form input {
    flex: 1;
    padding: 6px 10px;
    color: var(--vx-color-text-primary);
    background: var(--vx-color-surface);
    border: 1px solid var(--vx-color-border);
    border-radius: var(--vx-radius-control);
  }
  .comment-form { display: grid; gap: var(--vx-space-2); margin-top: var(--vx-space-4); }
  .comment-form label { color: var(--vx-color-text-secondary); font-size: 12px; }
  .comment-form textarea {
    width: 100%;
    padding: 8px 12px;
    color: var(--vx-color-text-primary);
    font: inherit;
    background: var(--vx-color-surface);
    border: 1px solid var(--vx-color-border);
    border-radius: var(--vx-radius-control);
    resize: vertical;
  }
  .form-actions { display: flex; gap: var(--vx-space-3); }
  .empty-text { color: var(--vx-color-text-secondary); }
  .login-link { text-decoration: underline; }
  .form-error { color: var(--vx-color-error); font-size: 12px; }
  .sr-only { position: absolute; width: 1px; height: 1px; padding: 0; overflow: hidden; clip: rect(0 0 0 0); white-space: nowrap; border: 0; }
  .meta a { color: inherit; }
</style>
