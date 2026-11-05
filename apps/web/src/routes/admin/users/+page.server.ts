import type { PageServerLoad } from './$types';
import { listUsers, type AdminUser } from '#lib/api/admin.js';

export const load: PageServerLoad = async ({ fetch, url }): Promise<{
  users: AdminUser[];
  page: number;
  hasMore: boolean;
  query: string;
  role: string;
  status: string;
  loadError: boolean;
}> => {
  const page = Math.max(1, Number(url.searchParams.get('page') ?? '1') || 1);
  const query = url.searchParams.get('q') ?? '';
  const role = url.searchParams.get('role') ?? '';
  const status = url.searchParams.get('status') ?? '';
  try {
    const result = await listUsers({ page, pageSize: 20, query, role, status }, fetch);
    return { users: result.items, page: result.page, hasMore: result.has_more, query, role, status, loadError: false };
  } catch {
    return { users: [], page, hasMore: false, query, role, status, loadError: true };
  }
};
