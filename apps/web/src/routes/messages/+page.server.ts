import { redirect } from '@sveltejs/kit';
import type { PageServerLoad } from './$types';
import { fetchCurrentUser, type AuthUser } from '#lib/api/auth.js';
import { listConversations, unreadMessageCount, type InboxConversation } from '#lib/api/inbox.js';

export const load: PageServerLoad = async ({ fetch, url }): Promise<{
  user: AuthUser;
  conversations: InboxConversation[];
  page: number;
  hasMore: boolean;
  unread: number;
  loadError: boolean;
}> => {
  const user = await fetchCurrentUser(fetch);
  if (!user) {
    redirect(303, '/login');
  }
  const page = Math.max(1, Number(url.searchParams.get('page') ?? '1') || 1);
  try {
    const [list, unread] = await Promise.all([
      listConversations(page, 20, fetch),
      unreadMessageCount(fetch)
    ]);
    return {
      user,
      conversations: list.items,
      page: list.page,
      hasMore: list.has_more,
      unread,
      loadError: false
    };
  } catch {
    return { user, conversations: [], page, hasMore: false, unread: 0, loadError: true };
  }
};
