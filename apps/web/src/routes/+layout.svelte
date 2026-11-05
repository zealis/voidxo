<script lang="ts">
  import '#lib/styles/global.css';
  import { page } from '$app/state';
  import { canonicalUrl, isNoindex } from '#lib/seo.js';
  import AppShell from '#lib/components/AppShell.svelte';
  import type { LayoutProps } from './$types';

  // M3：品牌色板由 scripts/generate-brand-css.mjs 在构建期生成并静态引入，
  // 首屏即为品牌色，不需要在 onMount 调用 setColorScheme（避免色板跳动）。
  // 主题 class 由 app.html 的内联脚本在首屏绘制前设置。

  let { data, children }: LayoutProps = $props();

  const noindex = $derived(isNoindex(page.url.pathname));
  const canonical = $derived(canonicalUrl(page.url.origin, page.url.pathname));
  const siteName = $derived(data.siteOptions?.site_name ?? 'VOIDXO');

  // 认证页与后台自带完整版式，不套用前台外壳。
  const bare = $derived(/^\/(login|register|forgot-password|admin)(\/|$)/.test(page.url.pathname));
  const width = $derived.by(() => {
    const path = page.url.pathname;
    if (/^\/(questions|articles)\/[^/]+$/.test(path) && !path.endsWith('/new')) return 'detail' as const;
    if (/^\/(questions|articles|search|notifications|messages)(\/|$)/.test(path)) return 'list' as const;
    return 'content' as const;
  });
</script>

<svelte:head>
  <!-- 私有与交互页不参与索引，公开页给出唯一 canonical，避免重复收录。 -->
  {#if noindex}
    <meta name="robots" content="noindex, nofollow" />
  {:else}
    <link rel="canonical" href={canonical} />
  {/if}
</svelte:head>

{#if bare}
  {@render children()}
{:else}
  <AppShell siteName={siteName} user={data.siteUser} width={width}>
    {@render children()}
  </AppShell>
{/if}
