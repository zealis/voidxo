<script lang="ts">
  import Avatar from './Avatar.svelte';
  import FollowButton from './FollowButton.svelte';
  import type { PublicUser } from '#lib/api/users.js';

  type Props = {
    user: PublicUser;
    canFollow: boolean;
    following?: boolean;
  };

  let { user, canFollow, following = false }: Props = $props();
</script>

<article class="user-card vx-card">
  <a class="user-cover" href={`/users/${user.id}`} aria-hidden="true" tabindex={-1}>
    {#if user.cover_key}
      <img src={`/media/${user.cover_key}/middle`} alt="" loading="lazy" />
    {:else}
      <span class="cover-placeholder"></span>
    {/if}
  </a>
  <div class="user-body">
    <a class="user-avatar" href={`/users/${user.id}`}>
      <Avatar name={user.username} avatarKey={user.avatar_key} size={64} />
    </a>
    <a class="user-name vx-clamp-1" href={`/users/${user.id}`}>{user.username}</a>
    <p class="user-headline vx-clamp-1">{user.headline || '这个人很神秘，什么都没写'}</p>
    <div class="user-actions">
      <FollowButton
        kind="user"
        id={user.id}
        initial={{ following, follower_count: user.follower_count }}
        canFollow={canFollow}
        variant="icon"
      />
      <span class="user-followers">{user.follower_count} 人关注</span>
    </div>
  </div>
</article>

<style>
  .user-card { display: flex; flex-direction: column; overflow: hidden; }
  .user-cover { display: block; aspect-ratio: 16 / 6; background: var(--vx-color-surface-subtle); }
  .user-cover img { display: block; width: 100%; height: 100%; object-fit: cover; }
  .cover-placeholder {
    display: block;
    width: 100%;
    height: 100%;
    background: linear-gradient(135deg, hsl(214 45% 92%), hsl(214 30% 84%));
  }
  .user-body { position: relative; display: flex; flex-direction: column; gap: var(--vx-space-1); padding: 0 var(--vx-space-4) var(--vx-space-4); }
  .user-avatar { margin-top: -32px; margin-bottom: var(--vx-space-2); width: fit-content; }
  .user-avatar :global(.avatar) { border: 3px solid var(--vx-color-surface); }
  .user-name { font-size: var(--vx-font-size-subtitle); font-weight: 500; }
  .user-name:hover { color: var(--vx-color-primary); }
  .user-headline { color: var(--vx-color-text-secondary); font-size: var(--vx-font-size-small); }
  .user-actions { display: flex; align-items: center; justify-content: space-between; margin-top: var(--vx-space-2); }
  .user-followers { color: var(--vx-color-text-secondary); font-size: var(--vx-font-size-small); }
</style>
