import type { PageServerLoad } from './$types';
import { listTopics, type AdminTopic } from '#lib/api/admin.js';

export const load: PageServerLoad = async ({ fetch, url }): Promise<{
  topics: AdminTopic[];
  page: number;
  hasMore: boolean;
  query: string;
  includeDeleted: boolean;
  loadError: boolean;
}> => {
  const page = Math.max(1, Number(url.searchParams.get('page') ?? '1') || 1);
  const query = url.searchParams.get('q') ?? '';
  const includeDeleted = url.searchParams.get('include_deleted') === '1';
  try {
    const result = await listTopics({ page, pageSize: 20, query, includeDeleted }, fetch);
    return { topics: result.items, page: result.page, hasMore: result.has_more, query, includeDeleted, loadError: false };
  } catch {
    return { topics: [], page, hasMore: false, query, includeDeleted, loadError: true };
  }
};
