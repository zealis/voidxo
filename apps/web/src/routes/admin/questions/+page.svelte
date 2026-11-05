<script lang="ts">
  import AdminContentManager from '#lib/components/admin/AdminContentManager.svelte';
  import type { PageProps } from './$types';

  let { data }: PageProps = $props();
</script>

<svelte:head>
  <title>问题 · 管理后台 · VOIDXO</title>
</svelte:head>

<AdminContentManager
  kind="question"
  label="问题"
  items={data.items}
  page={data.page}
  hasMore={data.hasMore}
  query={data.query}
  deleted={data.deleted}
  loadError={data.loadError}
/>
