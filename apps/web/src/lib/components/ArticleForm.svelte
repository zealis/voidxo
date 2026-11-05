<script lang="ts">
  import { untrack } from 'svelte';
  import VxButton from '#lib/components/ui/VxButton.svelte';
  import { MediaRequestError, uploadContentImage } from '#lib/api/media.js';
  import type { Topic } from '#lib/api/content.js';

  type Props = {
    topics: Topic[];
    initialTitle?: string;
    initialMarkdown?: string;
    initialTopicIds?: number[];
    submitLabel: string;
    enableImageUpload?: boolean;
    onSubmit: (input: { title: string; content_markdown: string; topic_ids: number[] }) => Promise<void>;
  };

  let {
    topics,
    initialTitle = '',
    initialMarkdown = '',
    initialTopicIds = [],
    submitLabel,
    enableImageUpload = false,
    onSubmit
  }: Props = $props();

  let title = $state(untrack(() => initialTitle));
  let markdown = $state(untrack(() => initialMarkdown));
  let selected = $state<number[]>(untrack(() => initialTopicIds));
  let pending = $state(false);
  let error = $state('');
  let uploading = $state(false);

  function toggleTopic(id: number, checked: boolean) {
    if (checked) {
      if (selected.length >= 10 || selected.includes(id)) return;
      selected = [...selected, id];
    } else {
      selected = selected.filter((item) => item !== id);
    }
  }

  async function insertImage(event: Event) {
    const input = event.currentTarget as HTMLInputElement;
    const file = input.files?.[0];
    if (!file || uploading) return;
    error = '';
    uploading = true;
    try {
      const image = await uploadContentImage(file);
      markdown = `${markdown}${markdown.endsWith('\n') || markdown === '' ? '' : '\n'}![${file.name}](${image.urls.original})\n`;
    } catch (cause) {
      error = cause instanceof MediaRequestError ? cause.message : '图片上传失败';
    } finally {
      uploading = false;
      input.value = '';
    }
  }

  async function submit(event: SubmitEvent) {
    event.preventDefault();
    if (pending) return;
    if (selected.length === 0) {
      error = '请至少选择一个话题';
      return;
    }
    error = '';
    pending = true;
    try {
      await onSubmit({ title, content_markdown: markdown, topic_ids: selected });
    } catch (cause) {
      error = cause instanceof Error ? cause.message : '保存失败，请稍后重试';
    } finally {
      pending = false;
    }
  }
</script>

<form class="article-form" onsubmit={submit} novalidate>
  <div class="field">
    <label for="article-title">标题</label>
    <input id="article-title" maxlength="80" bind:value={title} required placeholder="2 至 80 个字符" />
  </div>

  <fieldset class="topics">
    <legend>话题（1 至 10 个）</legend>
    {#each topics as topic}
      <label class="topic-option">
        <input
          type="checkbox"
          checked={selected.includes(topic.id)}
          onchange={(event) => toggleTopic(topic.id, event.currentTarget.checked)}
        />
        <span>{topic.name}</span>
      </label>
    {/each}
  </fieldset>

  <div class="field">
    <div class="field-head">
      <label for="article-content">正文（Markdown）</label>
      {#if enableImageUpload}
        <label class="image-upload">
          {uploading ? '上传中…' : '插入图片'}
          <input type="file" accept="image/png,image/jpeg,image/gif" disabled={uploading} onchange={insertImage} />
        </label>
      {/if}
    </div>
    <textarea id="article-content" rows="16" maxlength="200000" bind:value={markdown} required></textarea>
  </div>

  {#if error}<p class="form-error" role="alert">{error}</p>{/if}
  <div class="form-actions">
    <VxButton type="submit" disabled={pending}>{submitLabel}</VxButton>
  </div>
</form>

<style>
  .article-form { display: grid; gap: var(--vx-space-4); max-width: 760px; padding-top: var(--vx-space-4); }
  .field { display: grid; gap: var(--vx-space-1); }
  .field label, legend { color: var(--vx-color-text-secondary); font-size: 12px; }
  input:not([type]), textarea {
    width: 100%;
    padding: 8px 12px;
    color: var(--vx-color-text-primary);
    font: inherit;
    background: var(--vx-color-surface);
    border: 1px solid var(--vx-color-border);
    border-radius: var(--vx-radius-control);
  }
  textarea { resize: vertical; font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace; line-height: 1.6; }
  input:focus-visible, textarea:focus-visible { border-color: var(--vx-color-text-primary); outline: 2px solid var(--vx-color-focus); outline-offset: 1px; }
  .topics { display: flex; flex-wrap: wrap; gap: var(--vx-space-2); margin: 0; padding: 0; border: 0; }
  .topics legend { padding: 0; margin-bottom: var(--vx-space-2); }
  .topic-option { display: inline-flex; align-items: center; gap: 6px; padding: 3px 10px; font-size: 12px; border: 1px solid var(--vx-color-border); border-radius: var(--vx-radius-control); }
  .form-actions { display: flex; gap: var(--vx-space-3); }
  .form-error { color: var(--vx-color-error); font-size: 12px; }
  .field-head { display: flex; align-items: center; justify-content: space-between; }
  .image-upload { color: var(--vx-color-text-secondary); font-size: 12px; cursor: pointer; }
  .image-upload input { display: none; }
</style>
