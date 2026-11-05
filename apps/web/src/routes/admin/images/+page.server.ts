import type { PageServerLoad } from './$types';
import { listImages, type AdminImage } from '#lib/api/admin.js';

export const load: PageServerLoad = async ({ fetch, url }): Promise<{
  images: AdminImage[];
  page: number;
  hasMore: boolean;
  query: string;
  deleted: string;
  loadError: boolean;
}> => {
  const page = Math.max(1, Number(url.searchParams.get('page') ?? '1') || 1);
  const query = url.searchParams.get('q') ?? '';
  const deleted = url.searchParams.get('deleted') ?? 'live';
  try {
    const result = await listImages({ page, pageSize: 24, query, deleted: deleted as 'live' | 'deleted' | 'all' }, fetch);
    return { images: result.items, page: result.page, hasMore: result.has_more, query, deleted, loadError: false };
  } catch {
    return { images: [], page, hasMore: false, query, deleted, loadError: true };
  }
};
