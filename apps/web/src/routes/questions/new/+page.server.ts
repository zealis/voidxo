import { redirect } from '@sveltejs/kit';
import type { PageServerLoad } from './$types';
import { fetchCurrentUser } from '#lib/api/auth.js';
import { listTopics, type Topic } from '#lib/api/content.js';

export const load: PageServerLoad = async ({ fetch }): Promise<{ topics: Topic[]; loadError: boolean }> => {
  try {
    const user = await fetchCurrentUser(fetch);
    if (!user) {
      redirect(303, '/login');
    }
    const topics = await listTopics(50, fetch);
    return { topics: topics.items, loadError: false };
  } catch (cause) {
    if (cause && typeof cause === 'object' && 'status' in cause && 'location' in cause) {
      throw cause;
    }
    return { topics: [], loadError: true };
  }
};
