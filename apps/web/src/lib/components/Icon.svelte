<script lang="ts">
  // 旧站使用 Material Icons 字体；项目不引入图标运行时，改为内联 SVG。
  type IconName =
    | 'menu'
    | 'home'
    | 'tag'
    | 'forum'
    | 'article'
    | 'users'
    | 'search'
    | 'bell'
    | 'mail'
    | 'star'
    | 'arrow-right'
    | 'arrow-left'
    | 'plus'
    | 'comment'
    | 'thumb-up'
    | 'thumb-down'
    | 'more'
    | 'shield'
    | 'close';

  type Props = {
    name: IconName;
    size?: number;
    /** 实心图标（例如已关注）用 true。 */
    filled?: boolean;
    label?: string;
  };

  let { name, size = 24, filled = false, label }: Props = $props();

  const shapes: Record<IconName, string> = {
    menu: '<path d="M3.5 6.5h17M3.5 12h17M3.5 17.5h17"/>',
    home: '<path d="M3.5 10.6 12 3.5l8.5 7.1V20a1 1 0 0 1-1 1h-4.9v-6.4H9.4V21H4.5a1 1 0 0 1-1-1z"/>',
    tag: '<path d="M20.4 13.4 13.4 20.4a2 2 0 0 1-2.8 0L3.5 13.3V3.5h9.8l7.1 7.1a2 2 0 0 1 0 2.8Z"/><circle cx="7.8" cy="7.8" r="1.3"/>',
    forum: '<path d="M20.8 11.6a8.3 8.3 0 0 1-8.9 8.3 9 9 0 0 1-3.2-.6L3.5 20.8l1.7-5a8.2 8.2 0 0 1-.5-2.8 8.3 8.3 0 0 1 8.3-8.3 8.3 8.3 0 0 1 7.8 6.9Z"/>',
    article: '<path d="M6 3.5h8l4.5 4.5v12a1 1 0 0 1-1 1H6a1 1 0 0 1-1-1V4.5a1 1 0 0 1 1-1Z"/><path d="M14 3.5V8h4.5M8.5 12.5h7M8.5 16.5h7"/>',
    users: '<circle cx="9" cy="8" r="3.3"/><path d="M3.2 20c0-3.2 2.6-5.4 5.8-5.4s5.8 2.2 5.8 5.4"/><path d="M16 5.4a3.1 3.1 0 0 1 0 6M17.6 14.8c2.1.6 3.5 2.3 3.5 4.6"/>',
    search: '<circle cx="10.8" cy="10.8" r="6.3"/><path d="m15.5 15.5 4.5 4.5"/>',
    bell: '<path d="M18 9.2a6 6 0 1 0-12 0c0 6-2.4 6.6-2.4 8.3h16.8C20.4 15.8 18 15.2 18 9.2Z"/><path d="M10.2 20.5a2.1 2.1 0 0 0 3.6 0"/>',
    mail: '<rect x="3.5" y="5.5" width="17" height="13" rx="1.5"/><path d="m4.2 6.6 7.8 6 7.8-6"/>',
    star: '<path d="m12 3.8 2.5 5.1 5.6.8-4 3.9 1 5.6-5.1-2.7-5 2.7 1-5.6-4.1-3.9 5.6-.8Z"/>',
    'arrow-right': '<path d="M4.5 12h15M13.5 6l6 6-6 6"/>',
    'arrow-left': '<path d="M19.5 12h-15M10.5 6l-6 6 6 6"/>',
    plus: '<path d="M12 4.5v15M4.5 12h15"/>',
    comment: '<path d="M20.5 11.7a8 8 0 0 1-8.6 8 8.8 8.8 0 0 1-3-.5L4 20.5l1.3-4.6a8 8 0 0 1-.8-3.4 8 8 0 0 1 8-8 8 8 0 0 1 8 7.2Z"/>',
    'thumb-up': '<path d="M7 10.5v9.5H4.6a1 1 0 0 1-1-1v-7.5a1 1 0 0 1 1-1H7Z"/><path d="m7 10.5 3.6-7.1a1 1 0 0 1 1.8.4v5.2h4.9a2 2 0 0 1 2 2.3l-1 6.2a2 2 0 0 1-2 1.7H7"/>',
    'thumb-down': '<path d="M17 13.5V4h2.4a1 1 0 0 1 1 1v7.5a1 1 0 0 1-1 1H17Z"/><path d="m17 13.5-3.6 7.1a1 1 0 0 1-1.8-.4v-5.2H6.7a2 2 0 0 1-2-2.3l1-6.2a2 2 0 0 1 2-1.7H17"/>',
    more: '<circle cx="12" cy="5.5" r="1.4"/><circle cx="12" cy="12" r="1.4"/><circle cx="12" cy="18.5" r="1.4"/>',
    shield: '<path d="M12 3.5 5 6.2v5.3c0 4.3 2.9 7.6 7 9 4.1-1.4 7-4.7 7-9V6.2Z"/><path d="m9 11.8 2.2 2.2 4-4.2"/>',
    close: '<path d="m6 6 12 12M18 6 6 18"/>'
  };
</script>

<svg
  class="icon"
  width={size}
  height={size}
  viewBox="0 0 24 24"
  fill={filled ? 'currentColor' : 'none'}
  stroke="currentColor"
  stroke-width={filled ? 0 : 1.7}
  stroke-linecap="round"
  stroke-linejoin="round"
  role={label ? 'img' : 'presentation'}
  aria-hidden={label ? undefined : 'true'}
  aria-label={label}
>
  {@html shapes[name]}
</svg>

<style>
  .icon { display: block; flex: none; }
</style>
