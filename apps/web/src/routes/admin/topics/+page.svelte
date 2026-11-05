<script lang="ts">
  import { invalidateAll } from '$app/navigation';
  import AdminPager from '#lib/components/admin/AdminPager.svelte';
  import VxButton from '#lib/components/ui/VxButton.svelte';
  import {
    AdminRequestError,
    createTopic,
    deleteTopic,
    updateTopic,
    type AdminTopic
  } from '#lib/api/admin.js';
  import type { PageProps } from './$types';

  let { data }: PageProps = $props();

  let name = $state('');
  let description = $state('');
  let pending = $state(false);
  let error = $state('');
  let message = $state('');
  let editingId = $state(0);
  let editName = $state('');
  let editDescription = $state('');

  async function run(action: () => Promise<unknown>, success: string) {
    if (pending) return;
    pending = true;
    error = '';
    message = '';
    try {
      await action();
      message = success;
      await invalidateAll();
    } catch (cause) {
      error = cause instanceof AdminRequestError ? cause.message : '操作失败，请稍后重试';
    } finally {
      pending = false;
    }
  }

  function submit(event: SubmitEvent) {
    event.preventDefault();
    run(async () => {
      await createTopic({ name, description });
      name = '';
      description = '';
    }, '话题已创建');
  }

  function startEdit(topic: AdminTopic) {
    editingId = topic.id;
    editName = topic.name;
    editDescription = topic.description;
  }

  function cancelEdit() {
    editingId = 0;
  }

  function saveEdit(topic: AdminTopic) {
    run(async () => {
      await updateTopic(topic.id, { name: editName, description: editDescription });
      editingId = 0;
    }, '话题已更新');
  }

  function remove(topic: AdminTopic) {
    if (!confirm(`确定删除话题「${topic.name}」？`)) return;
    run(() => deleteTopic(topic.id), '话题已删除');
  }
</script>

<svelte:head>
  <title>话题 · 管理后台 · VOIDXO</title>
</svelte:head>

<section class="section">
  <div class="head">
    <h1>话题</h1>
    <form class="filters" method="get">
      <input type="search" name="q" value={data.query} placeholder="话题名称或描述" aria-label="搜索话题" />
      <label class="checkbox">
        <input type="checkbox" name="include_deleted" value="1" checked={data.includeDeleted} />
        含已删除
      </label>
      <VxButton type="submit" variant="outlined">筛选</VxButton>
    </form>
  </div>

  {#if error}<p class="error" role="alert">{error}</p>{/if}
  {#if message}<p class="success" role="status">{message}</p>{/if}
  {#if data.loadError}<p class="error" role="alert">话题列表加载失败。</p>{/if}

  <form class="create" onsubmit={submit}>
    <h2>新建话题</h2>
    <input type="text" bind:value={name} maxlength="20" placeholder="话题名称（1–20 字）" required aria-label="话题名称" />
    <textarea bind:value={description} maxlength="1000" rows="2" placeholder="话题描述（可选）" aria-label="话题描述"></textarea>
    <VxButton type="submit" disabled={pending}>创建</VxButton>
  </form>

  <div class="table-wrap">
    <table>
      <thead>
        <tr>
          <th>话题</th>
          <th>内容量</th>
          <th>状态</th>
          <th>创建时间</th>
          <th>操作</th>
        </tr>
      </thead>
      <tbody>
        {#each data.topics as topic (topic.id)}
          <tr class:removed={topic.deleted}>
            <td>
              {#if editingId === topic.id}
                <input type="text" bind:value={editName} maxlength="20" aria-label="编辑话题名称" />
                <textarea bind:value={editDescription} maxlength="1000" rows="2" aria-label="编辑话题描述"></textarea>
              {:else}
                <strong>{topic.name}</strong>
                {#if topic.description}<div class="muted">{topic.description}</div>{/if}
              {/if}
            </td>
            <td class="muted">问题 {topic.question_count} · 文章 {topic.article_count} · 关注 {topic.follower_count}</td>
            <td>
              {#if topic.deleted}
                <span class="badge badge-off">已删除</span>
              {:else}
                <span class="badge">正常</span>
              {/if}
            </td>
            <td class="muted">{topic.created_at.slice(0, 10)}</td>
            <td class="actions">
              {#if topic.deleted}
                <span class="muted">—</span>
              {:else if editingId === topic.id}
                <VxButton variant="text" disabled={pending} onclick={() => saveEdit(topic)}>保存</VxButton>
                <VxButton variant="text" disabled={pending} onclick={cancelEdit}>取消</VxButton>
              {:else}
                <VxButton variant="text" disabled={pending} onclick={() => startEdit(topic)}>编辑</VxButton>
                <VxButton variant="text" disabled={pending} onclick={() => remove(topic)}>删除</VxButton>
              {/if}
            </td>
          </tr>
        {:else}
          <tr><td colspan="5" class="muted">还没有话题。</td></tr>
        {/each}
      </tbody>
    </table>
  </div>

  <AdminPager current={data.page} hasMore={data.hasMore} />
</section>

<style>
  .section { display: flex; flex-direction: column; gap: var(--vx-space-3); }
  .head { display: flex; align-items: center; justify-content: space-between; gap: var(--vx-space-3); flex-wrap: wrap; }
  h1 { font-size: 18px; font-weight: 600; margin: 0; }
  h2 { font-size: 13px; font-weight: 600; margin: 0 0 var(--vx-space-2); color: var(--vx-color-text-secondary); }
  .filters { display: flex; gap: var(--vx-space-2); align-items: center; flex-wrap: wrap; }
  .checkbox { display: inline-flex; align-items: center; gap: 4px; font-size: 12px; color: var(--vx-color-text-secondary); }
  .create {
    display: flex;
    flex-direction: column;
    gap: var(--vx-space-2);
    padding: var(--vx-space-3);
    background: var(--vx-color-surface);
    border: 1px solid var(--vx-color-border);
    border-radius: var(--vx-radius-card);
    max-width: 420px;
  }
  .create :global(button) { align-self: flex-start; }
  .table-wrap { overflow-x: auto; background: var(--vx-color-surface); border: 1px solid var(--vx-color-border); border-radius: var(--vx-radius-card); }
  table { width: 100%; border-collapse: collapse; font-size: 13px; }
  th, td { padding: 10px 12px; text-align: left; vertical-align: top; }
  th { color: var(--vx-color-text-secondary); font-weight: 500; font-size: 12px; border-bottom: 1px solid var(--vx-color-border); }
  tbody tr + tr td { border-top: 1px solid var(--vx-color-border); }
  tr.removed td { opacity: 0.55; }
  .muted { color: var(--vx-color-text-secondary); font-size: 12px; }
  .error { color: var(--vx-color-error); font-size: 13px; }
  .success { color: var(--vx-color-text-secondary); font-size: 13px; }
  .badge { font-size: 12px; padding: 1px 8px; border-radius: 999px; border: 1px solid var(--vx-color-border); }
  .badge-off { color: var(--vx-color-error); border-color: var(--vx-color-error); }
  .actions { display: flex; gap: var(--vx-space-1); }
  input, textarea { padding: 6px 8px; border: 1px solid var(--vx-color-border); border-radius: var(--vx-radius-control); background: var(--vx-color-surface); color: inherit; font: inherit; font-size: 13px; }
</style>
