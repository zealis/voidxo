<script lang="ts">
  import Button from '#lib/components/ui/Button.svelte';
  import type { PageProps } from './$types';

  let { data }: PageProps = $props();

  const total = $derived(
    (data.result?.questions.items.length ?? 0) +
      (data.result?.articles.items.length ?? 0) +
      (data.result?.topics.items.length ?? 0)
  );
  const hasMore = $derived(
    !!data.result && (data.result.questions.has_more || data.result.articles.has_more || data.result.topics.has_more)
  );
  const showQuestions = $derived(data.kind === 'all' || data.kind === 'question');
  const showArticles = $derived(data.kind === 'all' || data.kind === 'article');
  const showTopics = $derived(data.kind === 'all' || data.kind === 'topic');
</script>

<svelte:head>
  <title>{data.query ? `${data.query} · 搜索` : '搜索'} · VOIDXO</title>
</svelte:head>


  <h1 class="page-title">搜索</h1>

  <form class="search-form" action="/search" method="GET" role="search">
    <label class="sr-only" for="search-input">搜索关键词</label>
    <input id="search-input" name="q" maxlength="50" value={data.query} placeholder="搜索问题、文章与话题" required />
    <label class="sr-only" for="search-kind">搜索范围</label>
    <select id="search-kind" name="type">
      <option value="all" selected={data.kind === 'all'}>全部</option>
      <option value="question" selected={data.kind === 'question'}>问题</option>
      <option value="article" selected={data.kind === 'article'}>文章</option>
      <option value="topic" selected={data.kind === 'topic'}>话题</option>
    </select>
    <Button type="submit">搜索</Button>
  </form>

  {#if data.errorMessage}
    <p class="form-error" role="alert">{data.errorMessage}</p>
  {:else if !data.query}
    <p class="hint">输入关键词开始搜索，支持问题标题与正文、文章标题与正文、话题名称与描述。</p>
  {:else if total === 0}
    <p class="hint">没有找到与「{data.query}」相关的内容。</p>
  {:else}
    <p class="hint">共找到 {total} 条结果{hasMore ? '（还有更多）' : ''}</p>

    {#if showTopics && data.result && data.result.topics.items.length > 0}
      <section class="group" aria-labelledby="topics-title">
        <h2 id="topics-title">话题</h2>
        <ul>
          {#each data.result.topics.items as topic}
            <li>
              <a href={`/questions?topic_id=${topic.id}`}>
                <strong>{topic.name}</strong>
                <span class="meta">问题 {topic.question_count} · 文章 {topic.article_count} · 关注 {topic.follower_count}</span>
                {#if topic.description}<span class="excerpt">{topic.description}</span>{/if}
              </a>
            </li>
          {/each}
        </ul>
      </section>
    {/if}

    {#if showQuestions && data.result && data.result.questions.items.length > 0}
      <section class="group" aria-labelledby="questions-title">
        <h2 id="questions-title">问题</h2>
        <ul>
          {#each data.result.questions.items as question}
            <li>
              <a href={`/questions/${question.id}`}>
                <strong>{question.title}</strong>
                <span class="meta">{question.username} · 回答 {question.answer_count} · 评论 {question.comment_count}</span>
              </a>
            </li>
          {/each}
        </ul>
      </section>
    {/if}

    {#if showArticles && data.result && data.result.articles.items.length > 0}
      <section class="group" aria-labelledby="articles-title">
        <h2 id="articles-title">文章</h2>
        <ul>
          {#each data.result.articles.items as article}
            <li>
              <a href={`/articles/${article.id}`}>
                <strong>{article.title}</strong>
                <span class="meta">{article.username} · 评论 {article.comment_count} · 票数 {article.vote_count}</span>
              </a>
            </li>
          {/each}
        </ul>
      </section>
    {/if}

    {#if hasMore}
      <p class="more">
        <a href={`/search?q=${encodeURIComponent(data.query)}&type=${data.kind}&page=${data.page + 1}`}>下一页</a>
      </p>
    {/if}
  {/if}

<style>
  .page-title { padding-top: var(--vx-space-4); font-size: 24px; font-weight: 600; }
  .search-form { display: flex; flex-wrap: wrap; gap: var(--vx-space-2); padding: var(--vx-space-4) 0; }
  .search-form input {
    flex: 1 1 240px;
    padding: 8px 12px;
    color: var(--vx-color-text-primary);
    font: inherit;
    background: var(--vx-color-surface);
    border: 1px solid var(--vx-color-border);
    border-radius: var(--vx-radius-control);
  }
  .search-form select {
    padding: 8px 12px;
    color: var(--vx-color-text-primary);
    background: var(--vx-color-surface);
    border: 1px solid var(--vx-color-border);
    border-radius: var(--vx-radius-control);
  }
  .hint { padding: var(--vx-space-3) 0; color: var(--vx-color-text-secondary); font-size: 13px; }
  .group { margin-top: var(--vx-space-4); }
  .group h2 { margin: 0 0 var(--vx-space-2); font-size: 16px; font-weight: 600; }
  .group ul { margin: 0; padding: 0; list-style: none; }
  .group li { border-bottom: 1px solid var(--vx-color-border); }
  .group a { display: grid; gap: 2px; padding: var(--vx-space-3) 0; }
  .group a:hover strong { text-decoration: underline; }
  .group strong { font-weight: 500; }
  .meta, .excerpt { color: var(--vx-color-text-secondary); font-size: 12px; }
  .excerpt { display: -webkit-box; -webkit-line-clamp: 2; line-clamp: 2; -webkit-box-orient: vertical; overflow: hidden; }
  .more { padding: var(--vx-space-4) 0; }
  .more a { text-decoration: underline; }
  .form-error { color: var(--vx-color-error); font-size: 13px; }
  .sr-only { position: absolute; width: 1px; height: 1px; padding: 0; overflow: hidden; clip: rect(0 0 0 0); white-space: nowrap; border: 0; }
</style>
