<script lang="ts">
  import { goto } from '$app/navigation';
  import VxButton from '#lib/components/ui/VxButton.svelte';
  import { createQuestion, ContentRequestError } from '#lib/api/content.js';
  import type { PageProps } from './$types';

  let { data }: PageProps = $props();
  let title = $state('');
  let content = $state('');
  let topicIDs = $state<number[]>([]);
  let saving = $state(false);
  let errorMessage = $state('');

  async function submit(event: SubmitEvent) {
    event.preventDefault();
    if (saving) return;
    errorMessage = '';
    if (topicIDs.length === 0) {
      errorMessage = '请选择至少一个话题';
      return;
    }
    saving = true;
    try {
      const question = await createQuestion({ title, content_markdown: content, topic_ids: topicIDs });
      await goto(`/questions/${question.id}`);
    } catch (error) {
      errorMessage = error instanceof ContentRequestError ? error.message : '发布失败，请稍后重试';
    } finally {
      saving = false;
    }
  }
</script>

<svelte:head>
  <title>提问 · VOIDXO</title>
</svelte:head>

<header class="app-bar">
  <a class="brand" href="/" aria-label="VOIDXO 首页">VOIDXO</a>
  <nav aria-label="导航">
    <a class="nav-link" href="/">返回首页</a>
  </nav>
</header>

<main class="page-container">
  <section class="editor-panel" aria-labelledby="ask-title">
    <h1 id="ask-title">提问</h1>
    {#if data.loadError}
      <div class="error-message" role="alert">
        <span>编辑环境加载失败，请刷新重试。</span>
        <VxButton variant="outlined" onclick={() => location.reload()}>重试</VxButton>
      </div>
    {:else}
      <form onsubmit={submit} novalidate>
        <div class="field">
          <label for="title">标题</label>
          <input id="title" type="text" maxlength="80" bind:value={title} required />
        </div>
        <div class="field">
          <label for="content">正文</label>
          <textarea id="content" rows="12" maxlength="200000" bind:value={content} required></textarea>
        </div>
        <fieldset class="topics-field">
          <legend>话题</legend>
          {#if data.topics.length === 0}
            <p class="empty-text">暂无可用话题</p>
          {:else}
            <div class="topic-options">
              {#each data.topics as topic}
                <label class="topic-option">
                  <input type="checkbox" value={topic.id} bind:group={topicIDs} />
                  <span>{topic.name}</span>
                </label>
              {/each}
            </div>
          {/if}
        </fieldset>
        {#if errorMessage}<p class="form-error" role="alert">{errorMessage}</p>{/if}
        <div class="form-actions">
          <VxButton type="submit" disabled={saving || data.topics.length === 0}>发布</VxButton>
        </div>
      </form>
    {/if}
  </section>
</main>

<style>
  .editor-panel { max-width: 760px; padding: var(--vx-space-8) 0; }
  h1 { margin: 0 0 var(--vx-space-6); font-size: 24px; font-weight: 500; }
  form { display: grid; gap: var(--vx-space-4); }
  .field { display: grid; gap: var(--vx-space-1); }
  label, legend { color: var(--vx-color-text-secondary); font-size: 12px; }
  input[type='text'], textarea {
    width: 100%;
    padding: 8px 12px;
    color: var(--vx-color-text-primary);
    font: inherit;
    background: var(--vx-color-surface);
    border: 1px solid var(--vx-color-border);
    border-radius: var(--vx-radius-control);
    resize: vertical;
  }
  input[type='text'] { min-height: var(--vx-input-height); }
  input:focus-visible, textarea:focus-visible {
    border-color: var(--vx-color-text-primary);
    outline: 2px solid var(--vx-color-focus);
    outline-offset: 1px;
  }
  .topics-field { padding: 0; border: 0; }
  .topic-options { display: flex; flex-wrap: wrap; gap: var(--vx-space-3); margin-top: var(--vx-space-2); }
  .topic-option { display: flex; align-items: center; gap: var(--vx-space-1); color: var(--vx-color-text-primary); font-size: 14px; }
  .form-actions { display: flex; align-items: center; gap: var(--vx-space-3); }
  .form-error { color: var(--vx-color-error); font-size: 12px; }
  .empty-text { color: var(--vx-color-text-secondary); }
</style>
