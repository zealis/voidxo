<script lang="ts">
  import type { PageProps } from './$types';

  let { data }: PageProps = $props();

  const cards = $derived(
    data.stats
      ? [
          { label: '用户', value: data.stats.users, extra: `停用 ${data.stats.disabled_users}`, href: '/admin/users' },
          { label: '问题', value: data.stats.questions, extra: `7 天新增 ${data.stats.new_questions_7d}`, href: '/admin/questions' },
          { label: '回答', value: data.stats.answers, extra: '', href: '/admin/answers' },
          { label: '文章', value: data.stats.articles, extra: `7 天新增 ${data.stats.new_articles_7d}`, href: '/admin/articles' },
          { label: '评论', value: data.stats.comments, extra: `7 天新增 ${data.stats.new_comments_7d}`, href: '/admin/comments' },
          { label: '话题', value: data.stats.topics, extra: '', href: '/admin/topics' },
          { label: '图片', value: data.stats.images, extra: '', href: '/admin/images' },
          { label: '待处理举报', value: data.stats.pending_reports, extra: '', href: '/admin/reports' }
        ]
      : []
  );
</script>

<svelte:head>
  <title>仪表盘 · 管理后台 · VOIDXO</title>
</svelte:head>

<section class="section">
  <h1>仪表盘</h1>
  {#if data.loadError || !data.stats}
    <p class="error" role="alert">统计数据加载失败，请稍后刷新。</p>
  {:else}
    <div class="grid">
      {#each cards as card (card.label)}
        <a class="card" href={card.href}>
          <span class="label">{card.label}</span>
          <strong class="value">{card.value}</strong>
          {#if card.extra}<span class="extra">{card.extra}</span>{/if}
        </a>
      {/each}
    </div>
    <p class="hint">近 7 天新增用户 {data.stats.new_users_7d} 人。</p>
  {/if}
</section>

<style>
  .section { display: flex; flex-direction: column; gap: var(--vx-space-4); }
  h1 { font-size: 18px; font-weight: 600; margin: 0; }
  .grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(160px, 1fr)); gap: var(--vx-space-3); }
  .card {
    display: flex;
    flex-direction: column;
    gap: 2px;
    padding: var(--vx-space-4);
    background: var(--vx-color-surface);
    border: 1px solid var(--vx-color-border);
    border-radius: var(--vx-radius-card);
    text-decoration: none;
    color: inherit;
  }
  .card:hover { border-color: var(--vx-color-text-secondary); }
  .label { color: var(--vx-color-text-secondary); font-size: 12px; }
  .value { font-size: 24px; font-weight: 600; }
  .extra { color: var(--vx-color-text-secondary); font-size: 12px; }
  .hint { color: var(--vx-color-text-secondary); font-size: 12px; }
  .error { color: var(--vx-color-error); font-size: 13px; }
</style>
