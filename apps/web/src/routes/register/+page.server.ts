import { redirect } from '@sveltejs/kit';
import type { PageServerLoad } from './$types';
import { fetchCurrentUser } from '#lib/api/auth.js';

export const load: PageServerLoad = async ({ fetch }) => {
  const user = await fetchCurrentUser(fetch);
  if (user) {
    // 已登录用户不需要注册页。
    redirect(303, '/');
  }
  return {};
};
