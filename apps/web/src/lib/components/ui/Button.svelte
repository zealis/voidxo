<script lang="ts">
  import type { Snippet } from 'svelte';

  type Props = {
    type?: 'button' | 'submit' | 'reset';
    variant?: 'primary' | 'outlined' | 'text';
    disabled?: boolean;
    loading?: boolean;
    href?: string;
    ariaLabel?: string;
    onclick?: () => void;
    children: Snippet;
  };

  let { type = 'button', variant = 'primary', disabled = false, loading = false, href, ariaLabel, onclick, children }: Props = $props();
</script>

{#if href}
  <a
    class="vx-button"
    class:vx-button--outlined={variant === 'outlined'}
    class:vx-button--text={variant === 'text'}
    {href}
    aria-disabled={disabled || loading}
    aria-label={ariaLabel}
  >
    {@render children()}
  </a>
{:else}
  <button
    class="vx-button"
    class:vx-button--outlined={variant === 'outlined'}
    class:vx-button--text={variant === 'text'}
    {type}
    disabled={disabled || loading}
    aria-disabled={disabled || loading}
    aria-busy={loading}
    aria-label={ariaLabel}
    {onclick}
  >
    {@render children()}
  </button>
{/if}

<style>
  .vx-button {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    min-height: var(--vx-button-height);
    padding: var(--vx-button-padding);
    color: var(--vx-button-primary-text);
    font: inherit;
    font-weight: 500;
    background: var(--vx-button-primary-bg);
    border: 0;
    border-radius: var(--vx-button-radius);
    cursor: pointer;
    transition: background-color var(--vx-motion-standard) ease;
  }
  .vx-button--outlined {
    color: var(--vx-button-outlined-text);
    background: transparent;
    border: 1px solid var(--vx-button-outlined-border);
  }
  .vx-button--text {
    color: var(--vx-button-outlined-text);
    background: transparent;
  }
  .vx-button:hover:not(:disabled) { filter: brightness(0.96); }
  .vx-button:focus-visible { outline: 2px solid var(--vx-color-focus); outline-offset: 2px; }
  .vx-button:disabled,
  .vx-button[aria-disabled='true'] { cursor: not-allowed; opacity: 0.48; }
  @media (prefers-reduced-motion: reduce) { .vx-button { transition: none; } }
</style>
