<script lang="ts">
  import { untrack } from 'svelte';
  import { goto } from '$app/navigation';
  import Icon from './Icon.svelte';
  import { EngagementRequestError, follow, unfollow, type FollowSummary, type FollowTarget } from '#lib/api/engagement.js';

  type Props = {
    kind: FollowTarget;
    id: number;
    initial: FollowSummary;
    canFollow: boolean;
    /** text 用于详情页，icon 用于旧站风格的星标按钮。 */
    variant?: 'text' | 'icon';
    label?: string;
  };

  let { kind, id, initial, canFollow, variant = 'text', label }: Props = $props();
  let summary = $state(untrack(() => initial));
  let pending = $state(false);
  let error = $state('');

  async function toggle() {
    if (pending) return;
    if (!canFollow) {
      await goto('/login');
      return;
    }
    error = '';
    pending = true;
    try {
      summary = summary.following ? await unfollow(kind, id) : await follow(kind, id);
    } catch (cause) {
      if (cause instanceof EngagementRequestError && cause.status === 401) {
        await goto('/login');
        return;
      }
      error = cause instanceof EngagementRequestError ? cause.message : '操作失败';
    } finally {
      pending = false;
    }
  }
</script>

<div class="follow follow--{variant}">
  <button
    type="button"
    class="follow-button"
    class:following={summary.following}
    aria-pressed={summary.following}
    aria-label={variant === 'icon' ? (summary.following ? '取消关注' : '关注') : undefined}
    disabled={pending}
    onclick={toggle}
  >
    {#if variant === 'icon'}
      <Icon name="star" size={20} filled={summary.following} />
    {:else}
      {summary.following ? '已关注' : '关注'}<span>{summary.follower_count}</span>
    {/if}
  </button>
  {#if error}<p class="follow-error" role="alert">{error}</p>{/if}
</div>

<style>
  .follow { display: inline-flex; flex-direction: column; gap: 2px; }
  .follow-button {
    display: inline-flex;
    align-items: center;
    gap: 6px;
    padding: 4px 10px;
    color: var(--vx-color-text-secondary);
    font-size: var(--vx-font-size-caption);
    background: var(--vx-color-surface);
    border: 1px solid var(--vx-color-border);
    border-radius: var(--vx-radius-control);
    cursor: pointer;
    transition: color var(--vx-motion-standard) var(--vx-motion-easing),
      border-color var(--vx-motion-standard) var(--vx-motion-easing);
  }
  .follow-button:hover { color: var(--vx-color-text-primary); }
  .follow-button.following { color: var(--vx-color-text-primary); border-color: var(--vx-color-text-primary); }
  .follow-button:disabled { opacity: 0.6; cursor: default; }

  /* 旧站星标按钮：圆形描边图标按钮。 */
  .follow--icon .follow-button {
    width: 36px;
    height: 36px;
    padding: 0;
    justify-content: center;
    color: var(--vx-color-text-secondary);
    border-radius: var(--vx-radius-pill);
  }
  .follow--icon .follow-button:hover { background: var(--vx-color-surface-hover); }
  .follow--icon .follow-button.following { color: var(--vx-color-primary); border-color: var(--vx-color-border); }
  .follow-error { color: var(--vx-color-error); font-size: var(--vx-font-size-caption); }
</style>
