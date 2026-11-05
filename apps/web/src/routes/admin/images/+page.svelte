<script lang="ts">
  import { invalidateAll } from '$app/navigation';
  import AdminPager from '#lib/components/admin/AdminPager.svelte';
  import Button from '#lib/components/ui/Button.svelte';
  import { AdminRequestError, deleteImage, type AdminImage } from '#lib/api/admin.js';
  import type { PageProps } from './$types';

  let { data }: PageProps = $props();

  let pendingKey = $state('');
  let error = $state('');

  const kindLabels: Record<string, string> = {
    content: '内容图',
    avatar: '头像',
    cover: '封面',
    topic_cover: '话题封面'
  };

  const deletedLabels: Record<string, string> = {
    live: '仅正常图片',
    deleted: '仅已删除',
    all: '全部'
  };

  // 每种图片的规格名不同，缩略图取各自的最小档。
  function preview(image: AdminImage): string {
    const variant = image.kind === 'content' ? 'thumb' : 'small';
    return `/media/${image.key}/${variant}`;
  }

  function size(bytes: number): string {
    if (bytes >= 1024 * 1024) return `${(bytes / 1024 / 1024).toFixed(1)} MB`;
    return `${Math.max(1, Math.round(bytes / 1024))} KB`;
  }

  async function remove(image: AdminImage) {
    if (pendingKey) return;
    if (!confirm(`确定删除图片「${image.filename}」？`)) return;
    pendingKey = image.key;
    error = '';
    try {
      await deleteImage(image.key);
      await invalidateAll();
    } catch (cause) {
      error = cause instanceof AdminRequestError ? cause.message : '删除失败，请稍后重试';
    } finally {
      pendingKey = '';
    }
  }
</script>

<svelte:head>
  <title>图片 · 管理后台 · VOIDXO</title>
</svelte:head>

<section class="section">
  <div class="head">
    <h1>图片</h1>
    <form class="filters" method="get">
      <input type="search" name="q" value={data.query} placeholder="文件名或上传者" aria-label="搜索图片" />
      <select name="deleted" aria-label="删除状态">
        {#each Object.entries(deletedLabels) as [value, label] (value)}
          <option {value} selected={data.deleted === value}>{label}</option>
        {/each}
      </select>
      <Button type="submit" variant="outlined">筛选</Button>
    </form>
  </div>

  {#if error}<p class="error" role="alert">{error}</p>{/if}
  {#if data.loadError}<p class="error" role="alert">图片列表加载失败。</p>{/if}

  <div class="grid">
    {#each data.images as image (image.key)}
      <figure class="card" class:removed={image.deleted}>
        <div class="thumb">
          <img src={preview(image)} alt={image.filename} loading="lazy" width="132" height="88" />
        </div>
        <figcaption>
          <div class="name" title={image.filename}>{image.filename}</div>
          <div class="meta">{kindLabels[image.kind] ?? image.kind} · {image.width}×{image.height} · {size(image.size)}</div>
          <div class="meta">{image.username} · {image.created_at.slice(0, 10)}</div>
          <div class="ops">
            {#if image.deleted}
              <span class="meta">已删除</span>
            {:else}
              <Button variant="text" disabled={pendingKey === image.key} onclick={() => remove(image)}>删除</Button>
            {/if}
          </div>
        </figcaption>
      </figure>
    {:else}
      <p class="muted">没有符合条件的图片。</p>
    {/each}
  </div>

  <AdminPager current={data.page} hasMore={data.hasMore} />
</section>

<style>
  .section { display: flex; flex-direction: column; gap: var(--vx-space-3); }
  .head { display: flex; align-items: center; justify-content: space-between; gap: var(--vx-space-3); flex-wrap: wrap; }
  h1 { font-size: 18px; font-weight: 600; margin: 0; }
  .filters { display: flex; gap: var(--vx-space-2); align-items: center; flex-wrap: wrap; }
  .grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(180px, 1fr)); gap: var(--vx-space-3); }
  .card {
    margin: 0;
    background: var(--vx-color-surface);
    border: 1px solid var(--vx-color-border);
    border-radius: var(--vx-radius-card);
    overflow: hidden;
    display: flex;
    flex-direction: column;
  }
  .card.removed { opacity: 0.55; }
  .thumb { aspect-ratio: 3 / 2; background: var(--vx-color-bg); display: flex; align-items: center; justify-content: center; overflow: hidden; }
  .thumb img { width: 100%; height: 100%; object-fit: cover; }
  figcaption { padding: var(--vx-space-2); display: flex; flex-direction: column; gap: 2px; }
  .name { font-size: 12px; font-weight: 500; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
  .meta { color: var(--vx-color-text-secondary); font-size: 11px; }
  .ops { display: flex; justify-content: flex-end; }
  .muted { color: var(--vx-color-text-secondary); font-size: 12px; }
  .error { color: var(--vx-color-error); font-size: 13px; }
  input, select { height: 32px; padding: 0 8px; border: 1px solid var(--vx-color-border); border-radius: var(--vx-radius-control); background: var(--vx-color-surface); color: inherit; }
</style>
