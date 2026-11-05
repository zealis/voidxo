<script lang="ts">
  type Props = {
    name: string;
    avatarKey?: string;
    size?: number;
    variant?: 'small' | 'middle' | 'large';
    alt?: string;
  };

  let { name, avatarKey = '', size = 40, variant = 'middle', alt }: Props = $props();

  // 无头像时用用户名首字与稳定色相兜底，避免列表出现空白方块。
  const initial = $derived((name || '?').trim().slice(0, 1).toUpperCase());
  const hue = $derived(
    Array.from(name || '?').reduce((acc, char) => (acc * 31 + char.codePointAt(0)!) % 360, 7)
  );
  // 字号沿用旧系统 MDAvatars 的默认头像比例（256px 图：拉丁 196px、中文 151px）。
  // 中文字形接近满 em，需要比拉丁字母更小的字号才不会撑破圆形。
  const isAsian = $derived(/[\u3400-\u9fff\u3007\uf900-\ufaff]/.test(initial));
  const fontSize = $derived(Math.round(size * (isAsian ? 151 / 256 : 196 / 256) * 100) / 100);
</script>

{#if avatarKey}
  <img
    class="avatar"
    src={`/media/${avatarKey}/${variant}`}
    alt={alt ?? `${name} 的头像`}
    width={size}
    height={size}
    loading="lazy"
    style={`--size:${size}px`}
  />
{:else}
  <span
    class="avatar avatar--fallback"
    aria-hidden="true"
    style={`--size:${size}px;--hue:${hue};--font-size:${fontSize}px`}
  >{initial}</span>
{/if}

<style>
  .avatar {
    display: inline-flex;
    flex: none;
    align-items: center;
    justify-content: center;
    width: var(--size);
    height: var(--size);
    border-radius: var(--vx-radius-pill);
    object-fit: cover;
    background: var(--vx-color-surface-subtle);
  }
  .avatar--fallback {
    color: hsl(var(--hue) 32% 32%);
    font-size: var(--font-size);
    font-weight: 400;
    line-height: 1;
    background: hsl(var(--hue) 42% 92%);
  }
</style>
