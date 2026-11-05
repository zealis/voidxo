import { redirect } from '@sveltejs/kit';
import type { PageServerLoad } from './$types';
import { fetchCurrentUser } from '#lib/api/auth.js';

export const load: PageServerLoad = async ({ fetch }) => {
  const user = await fetchCurrentUser(fetch);
  if (user) {
    // 已登录用户直接在资料页改密码。
    redirect(303, '/user');
  }
  return {};
};
