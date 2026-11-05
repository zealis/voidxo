<script lang="ts">
  import { goto, invalidateAll } from '$app/navigation';
  import ArticleForm from '#lib/components/ArticleForm.svelte';
  import { updateArticle } from '#lib/api/content.js';
  import type { PageProps } from './$types';

  let { data }: PageProps = $props();

  async function submit(input: { title: string; content_markdown: string; topic_ids: number[] }) {
    await updateArticle(data.article.id, input);
    await invalidateAll();
    await goto(`/articles/${data.article.id}`);
  }
</script>

<svelte:head>
  <title>编辑文章 · VOIDXO</title>
</svelte:head>


  <h1 class="page-title">编辑文章</h1>
  <ArticleForm
    topics={data.topics}
    initialTitle={data.article.title}
    initialMarkdown={data.article.content_markdown ?? ''}
    initialTopicIds={data.article.topics.map((topic) => topic.id)}
    submitLabel="保存修改"
    enableImageUpload
    onSubmit={submit}
  />

<style>
  .page-title { padding-top: var(--vx-space-4); font-size: 24px; font-weight: 600; }
</style>
