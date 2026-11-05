import { error, redirect } from '@sveltejs/kit';
import type { LayoutServerLoad } from './$types';
import { fetchCurrentUser, type AuthUser } from '#lib/api/auth.js';

export const load: LayoutServerLoad = async ({ fetch }): Promise<{ user: AuthUser }> => {
  // 私有页面的 cache-control: private, no-store 已由根布局统一设置；
  // 子布局再次 setHeaders 同名头会触发 SvelteKit 的 header_already_set（500）。
  const user = await fetchCurrentUser(fetch);
  if (!user) {
    redirect(303, '/login');
  }
  // 权限判断放在数据加载之外，避免 error() 被 catch 重新包装成 500。
  if (!user.permissions.includes('admin.access')) {
    error(403, '需要管理权限');
  }
  return { user };
};
