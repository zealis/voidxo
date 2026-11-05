import type { PageServerLoad } from './$types';
import { listQuestions, type QuestionSort, type QuestionSummary } from '#lib/api/content.js';

const SORTS: QuestionSort[] = ['recent', 'popular', 'new'];

export const load: PageServerLoad = async ({ fetch, url }): Promise<{
  questions: QuestionSummary[];
  sort: QuestionSort;
  page: number;
  hasMore: boolean;
  topicId?: number;
  loadError: boolean;
}> => {
  const requested = url.searchParams.get('sort') as QuestionSort | null;
  const sort: QuestionSort = requested && SORTS.includes(requested) ? requested : 'recent';
  const page = Math.max(1, Number(url.searchParams.get('page') ?? '1') || 1);
  const rawTopic = Number(url.searchParams.get('topic_id'));
  const topicId = Number.isInteger(rawTopic) && rawTopic > 0 ? rawTopic : undefined;

  try {
    const result = await listQuestions(page, 20, topicId, fetch, sort);
    return {
      questions: result.items,
      sort,
      page: result.page,
      hasMore: result.has_more,
      topicId,
      loadError: false
    };
  } catch {
    return { questions: [], sort, page, hasMore: false, topicId, loadError: true };
  }
};
