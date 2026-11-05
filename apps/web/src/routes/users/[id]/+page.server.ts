import type { PageServerLoad } from './$types';
import { fetchCurrentUser, type AuthUser } from '#lib/api/auth.js';
import { getFollowState, type FollowSummary } from '#lib/api/engagement.js';
import {
  UserRequestError,
  fetchPublicProfile,
  listUserContent,
  type PublicProfile,
  type UserContentItem,
  type UserContentKind
} from '#lib/api/users.js';

const TABS: UserContentKind[] = ['questions', 'answers', 'articles', 'comments'];

type LoadResult = {
  user: AuthUser | null;
  profile: PublicProfile | null;
  follow: FollowSummary;
  items: UserContentItem[];
  tab: UserContentKind;
  page: number;
  hasMore: boolean;
  notice: string;
  loadError: boolean;
};

export const load: PageServerLoad = async ({ fetch, params, url }): Promise<LoadResult> => {
  const [userResult] = await Promise.allSettled([fetchCurrentUser(fetch)]);
  const currentUser = userResult.status === 'fulfilled' ? userResult.value : null;

  const id = Number(params.id);
  const tabParam = url.searchParams.get('tab') ?? 'questions';
  const tab = (TABS as string[]).includes(tabParam) ? (tabParam as UserContentKind) : 'questions';
  const page = Math.max(1, Number(url.searchParams.get('page') ?? '1') || 1);

  const empty: LoadResult = {
    user: currentUser,
    profile: null,
    follow: { following: false, follower_count: 0 },
    items: [],
    tab,
    page,
    hasMore: false,
    notice: '',
    loadError: false
  };

  if (!Number.isInteger(id) || id <= 0) {
    return { ...empty, notice: '用户不存在' };
  }

  let profile: PublicProfile;
  try {
    profile = await fetchPublicProfile(id, fetch);
  } catch (cause) {
    if (cause instanceof UserRequestError && cause.status === 404) {
      return { ...empty, notice: '用户不存在或已停用' };
    }
    return { ...empty, loadError: true };
  }

  let follow: FollowSummary = { following: false, follower_count: profile.follower_count };
  if (currentUser && currentUser.id !== profile.id) {
    try {
      follow = await getFollowState('user', profile.id, fetch);
    } catch {
      // 关注状态失败不影响主页展示。
    }
  }

  try {
    const content = await listUserContent(profile.id, tab, page, 20, fetch);
    return {
      user: currentUser,
      profile,
      follow,
      items: content.items,
      tab,
      page: content.page,
      hasMore: content.has_more,
      notice: '',
      loadError: false
    };
  } catch {
    return { user: currentUser, profile, follow, items: [], tab, page, hasMore: false, notice: '', loadError: true };
  }
};
