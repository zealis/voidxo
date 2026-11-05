<script lang="ts">
  import { untrack } from 'svelte';
  import { goto } from '$app/navigation';
  import Icon from './Icon.svelte';
  import { EngagementRequestError, unvote, vote, type VoteSummary, type VoteTarget } from '#lib/api/engagement.js';

  type Props = {
    kind: VoteTarget;
    id: number;
    initial: VoteSummary;
    canVote: boolean;
  };

  let { kind, id, initial, canVote }: Props = $props();
  let summary = $state(untrack(() => initial));
  let pending = $state(false);
  let error = $state('');

  async function act(next: 'up' | 'down') {
    if (pending) return;
    if (!canVote) {
      await goto('/login');
      return;
    }
    error = '';
    pending = true;
    try {
      if (summary.my_vote === next) {
        summary = await unvote(kind, id);
      } else {
        summary = await vote(kind, id, next);
      }
    } catch (cause) {
      if (cause instanceof EngagementRequestError && cause.status === 401) {
        await goto('/login');
        return;
      }
      error = cause instanceof EngagementRequestError ? cause.message : '投票失败';
    } finally {
      pending = false;
    }
  }
</script>

<div class="vote" role="group" aria-label="投票">
  <!-- 旧站为圆形描边图标按钮，选中态只改变图标颜色。 -->
  <button
    type="button"
    class="vote-button"
    class:active={summary.my_vote === 'up'}
    aria-pressed={summary.my_vote === 'up'}
    aria-label={`赞同 ${summary.vote_up_count}`}
    title="赞同"
    disabled={pending}
    onclick={() => act('up')}
  >
    <Icon name="thumb-up" size={20} filled={summary.my_vote === 'up'} />
  </button>
  <strong class="vote-score" aria-label="净票数">{summary.vote_count}</strong>
  <button
    type="button"
    class="vote-button"
    class:active={summary.my_vote === 'down'}
    aria-pressed={summary.my_vote === 'down'}
    aria-label={`反对 ${summary.vote_down_count}`}
    title="反对"
    disabled={pending}
    onclick={() => act('down')}
  >
    <Icon name="thumb-down" size={20} filled={summary.my_vote === 'down'} />
  </button>
</div>
{#if error}<p class="vote-error" role="alert">{error}</p>{/if}

<style>
  .vote { display: flex; align-items: center; gap: var(--vx-space-2); }
  .vote-button {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    width: 36px;
    height: 36px;
    padding: 0;
    color: var(--vx-color-text-secondary);
    background: var(--vx-color-surface);
    border: 1px solid var(--vx-color-border);
    border-radius: var(--vx-radius-pill);
    cursor: pointer;
    transition: color var(--vx-motion-standard) var(--vx-motion-easing),
      background-color var(--vx-motion-standard) var(--vx-motion-easing);
  }
  .vote-button:hover { color: var(--vx-color-text-primary); background: var(--vx-color-surface-hover); }
  .vote-button.active { color: var(--vx-color-primary); }
  .vote-button:disabled { opacity: 0.6; cursor: default; }
  .vote-score { min-width: 20px; font-size: var(--vx-font-size-body); font-weight: 500; text-align: center; }
  .vote-error { margin-top: var(--vx-space-1); color: var(--vx-color-error); font-size: var(--vx-font-size-caption); }
</style>
