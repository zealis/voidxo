<script lang="ts">
  import { goto, invalidateAll } from '$app/navigation';
  import ArticleForm from '#lib/components/ArticleForm.svelte';
  import { updateQuestion } from '#lib/api/content.js';
  import type { PageProps } from './$types';

  let { data }: PageProps = $props();

  async function submit(input: { title: string; content_markdown: string; topic_ids: number[] }) {
    await updateQuestion(data.question.id, input);
    await invalidateAll();
    await goto(`/questions/${data.question.id}`);
  }
</script>

<svelte:head>
  <title>编辑问题 · VOIDXO</title>
</svelte:head>


  <h1 class="page-title">编辑问题</h1>
  <ArticleForm
    topics={data.topics}
    initialTitle={data.question.title}
    initialMarkdown={data.question.content_markdown ?? ''}
    initialTopicIds={data.question.topics.map((topic) => topic.id)}
    submitLabel="保存修改"
    onSubmit={submit}
  />

<style>
  .page-title { padding-top: var(--vx-space-4); font-size: 24px; font-weight: 600; }
</style>
