import type { PageServerLoad } from './$types';
import { listUsers, type PublicUser } from '#lib/api/users.js';
import { getFollowStates } from '#lib/api/engagement.js';

export const load: PageServerLoad = async ({ fetch, url, parent }): Promise<{
  users: PublicUser[];
  following: Record<string, boolean>;
  sort: 'recommended' | 'new';
  page: number;
  hasMore: boolean;
  loadError: boolean;
}> => {
  const { siteUser } = await parent();
  const sort = url.searchParams.get('sort') === 'new' ? 'new' : 'recommended';
  const page = Math.max(1, Number(url.searchParams.get('page') ?? '1') || 1);

  try {
    const result = await listUsers(page, 20, sort, fetch);
    const ids = result.items.map((item) => item.id);
    const following = siteUser && ids.length > 0 ? await getFollowStates('user', ids, fetch) : {};
    return {
      users: result.items,
      following,
      sort,
      page: result.page,
      hasMore: result.has_more,
      loadError: false
    };
  } catch {
    return { users: [], following: {}, sort, page, hasMore: false, loadError: true };
  }
};
