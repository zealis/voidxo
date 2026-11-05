import type { PageServerLoad } from './$types';
import { fetchStats, type AdminStats } from '#lib/api/admin.js';

export const load: PageServerLoad = async ({ fetch }): Promise<{ stats: AdminStats | null; loadError: boolean }> => {
  try {
    return { stats: await fetchStats(fetch), loadError: false };
  } catch {
    return { stats: null, loadError: true };
  }
};
