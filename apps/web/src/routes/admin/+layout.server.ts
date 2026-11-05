import { error, redirect } from '@sveltejs/kit';
import type { LayoutServerLoad } from './$types';
import { fetchCurrentUser, type AuthUser } from '#lib/api/auth.js';

export const load: LayoutServerLoad = async ({ fetch, setHeaders }): Promise<{ user: AuthUser }> => {
  setHeaders({ 'cache-control': 'private, no-store' });
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
