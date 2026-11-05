import type { PageServerLoad } from './$types';
import { listReports, type AdminReport } from '#lib/api/admin.js';

export const load: PageServerLoad = async ({ fetch, url }): Promise<{
  reports: AdminReport[];
  page: number;
  hasMore: boolean;
  status: string;
  loadError: boolean;
}> => {
  const page = Math.max(1, Number(url.searchParams.get('page') ?? '1') || 1);
  const status = url.searchParams.get('status') ?? 'pending';
  try {
    const result = await listReports({ page, pageSize: 20, status }, fetch);
    return { reports: result.items, page: result.page, hasMore: result.has_more, status, loadError: false };
  } catch {
    return { reports: [], page, hasMore: false, status, loadError: true };
  }
};
