import type { PageServerLoad } from './$types';
import { listTopics, type Topic, type TopicSort } from '#lib/api/content.js';
import { getFollowStates } from '#lib/api/engagement.js';

export const load: PageServerLoad = async ({ fetch, url, parent }): Promise<{
  topics: Topic[];
  following: Record<string, boolean>;
  sort: TopicSort;
  page: number;
  hasMore: boolean;
  loadError: boolean;
}> => {
  const { siteUser } = await parent();
  const sort: TopicSort = url.searchParams.get('sort') === 'new' ? 'new' : 'recommended';
  const page = Math.max(1, Number(url.searchParams.get('page') ?? '1') || 1);

  try {
    const result = await listTopics(20, fetch, sort, page);
    const ids = result.items.map((topic) => topic.id);
    // 登录后一次性取回本页关注状态，避免每张卡片各发一个请求。
    const following = siteUser && ids.length > 0 ? await getFollowStates('topic', ids, fetch) : {};
    return {
      topics: result.items,
      following,
      sort,
      page: result.page,
      hasMore: result.has_more,
      loadError: false
    };
  } catch {
    return { topics: [], following: {}, sort, page, hasMore: false, loadError: true };
  }
};
