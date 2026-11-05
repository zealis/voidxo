<script lang="ts">
  import { goto, invalidateAll } from '$app/navigation';
  import ArticleForm from '#lib/components/ArticleForm.svelte';
  import { createArticle } from '#lib/api/content.js';
  import type { PageProps } from './$types';

  let { data }: PageProps = $props();

  async function submit(input: { title: string; content_markdown: string; topic_ids: number[] }) {
    const article = await createArticle(input);
    await invalidateAll();
    await goto(`/articles/${article.id}`);
  }
</script>

<svelte:head>
  <title>写文章 · VOIDXO</title>
</svelte:head>


  <h1 class="page-title">写文章</h1>
  <ArticleForm topics={data.topics} submitLabel="发布文章" enableImageUpload onSubmit={submit} />

<style>
  .page-title { padding-top: var(--vx-space-4); font-size: 24px; font-weight: 600; }
</style>
