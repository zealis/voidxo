import type { PageServerLoad } from './$types';
import { listArticles, type ArticleSort, type ArticleSummary } from '#lib/api/content.js';

const SORTS: ArticleSort[] = ['recent', 'popular', 'new'];

export const load: PageServerLoad = async ({ fetch, url }): Promise<{
  articles: ArticleSummary[];
  sort: ArticleSort;
  page: number;
  hasMore: boolean;
  topicId?: number;
  loadError: boolean;
}> => {
  const requested = url.searchParams.get('sort') as ArticleSort | null;
  const sort: ArticleSort = requested && SORTS.includes(requested) ? requested : 'recent';
  const page = Math.max(1, Number(url.searchParams.get('page') ?? '1') || 1);
  const rawTopic = Number(url.searchParams.get('topic_id'));
  const topicId = Number.isInteger(rawTopic) && rawTopic > 0 ? rawTopic : undefined;

  try {
    const result = await listArticles(page, 20, topicId, fetch, sort);
    return {
      articles: result.items,
      sort,
      page: result.page,
      hasMore: result.has_more,
      topicId,
      loadError: false
    };
  } catch {
    return { articles: [], sort, page, hasMore: false, topicId, loadError: true };
  }
};
