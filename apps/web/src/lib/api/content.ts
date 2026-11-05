import { AuthRequestError, fetchCSRFToken } from './auth.js';

export type TopicRef = { id: number; name: string };

export type QuestionSummary = {
  id: number;
  user_id: number;
  username: string;
  avatar_key: string;
  title: string;
  content_markdown?: string;
  content_rendered?: string;
  comment_count: number;
  answer_count: number;
  follower_count: number;
  vote_count: number;
  last_answer_at?: string;
  created_at: string;
  updated_at: string;
  topics: TopicRef[];
};

export type QuestionDetail = QuestionSummary & { content_rendered: string };

export type Answer = {
  id: number;
  question_id: number;
  user_id: number;
  username: string;
  avatar_key: string;
  content_markdown?: string;
  content_rendered: string;
  comment_count: number;
  vote_count: number;
  created_at: string;
  updated_at: string;
};

export type TopicSort = 'recommended' | 'new' | 'name';
export type QuestionSort = 'recent' | 'new' | 'popular';
export type ArticleSort = 'recent' | 'new' | 'popular';

export type Topic = {
  id: number;
  name: string;
  description: string;
  cover_key: string;
  article_count: number;
  question_count: number;
  follower_count: number;
  created_at: string;
};

export type ArticleSummary = {
  id: number;
  user_id: number;
  username: string;
  avatar_key: string;
  title: string;
  content_markdown?: string;
  content_rendered?: string;
  comment_count: number;
  follower_count: number;
  vote_count: number;
  created_at: string;
  updated_at: string;
  topics: TopicRef[];
};

export type ArticleDetail = ArticleSummary & { content_rendered: string };

export type CommentItem = {
  id: number;
  commentable_id: number;
  commentable_type: string;
  root_id?: number;
  user_id: number;
  username: string;
  avatar_key: string;
  content: string;
  reply_count: number;
  vote_count: number;
  created_at: string;
  updated_at: string;
};

export type Page<T> = {
  items: T[];
  page: number;
  page_size: number;
  has_more: boolean;
};

export class ContentRequestError extends Error {
  constructor(message: string, readonly status: number, readonly code: string) {
    super(message);
    this.name = 'ContentRequestError';
  }
}

export async function getTopic(id: number, fetchFn: typeof fetch = fetch): Promise<Topic> {
  const payload = await request(`/api/v1/topics/${id}`, { credentials: 'same-origin' }, fetchFn);
  return requireItem<Topic>(payload, 'topic', '话题');
}

export async function listQuestions(
  page = 1,
  pageSize = 10,
  topicId?: number,
  fetchFn: typeof fetch = fetch,
  sort?: QuestionSort
): Promise<Page<QuestionSummary>> {
  const query = new URLSearchParams({ page: String(page), page_size: String(pageSize) });
  if (topicId) query.set('topic_id', String(topicId));
  if (sort) query.set('sort', sort);
  return readPage<QuestionSummary>(`/api/v1/questions?${query}`, '问题', fetchFn);
}

export async function getQuestion(id: number, fetchFn: typeof fetch = fetch): Promise<QuestionDetail> {
  const payload = await request(`/api/v1/questions/${id}`, { credentials: 'same-origin' }, fetchFn);
  return requireItem<QuestionDetail>(payload, 'question', '问题');
}

export async function listAnswers(questionId: number, page = 1, pageSize = 20, fetchFn: typeof fetch = fetch): Promise<Page<Answer>> {
  return readPage<Answer>(`/api/v1/questions/${questionId}/answers?page=${page}&page_size=${pageSize}`, '回答', fetchFn);
}

export async function listTopics(
  pageSize = 50,
  fetchFn: typeof fetch = fetch,
  sort?: TopicSort,
  page = 1
): Promise<Page<Topic>> {
  const query = new URLSearchParams({ page: String(page), page_size: String(pageSize) });
  if (sort) query.set('sort', sort);
  return readPage<Topic>(`/api/v1/topics?${query}`, '话题', fetchFn);
}

export async function createQuestion(input: {
  title: string;
  content_markdown: string;
  topic_ids: number[];
}, fetchFn: typeof fetch = fetch): Promise<QuestionDetail> {
  const csrfToken = await fetchCSRFToken(fetchFn);
  const payload = await request('/api/v1/questions', {
    method: 'POST',
    credentials: 'same-origin',
    headers: { 'content-type': 'application/json', 'X-CSRF-Token': csrfToken },
    body: JSON.stringify(input)
  }, fetchFn);
  return requireItem<QuestionDetail>(payload, 'question', '问题');
}

export async function updateQuestion(id: number, input: {
  title: string;
  content_markdown: string;
  topic_ids: number[];
}, fetchFn: typeof fetch = fetch): Promise<QuestionDetail> {
  const csrfToken = await fetchCSRFToken(fetchFn);
  const payload = await request(`/api/v1/questions/${id}`, {
    method: 'PATCH',
    credentials: 'same-origin',
    headers: { 'content-type': 'application/json', 'X-CSRF-Token': csrfToken },
    body: JSON.stringify(input)
  }, fetchFn);
  return requireItem<QuestionDetail>(payload, 'question', '问题');
}

export async function deleteQuestion(id: number, fetchFn: typeof fetch = fetch): Promise<void> {
  const csrfToken = await fetchCSRFToken(fetchFn);
  await request(`/api/v1/questions/${id}`, {
    method: 'DELETE',
    credentials: 'same-origin',
    headers: { 'X-CSRF-Token': csrfToken }
  }, fetchFn);
}

export async function updateAnswer(answerId: number, contentMarkdown: string, fetchFn: typeof fetch = fetch): Promise<Answer> {
  const csrfToken = await fetchCSRFToken(fetchFn);
  const payload = await request(`/api/v1/answers/${answerId}`, {
    method: 'PATCH',
    credentials: 'same-origin',
    headers: { 'content-type': 'application/json', 'X-CSRF-Token': csrfToken },
    body: JSON.stringify({ content_markdown: contentMarkdown })
  }, fetchFn);
  return requireItem<Answer>(payload, 'answer', '回答');
}

export async function deleteAnswer(answerId: number, fetchFn: typeof fetch = fetch): Promise<void> {
  const csrfToken = await fetchCSRFToken(fetchFn);
  await request(`/api/v1/answers/${answerId}`, {
    method: 'DELETE',
    credentials: 'same-origin',
    headers: { 'X-CSRF-Token': csrfToken }
  }, fetchFn);
}

export async function createAnswer(questionId: number, contentMarkdown: string, fetchFn: typeof fetch = fetch): Promise<Answer> {
  const csrfToken = await fetchCSRFToken(fetchFn);
  const payload = await request(`/api/v1/questions/${questionId}/answers`, {
    method: 'POST',
    credentials: 'same-origin',
    headers: { 'content-type': 'application/json', 'X-CSRF-Token': csrfToken },
    body: JSON.stringify({ content_markdown: contentMarkdown })
  }, fetchFn);
  return requireItem<Answer>(payload, 'answer', '回答');
}

type Payload = {
  data?: Record<string, unknown>;
  error?: { code?: string; message?: string };
};

export async function listArticles(
  page = 1,
  pageSize = 10,
  topicId?: number,
  fetchFn: typeof fetch = fetch,
  sort?: ArticleSort
): Promise<Page<ArticleSummary>> {
  const query = new URLSearchParams({ page: String(page), page_size: String(pageSize) });
  if (topicId) query.set('topic_id', String(topicId));
  if (sort) query.set('sort', sort);
  return readPage<ArticleSummary>(`/api/v1/articles?${query}`, '文章', fetchFn);
}

export async function getArticle(id: number, fetchFn: typeof fetch = fetch): Promise<ArticleDetail> {
  const payload = await request(`/api/v1/articles/${id}`, { credentials: 'same-origin' }, fetchFn);
  return requireItem<ArticleDetail>(payload, 'article', '文章');
}

export async function createArticle(input: {
  title: string;
  content_markdown: string;
  topic_ids: number[];
}, fetchFn: typeof fetch = fetch): Promise<ArticleDetail> {
  const csrfToken = await fetchCSRFToken(fetchFn);
  const payload = await request('/api/v1/articles', {
    method: 'POST',
    credentials: 'same-origin',
    headers: { 'content-type': 'application/json', 'X-CSRF-Token': csrfToken },
    body: JSON.stringify(input)
  }, fetchFn);
  return requireItem<ArticleDetail>(payload, 'article', '文章');
}

export async function updateArticle(id: number, input: {
  title: string;
  content_markdown: string;
  topic_ids: number[];
}, fetchFn: typeof fetch = fetch): Promise<ArticleDetail> {
  const csrfToken = await fetchCSRFToken(fetchFn);
  const payload = await request(`/api/v1/articles/${id}`, {
    method: 'PATCH',
    credentials: 'same-origin',
    headers: { 'content-type': 'application/json', 'X-CSRF-Token': csrfToken },
    body: JSON.stringify(input)
  }, fetchFn);
  return requireItem<ArticleDetail>(payload, 'article', '文章');
}

export async function deleteArticle(id: number, fetchFn: typeof fetch = fetch): Promise<void> {
  const csrfToken = await fetchCSRFToken(fetchFn);
  await request(`/api/v1/articles/${id}`, {
    method: 'DELETE',
    credentials: 'same-origin',
    headers: { 'X-CSRF-Token': csrfToken }
  }, fetchFn);
}

export async function listComments(kind: string, id: number, page = 1, pageSize = 50, fetchFn: typeof fetch = fetch): Promise<Page<CommentItem>> {
  return readPage<CommentItem>(`/api/v1/comments?type=${kind}&id=${id}&page=${page}&page_size=${pageSize}`, '评论', fetchFn);
}

export async function createComment(input: {
  commentable_type: string;
  commentable_id: number;
  content: string;
  root_id?: number;
}, fetchFn: typeof fetch = fetch): Promise<CommentItem> {
  const csrfToken = await fetchCSRFToken(fetchFn);
  const payload = await request('/api/v1/comments', {
    method: 'POST',
    credentials: 'same-origin',
    headers: { 'content-type': 'application/json', 'X-CSRF-Token': csrfToken },
    body: JSON.stringify(input)
  }, fetchFn);
  return requireItem<CommentItem>(payload, 'comment', '评论');
}

export async function deleteComment(id: number, fetchFn: typeof fetch = fetch): Promise<void> {
  const csrfToken = await fetchCSRFToken(fetchFn);
  await request(`/api/v1/comments/${id}`, {
    method: 'DELETE',
    credentials: 'same-origin',
    headers: { 'X-CSRF-Token': csrfToken }
  }, fetchFn);
}

async function readPage<T>(url: string, label: string, fetchFn: typeof fetch): Promise<Page<T>> {
  const payload = await request(url, { credentials: 'same-origin' }, fetchFn);
  const data = payload.data;
  if (!data || (data.items !== undefined && data.items !== null && !Array.isArray(data.items))) {
    throw new ContentRequestError(`${label}列表响应格式不正确`, 200, 'invalid_response');
  }
  // 服务端已保证空列表为 []，这里再兜底一次历史响应。
  return { ...(data as unknown as Page<T>), items: Array.isArray(data.items) ? data.items : [] };
}

async function request(url: string, init: RequestInit, fetchFn: typeof fetch): Promise<Payload> {
  const response = await fetchFn(url, init);
  let payload: Payload | null = null;
  try {
    payload = (await response.json()) as Payload;
  } catch {
    payload = null;
  }
  if (!response.ok) {
    throw new ContentRequestError(payload?.error?.message ?? '内容请求失败', response.status, payload?.error?.code ?? 'content_request_failed');
  }
  return payload ?? {};
}

function requireItem<T>(payload: Payload, key: string, label: string): T {
  const item = payload.data?.[key];
  if (!item || typeof item !== 'object' || typeof (item as { id?: unknown }).id !== 'number') {
    throw new ContentRequestError(`${label}响应格式不正确`, 200, 'invalid_response');
  }
  return item as T;
}

export { AuthRequestError };
