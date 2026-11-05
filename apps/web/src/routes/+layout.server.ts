import type { LayoutServerLoad } from './$types';
import { readPublicOptions, type PublicOptions } from '#lib/api/contracts.js';
import { fetchCurrentUser, type AuthUser } from '#lib/api/auth.js';

// 应用外壳需要站点名与登录态，统一在根布局加载，避免每个页面重复实现。
export const load: LayoutServerLoad = async ({ fetch, setHeaders }): Promise<{
  siteOptions: PublicOptions | null;
  siteUser: AuthUser | null;
}> => {
  setHeaders({ 'cache-control': 'private, no-store' });
  const [optionsResult, userResult] = await Promise.allSettled([
    fetch('/api/v1/options').then(readPublicOptions),
    fetchCurrentUser(fetch)
  ]);
  return {
    siteOptions: optionsResult.status === 'fulfilled' ? optionsResult.value : null,
    siteUser: userResult.status === 'fulfilled' ? userResult.value : null
  };
};
