import { redirect } from '@sveltejs/kit';
import type { PageServerLoad } from './$types';
import { AuthRequestError } from '#lib/api/auth.js';
import { fetchProfile, type ProfileUser } from '#lib/api/profile.js';

export const load: PageServerLoad = async ({ fetch }): Promise<{ profile: ProfileUser | null; loadError: boolean }> => {
  try {
    return { profile: await fetchProfile(fetch), loadError: false };
  } catch (error) {
    if (error instanceof AuthRequestError && error.status === 401) {
      redirect(303, '/login');
    }
    return { profile: null, loadError: true };
  }
};
