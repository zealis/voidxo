<script lang="ts">
  type Item = { value: string; label: string };
  type Props = {
    items: Item[];
    active: string;
    basePath: string;
  };

  let { items, active, basePath }: Props = $props();

  function hrefFor(value: string) {
    return value === items[0]?.value ? basePath : `${basePath}?sort=${value}`;
  }
</script>

<nav class="tabs" aria-label="列表排序">
  {#each items as item}
    <a
      class="tab"
      class:active={item.value === active}
      href={hrefFor(item.value)}
      aria-current={item.value === active ? 'page' : undefined}
    >{item.label}</a>
  {/each}
</nav>

<style>
  /* 旧站二级标签：文字 + 底部指示条，只靠颜色不足以区分，另有下划线。 */
  .tabs {
    display: flex;
    gap: var(--vx-space-1);
    padding: 0 var(--vx-space-2);
    border-bottom: 1px solid var(--vx-color-divider);
  }
  .tab {
    position: relative;
    padding: var(--vx-space-3) var(--vx-space-4);
    color: var(--vx-color-text-secondary);
    font-size: var(--vx-font-size-body);
    font-weight: 500;
  }
  .tab:hover { color: var(--vx-color-text-primary); }
  .tab.active { color: var(--vx-color-primary); }
  .tab.active::after {
    position: absolute;
    right: var(--vx-space-2);
    bottom: -1px;
    left: var(--vx-space-2);
    height: 2px;
    background: var(--vx-color-primary);
    border-radius: 2px;
    content: '';
  }
  @media (max-width: 599.9px) {
    .tabs { padding: 0; }
    .tab { padding: var(--vx-space-3) var(--vx-space-3); }
  }
</style>
