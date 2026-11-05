import { redirect } from '@sveltejs/kit';
import type { PageServerLoad } from './$types';
import { fetchCurrentUser, type AuthUser } from '#lib/api/auth.js';
import { listTopics, type Topic } from '#lib/api/content.js';

export const load: PageServerLoad = async ({ fetch }): Promise<{ user: AuthUser; topics: Topic[] }> => {
  const user = await fetchCurrentUser(fetch);
  if (!user) {
    redirect(303, '/login');
  }
  const topics = await listTopics(50, fetch);
  return { user, topics: topics.items };
};
