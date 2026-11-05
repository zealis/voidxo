import { redirect } from '@sveltejs/kit';
import type { PageServerLoad } from './$types';

export const load: PageServerLoad = async ({ fetch }) => {
  const response = await fetch('/api/v1/auth/me');
  if (response.ok) {
    redirect(303, '/');
  }
  return {};
};
