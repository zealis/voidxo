import { AuthRequestError } from './auth.js';
import type { ArticleSummary, Page, QuestionSummary, Topic } from './content.js';

export type SearchResult = {
  query: string;
  questions: Page<QuestionSummary>;
  articles: Page<ArticleSummary>;
  topics: Page<Topic>;
};

export type SearchKind = 'all' | 'question' | 'article' | 'topic';

export class SearchRequestError extends Error {
  constructor(message: string, readonly status: number, readonly code: string) {
    super(message);
    this.name = 'SearchRequestError';
  }
}

export async function search(
  query: string,
  kind: SearchKind = 'all',
  page = 1,
  pageSize = 10,
  fetchFn: typeof fetch = fetch
): Promise<SearchResult> {
  const params = new URLSearchParams({ q: query, type: kind, page: String(page), page_size: String(pageSize) });
  const response = await fetchFn(`/api/v1/search?${params}`, { credentials: 'same-origin' });

  let payload: { data?: SearchResult; error?: { code?: string; message?: string } } | null = null;
  try {
    payload = await response.json();
  } catch {
    payload = null;
  }
  if (!response.ok) {
    throw new SearchRequestError(
      payload?.error?.message ?? '搜索失败',
      response.status,
      payload?.error?.code ?? 'search_failed'
    );
  }
  const data = payload?.data;
  if (!data || typeof data.query !== 'string') {
    throw new SearchRequestError('搜索响应格式不正确', response.status, 'invalid_response');
  }
  return {
    query: data.query,
    questions: normalizePage(data.questions),
    articles: normalizePage(data.articles),
    topics: normalizePage(data.topics)
  };
}

function normalizePage<T>(page: Page<T> | undefined): Page<T> {
  return {
    items: Array.isArray(page?.items) ? page.items : [],
    page: page?.page ?? 1,
    page_size: page?.page_size ?? 10,
    has_more: page?.has_more ?? false
  };
}

export { AuthRequestError };
