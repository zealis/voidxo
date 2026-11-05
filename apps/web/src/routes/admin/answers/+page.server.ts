import type { PageServerLoad } from './$types';
import { listContent, type AdminContentItem, type DeletedFilter } from '#lib/api/admin.js';

const kind = 'answer' as const;

export const load: PageServerLoad = async ({ fetch, url }): Promise<{
  items: AdminContentItem[];
  page: number;
  hasMore: boolean;
  query: string;
  deleted: string;
  loadError: boolean;
}> => {
  const page = Math.max(1, Number(url.searchParams.get('page') ?? '1') || 1);
  const query = url.searchParams.get('q') ?? '';
  const deleted = (url.searchParams.get('deleted') ?? 'live') as DeletedFilter;
  try {
    const result = await listContent(kind, { page, pageSize: 20, query, deleted }, fetch);
    return { items: result.items, page: result.page, hasMore: result.has_more, query, deleted, loadError: false };
  } catch {
    return { items: [], page, hasMore: false, query, deleted, loadError: true };
  }
};
