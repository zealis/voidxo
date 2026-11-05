import { fetchCSRFToken } from './auth.js';

export type AdminPage<T> = {
  items: T[];
  page: number;
  page_size: number;
  has_more: boolean;
};

export type AdminStats = {
  users: number;
  disabled_users: number;
  questions: number;
  answers: number;
  articles: number;
  comments: number;
  topics: number;
  images: number;
  pending_reports: number;
  new_users_7d: number;
  new_questions_7d: number;
  new_articles_7d: number;
  new_comments_7d: number;
};

export type AdminUser = {
  id: number;
  username: string;
  email: string;
  status: number;
  headline: string;
  roles: string[];
  article_count: number;
  question_count: number;
  answer_count: number;
  comment_count: number;
  created_at: string;
  last_login_at: string | null;
  disabled_at: string | null;
};

export type AdminContentItem = {
  id: number;
  title: string;
  excerpt: string;
  user_id: number;
  username: string;
  comment_count: number;
  vote_count: number;
  created_at: string;
  deleted: boolean;
  context: string;
  parent_id: number;
  parent_type?: string;
};

export type ContentKind = 'question' | 'answer' | 'article' | 'comment';

export type AdminTopic = {
  id: number;
  name: string;
  description: string;
  cover_key: string;
  article_count: number;
  question_count: number;
  follower_count: number;
  created_at: string;
  deleted: boolean;
};

export type AdminOption = {
  name: string;
  value: string;
  public: boolean;
  type: 'string' | 'bool' | 'int';
};

export type AdminImage = {
  key: string;
  filename: string;
  mime: string;
  size: number;
  width: number;
  height: number;
  kind: string;
  user_id: number;
  username: string;
  created_at: string;
  deleted: boolean;
};

export type DeletedFilter = 'live' | 'deleted' | 'all';

type Payload = {
  data?: Record<string, unknown>;
  error?: { code?: string; message?: string };
};

export class AdminRequestError extends Error {
  constructor(message: string, readonly status: number, readonly code: string) {
    super(message);
    this.name = 'AdminRequestError';
  }
}

async function request(path: string, init: RequestInit, fetchFn: typeof fetch): Promise<Payload> {
  let response: Response;
  try {
    response = await fetchFn(path, init);
  } catch {
    throw new AdminRequestError('无法连接服务，请稍后重试', 0, 'network_error');
  }
  let payload: Payload = {};
  try {
    payload = (await response.json()) as Payload;
  } catch {
    payload = {};
  }
  if (!response.ok) {
    throw new AdminRequestError(
      payload.error?.message ?? '请求失败',
      response.status,
      payload.error?.code ?? 'unknown_error'
    );
  }
  return payload;
}

async function write(path: string, method: string, body: unknown, fetchFn: typeof fetch): Promise<Payload> {
  const csrfToken = await fetchCSRFToken(fetchFn);
  return request(
    path,
    {
      method,
      credentials: 'same-origin',
      headers: { 'content-type': 'application/json', 'X-CSRF-Token': csrfToken },
      body: JSON.stringify(body)
    },
    fetchFn
  );
}

function requirePage<T>(payload: Payload, label: string): AdminPage<T> {
  const data = payload.data;
  if (!data || (data.items !== undefined && data.items !== null && !Array.isArray(data.items))) {
    throw new AdminRequestError(`${label}响应格式不正确`, 200, 'invalid_response');
  }
  return {
    ...(data as unknown as AdminPage<T>),
    items: Array.isArray(data.items) ? data.items : []
  };
}

export async function fetchStats(fetchFn: typeof fetch = fetch): Promise<AdminStats> {
  const payload = await request('/api/v1/admin/stats', { credentials: 'same-origin' }, fetchFn);
  const stats = payload.data?.stats as AdminStats | undefined;
  if (!stats) {
    throw new AdminRequestError('统计响应格式不正确', 200, 'invalid_response');
  }
  return stats;
}

export async function listUsers(
  params: { page?: number; pageSize?: number; query?: string; role?: string; status?: string },
  fetchFn: typeof fetch = fetch
): Promise<AdminPage<AdminUser>> {
  const search = new URLSearchParams();
  search.set('page', String(params.page ?? 1));
  search.set('page_size', String(params.pageSize ?? 20));
  if (params.query) search.set('q', params.query);
  if (params.role) search.set('role', params.role);
  if (params.status) search.set('status', params.status);
  const payload = await request(`/api/v1/admin/users?${search.toString()}`, { credentials: 'same-origin' }, fetchFn);
  return requirePage<AdminUser>(payload, '用户列表');
}

export async function updateUser(
  id: number,
  update: { role?: string; status?: number },
  fetchFn: typeof fetch = fetch
): Promise<void> {
  await write(`/api/v1/admin/users/${id}`, 'PATCH', update, fetchFn);
}

export async function listContent(
  kind: ContentKind,
  params: { page?: number; pageSize?: number; query?: string; deleted?: DeletedFilter },
  fetchFn: typeof fetch = fetch
): Promise<AdminPage<AdminContentItem>> {
  const search = new URLSearchParams();
  search.set('page', String(params.page ?? 1));
  search.set('page_size', String(params.pageSize ?? 20));
  if (params.query) search.set('q', params.query);
  if (params.deleted) search.set('deleted', params.deleted);
  const payload = await request(
    `/api/v1/admin/content/${kind}?${search.toString()}`,
    { credentials: 'same-origin' },
    fetchFn
  );
  return requirePage<AdminContentItem>(payload, '内容列表');
}

export async function deleteContent(
  kind: ContentKind,
  id: number,
  fetchFn: typeof fetch = fetch
): Promise<void> {
  const csrfToken = await fetchCSRFToken(fetchFn);
  await request(
    `/api/v1/admin/content/${kind}/${id}`,
    { method: 'DELETE', credentials: 'same-origin', headers: { 'X-CSRF-Token': csrfToken } },
    fetchFn
  );
}

export async function listTopics(
  params: { page?: number; pageSize?: number; query?: string; includeDeleted?: boolean },
  fetchFn: typeof fetch = fetch
): Promise<AdminPage<AdminTopic>> {
  const search = new URLSearchParams();
  search.set('page', String(params.page ?? 1));
  search.set('page_size', String(params.pageSize ?? 20));
  if (params.query) search.set('q', params.query);
  if (params.includeDeleted) search.set('include_deleted', '1');
  const payload = await request(`/api/v1/admin/topics?${search.toString()}`, { credentials: 'same-origin' }, fetchFn);
  return requirePage<AdminTopic>(payload, '话题列表');
}

export async function createTopic(
  input: { name: string; description: string },
  fetchFn: typeof fetch = fetch
): Promise<AdminTopic> {
  const payload = await write('/api/v1/admin/topics', 'POST', input, fetchFn);
  return payload.data?.topic as AdminTopic;
}

export async function updateTopic(
  id: number,
  input: { name: string; description: string },
  fetchFn: typeof fetch = fetch
): Promise<AdminTopic> {
  const payload = await write(`/api/v1/admin/topics/${id}`, 'PATCH', input, fetchFn);
  return payload.data?.topic as AdminTopic;
}

export async function deleteTopic(id: number, fetchFn: typeof fetch = fetch): Promise<void> {
  const csrfToken = await fetchCSRFToken(fetchFn);
  await request(
    `/api/v1/admin/topics/${id}`,
    { method: 'DELETE', credentials: 'same-origin', headers: { 'X-CSRF-Token': csrfToken } },
    fetchFn
  );
}

export async function listOptions(fetchFn: typeof fetch = fetch): Promise<AdminOption[]> {
  const payload = await request('/api/v1/admin/options', { credentials: 'same-origin' }, fetchFn);
  const items = payload.data?.items;
  if (!Array.isArray(items)) {
    throw new AdminRequestError('配置响应格式不正确', 200, 'invalid_response');
  }
  return items as AdminOption[];
}

export async function updateOptions(values: Record<string, string>, fetchFn: typeof fetch = fetch): Promise<AdminOption[]> {
  const payload = await write('/api/v1/admin/options', 'PATCH', { values }, fetchFn);
  const items = payload.data?.items;
  return Array.isArray(items) ? (items as AdminOption[]) : [];
}

export async function listImages(
  params: { page?: number; pageSize?: number; query?: string; deleted?: DeletedFilter },
  fetchFn: typeof fetch = fetch
): Promise<AdminPage<AdminImage>> {
  const search = new URLSearchParams();
  search.set('page', String(params.page ?? 1));
  search.set('page_size', String(params.pageSize ?? 24));
  if (params.query) search.set('q', params.query);
  if (params.deleted) search.set('deleted', params.deleted);
  const payload = await request(`/api/v1/admin/images?${search.toString()}`, { credentials: 'same-origin' }, fetchFn);
  return requirePage<AdminImage>(payload, '图片列表');
}

export async function deleteImage(key: string, fetchFn: typeof fetch = fetch): Promise<void> {
  const csrfToken = await fetchCSRFToken(fetchFn);
  await request(
    `/api/v1/admin/images/${key}`,
    { method: 'DELETE', credentials: 'same-origin', headers: { 'X-CSRF-Token': csrfToken } },
    fetchFn
  );
}

export type AdminReport = {
  id: number;
  reportable_id: number;
  reportable_type: string;
  user_id: number;
  username: string;
  reason: string;
  status: string;
  created_at: string;
};

export async function listReports(
  params: { page?: number; pageSize?: number; status?: string },
  fetchFn: typeof fetch = fetch
): Promise<AdminPage<AdminReport>> {
  const search = new URLSearchParams();
  search.set('page', String(params.page ?? 1));
  search.set('page_size', String(params.pageSize ?? 20));
  if (params.status) search.set('status', params.status);
  const payload = await request(`/api/v1/reports?${search.toString()}`, { credentials: 'same-origin' }, fetchFn);
  return requirePage<AdminReport>(payload, '举报列表');
}

export async function resolveReport(
  id: number,
  status: 'resolved' | 'rejected',
  fetchFn: typeof fetch = fetch
): Promise<void> {
  await write(`/api/v1/reports/${id}`, 'PATCH', { status }, fetchFn);
}
