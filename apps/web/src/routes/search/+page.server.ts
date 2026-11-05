import { redirect } from '@sveltejs/kit';
import type { PageServerLoad } from './$types';
import { fetchCurrentUser, type AuthUser } from '#lib/api/auth.js';
import { search, SearchRequestError, type SearchKind, type SearchResult } from '#lib/api/search.js';

const kinds: SearchKind[] = ['all', 'question', 'article', 'topic'];

export const load: PageServerLoad = async ({ fetch, url }): Promise<{
  user: AuthUser;
  query: string;
  kind: SearchKind;
  page: number;
  result: SearchResult | null;
  errorMessage: string;
}> => {
  const user = await fetchCurrentUser(fetch);
  if (!user) {
    redirect(303, '/login');
  }

  const query = (url.searchParams.get('q') ?? '').trim();
  const rawKind = url.searchParams.get('type') ?? 'all';
  const kind: SearchKind = kinds.includes(rawKind as SearchKind) ? (rawKind as SearchKind) : 'all';
  const page = Math.max(1, Number(url.searchParams.get('page') ?? 1) || 1);

  if (!query) {
    return { user, query: '', kind, page, result: null, errorMessage: '' };
  }

  try {
    const result = await search(query, kind, page, 10, fetch);
    return { user, query, kind, page, result, errorMessage: '' };
  } catch (cause) {
    if (cause instanceof SearchRequestError) {
      return { user, query, kind, page, result: null, errorMessage: cause.message };
    }
    return { user, query, kind, page, result: null, errorMessage: '搜索暂时不可用，请稍后重试' };
  }
};
