import type { PageServerLoad } from './$types';
import { listOptions, type AdminOption } from '#lib/api/admin.js';

export const load: PageServerLoad = async ({ fetch }): Promise<{ options: AdminOption[]; loadError: boolean }> => {
  try {
    return { options: await listOptions(fetch), loadError: false };
  } catch {
    return { options: [], loadError: true };
  }
};
