<script lang="ts">
  import VxChip from '#lib/components/ui/VxChip.svelte';
  import { goto, invalidateAll } from '$app/navigation';
  import VxButton from '#lib/components/ui/VxButton.svelte';
  import Avatar from '#lib/components/Avatar.svelte';
  import Icon from '#lib/components/Icon.svelte';
  import CommentSection from '#lib/components/CommentSection.svelte';
  import FollowButton from '#lib/components/FollowButton.svelte';
  import VoteButtons from '#lib/components/VoteButtons.svelte';
  import {
    createAnswer,
    ContentRequestError,
    deleteAnswer,
    deleteQuestion,
    updateAnswer,
    type Answer
  } from '#lib/api/content.js';
  import type { PageProps } from './$types';

  let { data }: PageProps = $props();
  let answerContent = $state('');
  let answerSaving = $state(false);
  let answerError = $state('');

  let editingAnswerId = $state<number | null>(null);
  let answerDraft = $state('');
  let actionPending = $state(false);
  let actionError = $state('');

  const canManageQuestion = $derived(
    !!data.user &&
      !!data.question &&
      (data.user.permissions.includes('content.manage') ||
        data.user.permissions.includes('content.moderate') ||
        data.user.id === data.question.user_id)
  );

  function canManageAnswer(answer: Answer): boolean {
    if (!data.user) return false;
    return (
      data.user.id === answer.user_id ||
      data.user.permissions.includes('content.moderate') ||
      data.user.permissions.includes('content.manage')
    );
  }

  async function submitAnswer(event: SubmitEvent) {
    event.preventDefault();
    if (answerSaving || !data.question) return;
    answerError = '';
    answerSaving = true;
    try {
      await createAnswer(data.question.id, answerContent);
      answerContent = '';
      await invalidateAll();
    } catch (error) {
      if (error instanceof ContentRequestError && error.status === 401) {
        await goto('/login');
        return;
      }
      answerError = error instanceof ContentRequestError ? error.message : '回答失败，请稍后重试';
    } finally {
      answerSaving = false;
    }
  }

  function startEdit(answer: Answer) {
    actionError = '';
    editingAnswerId = answer.id;
    answerDraft = answer.content_markdown ?? '';
  }

  async function saveAnswer(answer: Answer) {
    if (actionPending) return;
    actionPending = true;
    actionError = '';
    try {
      await updateAnswer(answer.id, answerDraft);
      editingAnswerId = null;
      await invalidateAll();
    } catch (cause) {
      actionError = cause instanceof ContentRequestError ? cause.message : '保存失败';
    } finally {
      actionPending = false;
    }
  }

  async function removeAnswer(answer: Answer) {
    if (actionPending) return;
    if (!confirm('确定删除这条回答？')) return;
    actionPending = true;
    actionError = '';
    try {
      await deleteAnswer(answer.id);
      await invalidateAll();
    } catch (cause) {
      actionError = cause instanceof ContentRequestError ? cause.message : '删除失败';
    } finally {
      actionPending = false;
    }
  }

  async function removeQuestion() {
    if (!data.question || actionPending) return;
    if (!confirm('确定删除这个问题？相关回答会一并删除。')) return;
    actionPending = true;
    actionError = '';
    try {
      await deleteQuestion(data.question.id);
      await goto('/questions');
    } catch (cause) {
      actionError = cause instanceof ContentRequestError ? cause.message : '删除失败';
      await invalidateAll();
    } finally {
      actionPending = false;
    }
  }
</script>

<svelte:head>
  <title>{data.question?.title ?? '问题'} · VOIDXO</title>
  <meta name="description" content={data.question?.title ?? ''} />
</svelte:head>

<nav class="page-nav" aria-label="返回">
  <a class="back-link" href="/questions"><Icon name="arrow-left" size={18} />返回问答</a>
</nav>

{#if data.loadError}
  <div class="error-message" role="alert">
    <span>问题加载失败，请刷新重试。</span>
    <VxButton variant="outlined" onclick={() => location.reload()}>重试</VxButton>
  </div>
{:else if data.question}
  <article class="vx-card vx-card--shadow question" aria-labelledby="question-title">
    <h1 class="question-title" id="question-title">{data.question.title}</h1>

    <div class="user-line">
      <a class="user-avatar" href={`/users/${data.question.user_id}`} aria-label={`${data.question.username} 的主页`}>
        <Avatar name={data.question.username} avatarKey={data.question.avatar_key} size={40} />
      </a>
      <div class="user-meta">
        <a class="user-name" href={`/users/${data.question.user_id}`}>{data.question.username}</a>
        <span class="user-time">{data.question.updated_at.slice(0, 16).replace('T', ' ')}</span>
      </div>
    </div>

    <div class="prose">{@html data.question.content_rendered}</div>

    {#if data.question.topics.length > 0}
      <ul class="topic-bar" aria-label="所属话题">
        {#each data.question.topics as topic (topic.id)}
          <li><VxChip href={`/topics/${topic.id}`}>{topic.name}</VxChip></li>
        {/each}
      </ul>
    {/if}

    <div class="question-actions">
      <VoteButtons kind="question" id={data.question.id} initial={data.vote} canVote={!!data.user} />
      <FollowButton kind="question" id={data.question.id} initial={data.follow} canFollow={!!data.user} />
      <div class="action-spacer"></div>
      {#if canManageQuestion}
        <a class="text-link" href={`/questions/${data.question.id}/edit`}>编辑</a>
        <button type="button" class="text-link danger" disabled={actionPending} onclick={removeQuestion}>删除</button>
      {/if}
    </div>
  </article>

  {#if actionError}<p class="form-error" role="alert">{actionError}</p>{/if}

  <section class="answers" aria-labelledby="answers-title">
    <h2 class="answers-title" id="answers-title">{data.question.answer_count} 个回答</h2>

    {#if data.answers.length === 0}
      <p class="vx-empty">还没有回答，来写下第一个回答。</p>
    {:else}
      <div class="vx-card vx-card--shadow answer-list">
        {#each data.answers as answer (answer.id)}
          <article class="answer" id={`answer-${answer.id}`}>
            <div class="user-line">
              <a class="user-avatar" href={`/users/${answer.user_id}`} aria-label={`${answer.username} 的主页`}>
                <Avatar name={answer.username} avatarKey={answer.avatar_key} size={40} />
              </a>
              <div class="user-meta">
                <a class="user-name" href={`/users/${answer.user_id}`}>{answer.username}</a>
                <span class="user-time">{answer.created_at.slice(0, 16).replace('T', ' ')}</span>
              </div>
            </div>

            {#if editingAnswerId === answer.id}
              <form class="answer-edit" onsubmit={(event) => { event.preventDefault(); saveAnswer(answer); }} novalidate>
                <label class="vx-visually-hidden" for={`answer-edit-${answer.id}`}>编辑回答</label>
                <textarea id={`answer-edit-${answer.id}`} rows="6" maxlength="200000" bind:value={answerDraft} required></textarea>
                <div class="form-actions">
                  <VxButton type="submit" disabled={actionPending}>保存</VxButton>
                  <VxButton type="button" variant="text" onclick={() => (editingAnswerId = null)}>取消</VxButton>
                </div>
              </form>
            {:else}
              <div class="prose answer-prose">{@html answer.content_rendered}</div>
              <div class="answer-actions">
                <VoteButtons
                  kind="answer"
                  id={answer.id}
                  initial={{ vote_count: answer.vote_count, vote_up_count: 0, vote_down_count: 0, my_vote: '' }}
                  canVote={!!data.user}
                />
                {#if canManageAnswer(answer)}
                  <button type="button" class="text-link" onclick={() => startEdit(answer)}>编辑</button>
                  <button type="button" class="text-link danger" disabled={actionPending} onclick={() => removeAnswer(answer)}>删除</button>
                {/if}
              </div>
            {/if}
          </article>
        {/each}
      </div>
    {/if}

    {#if data.user}
      <form class="answer-form vx-card vx-card--shadow" onsubmit={submitAnswer} novalidate>
        <label class="answer-form-label" for="answer-content">写回答</label>
        <textarea id="answer-content" rows="6" maxlength="200000" bind:value={answerContent} required></textarea>
        {#if answerError}<p class="form-error" role="alert">{answerError}</p>{/if}
        <div class="form-actions">
          <VxButton type="submit" disabled={answerSaving}>发布回答</VxButton>
        </div>
      </form>
    {/if}
  </section>

  <CommentSection kind="question" id={data.question.id} comments={data.comments} user={data.user} />
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

  /* 旧站详情卡片：左右 64px 内边距，标题 24/36，操作条通栏浅底。 */
  .question { padding: 0 var(--vx-space-12); }
  .question-title {
    padding: 36px 0 var(--vx-space-6);
    font-size: var(--vx-font-size-headline);
    font-weight: 400;
    line-height: 36px;
  }

  .user-line { position: relative; display: flex; align-items: center; gap: var(--vx-space-3); padding-bottom: var(--vx-space-4); }
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

  .question-actions {
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

  .answers { padding-top: var(--vx-space-6); }
  .answers-title { margin: 0 0 var(--vx-space-3); font-size: 18px; font-weight: 500; }
  .answer-list { overflow: hidden; }
  .answer { padding: var(--vx-space-4) var(--vx-space-12); }
  .answer + .answer { border-top: 1px solid var(--vx-color-border); }
  .answer-prose { padding: var(--vx-space-4) 0 0 64px; }
  .answer-actions { display: flex; flex-wrap: wrap; align-items: center; gap: var(--vx-space-3); padding-left: 64px; }

  .answer-form { display: grid; gap: var(--vx-space-3); margin-top: var(--vx-space-4); padding: var(--vx-space-6) var(--vx-space-12); }
  .answer-form-label { color: var(--vx-color-text-secondary); font-size: var(--vx-font-size-small); }
  textarea {
    width: 100%;
    padding: 10px 12px;
    color: var(--vx-color-text-primary);
    font: inherit;
    background: var(--vx-color-surface);
    border: 1px solid var(--vx-color-border);
    border-radius: var(--vx-radius-control);
    resize: vertical;
  }
  textarea:focus-visible { border-color: var(--vx-color-primary); outline: 2px solid var(--vx-color-primary); outline-offset: 1px; }
  .form-actions { display: flex; align-items: center; gap: var(--vx-space-3); }
  .form-error { color: var(--vx-color-error); font-size: var(--vx-font-size-caption); }
  .answer-edit { display: grid; gap: var(--vx-space-3); padding-top: var(--vx-space-4); padding-left: 64px; }

  @media (max-width: 599.9px) {
    .question { padding: 0 var(--vx-space-4); border-right: 0; border-left: 0; border-radius: 0; }
    .question-title { padding: var(--vx-space-4) 0; font-size: 22px; line-height: 32px; }
    .question-actions { margin: 0 calc(var(--vx-space-4) * -1); padding: var(--vx-space-2) var(--vx-space-4); }
    .answer { padding: var(--vx-space-4); }
    .answer-prose, .answer-actions, .answer-edit { padding-left: 0; }
    .answer-form { padding: var(--vx-space-4); border-right: 0; border-left: 0; border-radius: 0; }
  }
</style>
