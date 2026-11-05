<script lang="ts">
  import type { Topic } from '#lib/api/content.js';
  import FollowButton from './FollowButton.svelte';

  type Props = {
    topic: Topic;
    canFollow: boolean;
    following?: boolean;
  };

  let { topic, canFollow, following = false }: Props = $props();

  const hue = $derived(
    Array.from(topic.name).reduce((acc, char) => (acc * 31 + char.codePointAt(0)!) % 360, 11)
  );
  const coverStyle = $derived(
    topic.cover_key
      ? `background-image:url(/media/${topic.cover_key}/middle)`
      : `background-image:linear-gradient(135deg,hsl(${hue} 62% 88%),hsl(${(hue + 38) % 360} 58% 78%))`
  );
</script>

<article class="topic-card vx-card">
  <a class="topic-cover" href={`/topics/${topic.id}`} style={coverStyle} aria-label={`${topic.name} 话题`}></a>
  <div class="topic-body">
    <a class="topic-name vx-clamp-1" href={`/topics/${topic.id}`}>{topic.name}</a>
    <div class="topic-actions">
      <FollowButton
        kind="topic"
        id={topic.id}
        initial={{ following, follower_count: topic.follower_count }}
        canFollow={canFollow}
        variant="icon"
      />
      <a class="topic-followers" href={`/topics/${topic.id}`}>{topic.follower_count} 人关注</a>
    </div>
  </div>
</article>

<style>
  .topic-card { display: flex; flex-direction: column; overflow: hidden; }
  .topic-cover {
    display: block;
    aspect-ratio: 16 / 9;
    background-color: var(--vx-color-surface-subtle);
    background-position: center;
    background-size: cover;
  }
  .topic-body { display: flex; flex-direction: column; gap: var(--vx-space-2); padding: var(--vx-space-3) var(--vx-space-4) var(--vx-space-2); }
  .topic-name { font-size: var(--vx-font-size-subtitle); font-weight: 400; }
  .topic-name:hover { color: var(--vx-color-primary); }
  .topic-actions { display: flex; align-items: center; justify-content: space-between; }
  .topic-followers { color: var(--vx-color-text-secondary); font-size: var(--vx-font-size-small); }
  .topic-followers:hover { color: var(--vx-color-text-primary); }
</style>
