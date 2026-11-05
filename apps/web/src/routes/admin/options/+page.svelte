<script lang="ts">
  import { invalidateAll } from '$app/navigation';
  import VxButton from '#lib/components/ui/VxButton.svelte';
  import { AdminRequestError, updateOptions, type AdminOption } from '#lib/api/admin.js';
  import type { PageProps } from './$types';

  let { data }: PageProps = $props();

  let edits = $state<Record<string, string>>({});
  let pending = $state(false);
  let error = $state('');
  let message = $state('');

  const groupLabels: Record<string, string> = {
    site: '站点',
    question: '问题权限',
    answer: '回答权限',
    article: '文章权限',
    comment: '评论权限'
  };

  // 按配置名前缀分组，未知前缀归入“其他”。
  const groups = $derived.by(() => {
    const map = new Map<string, AdminOption[]>();
    for (const option of data.options) {
      const prefix = option.name.split('_')[0];
      const key = groupLabels[prefix] ? prefix : 'other';
      const list = map.get(key) ?? [];
      list.push(option);
      map.set(key, list);
    }
    return [...map.entries()].sort(([a], [b]) => a.localeCompare(b));
  });

  const changed = $derived(
    Object.entries(edits).filter(([name, value]) => {
      const original = data.options.find((option) => option.name === name);
      return original ? original.value !== value : false;
    })
  );

  function current(option: AdminOption): string {
    return edits[option.name] ?? option.value;
  }

  function setValue(name: string, value: string) {
    edits = { ...edits, [name]: value };
  }

  function label(name: string): string {
    return name;
  }

  async function save() {
    if (pending || changed.length === 0) return;
    pending = true;
    error = '';
    message = '';
    try {
      await updateOptions(Object.fromEntries(changed));
      edits = {};
      message = `已保存 ${changed.length} 项配置`;
      await invalidateAll();
    } catch (cause) {
      error = cause instanceof AdminRequestError ? cause.message : '保存失败，请稍后重试';
    } finally {
      pending = false;
    }
  }
</script>

<svelte:head>
  <title>配置 · 管理后台 · VOIDXO</title>
</svelte:head>

<section class="section">
  <div class="head">
    <h1>配置</h1>
    <div class="actions">
      {#if changed.length > 0}<span class="muted">待保存 {changed.length} 项</span>{/if}
      <VxButton disabled={pending || changed.length === 0} onclick={save}>保存修改</VxButton>
    </div>
  </div>

  {#if error}<p class="error" role="alert">{error}</p>{/if}
  {#if message}<p class="success" role="status">{message}</p>{/if}
  {#if data.loadError}<p class="error" role="alert">配置加载失败。</p>{/if}

  {#each groups as [prefix, options] (prefix)}
    <section class="group">
      <h2>{groupLabels[prefix] ?? '其他'}</h2>
      <div class="rows">
        {#each options as option (option.name)}
          <label class="row">
            <span class="name">{label(option.name)}</span>
            {#if option.type === 'bool'}
              <select value={current(option)} onchange={(event) => setValue(option.name, event.currentTarget.value)}>
                <option value="true">开启</option>
                <option value="false">关闭</option>
              </select>
            {:else if option.type === 'int'}
              <input
                type="number"
                min="0"
                value={current(option)}
                oninput={(event) => setValue(option.name, event.currentTarget.value)}
              />
            {:else}
              <input
                type="text"
                value={current(option)}
                oninput={(event) => setValue(option.name, event.currentTarget.value)}
              />
            {/if}
          </label>
        {/each}
      </div>
    </section>
  {/each}
</section>

<style>
  .section { display: flex; flex-direction: column; gap: var(--vx-space-4); }
  .head { display: flex; align-items: center; justify-content: space-between; gap: var(--vx-space-3); }
  .actions { display: flex; align-items: center; gap: var(--vx-space-2); }
  h1 { font-size: 18px; font-weight: 600; margin: 0; }
  h2 { font-size: 13px; font-weight: 600; margin: 0 0 var(--vx-space-2); color: var(--vx-color-text-secondary); }
  .group { background: var(--vx-color-surface); border: 1px solid var(--vx-color-border); border-radius: var(--vx-radius-card); padding: var(--vx-space-3); }
  .rows { display: flex; flex-direction: column; gap: var(--vx-space-2); }
  .row { display: grid; grid-template-columns: minmax(180px, 260px) minmax(120px, 260px); align-items: center; gap: var(--vx-space-3); font-size: 13px; }
  .name { color: var(--vx-color-text-secondary); font-family: ui-monospace, SFMono-Regular, Menlo, monospace; font-size: 12px; }
  .muted { color: var(--vx-color-text-secondary); font-size: 12px; }
  .error { color: var(--vx-color-error); font-size: 13px; }
  .success { color: var(--vx-color-text-secondary); font-size: 13px; }
  input, select { height: 32px; padding: 0 8px; border: 1px solid var(--vx-color-border); border-radius: var(--vx-radius-control); background: var(--vx-color-surface); color: inherit; }
</style>
