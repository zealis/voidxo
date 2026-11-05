import { redirect } from '@sveltejs/kit';
import type { PageServerLoad } from './$types';
import { fetchCurrentUser, type AuthUser } from '#lib/api/auth.js';
import { listNotifications, unreadNotificationCount, type NotificationItem } from '#lib/api/engagement.js';

export const load: PageServerLoad = async ({ fetch }): Promise<{
  user: AuthUser;
  notifications: NotificationItem[];
  unread: number;
}> => {
  const user = await fetchCurrentUser(fetch);
  if (!user) {
    redirect(303, '/login');
  }
  const [list, unread] = await Promise.all([listNotifications(1, 50, fetch), unreadNotificationCount(fetch)]);
  return { user, notifications: list.items, unread };
};
