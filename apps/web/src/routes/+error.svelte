<script lang="ts">
  import { page } from '$app/state';

  const status = $derived(page.status);
  const message = $derived(
    page.error?.message && page.error.message !== 'Not Found'
      ? page.error.message
      : status === 404
        ? '你访问的页面不存在或已被删除。'
        : '页面加载失败，请稍后重试。'
  );
</script>

<svelte:head>
  <title>{status} · VOIDXO</title>
  <meta name="robots" content="noindex, nofollow" />
</svelte:head>

<section class="error-page">
  <p class="status">{status}</p>
  <h1>{status === 404 ? '页面走丢了' : '出了点问题'}</h1>
  <p class="message">{message}</p>
  <div class="actions">
    <a class="primary" href="/">回到首页</a>
    <button type="button" onclick={() => history.back()}>返回上一页</button>
  </div>
</section>

<style>
  .error-page {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: var(--vx-space-3);
    padding: var(--vx-space-12) var(--vx-space-6);
    margin-top: var(--vx-space-6);
    text-align: center;
    background: var(--vx-color-surface);
    border: 1px solid var(--vx-color-border);
    border-radius: var(--vx-radius-card);
  }
  .status { color: var(--vx-color-text-tertiary); font-size: 56px; font-weight: 300; line-height: 1; }
  h1 { font-size: var(--vx-font-size-title); font-weight: 500; }
  .message { color: var(--vx-color-text-secondary); font-size: var(--vx-font-size-body); }
  .actions { display: flex; gap: var(--vx-space-3); margin-top: var(--vx-space-4); }
  .actions a,
  .actions button {
    padding: var(--vx-space-2) var(--vx-space-4);
    font-size: var(--vx-font-size-small);
    border-radius: var(--vx-radius-control);
    cursor: pointer;
  }
  .primary { color: #fff; background: var(--vx-color-primary); border: 1px solid var(--vx-color-primary); }
  .primary:hover { opacity: 0.92; }
  .actions button { color: var(--vx-color-text-secondary); background: transparent; border: 1px solid var(--vx-color-border); }
  .actions button:hover { color: var(--vx-color-text-primary); background: var(--vx-color-surface-hover); }
</style>
