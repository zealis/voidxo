import { AuthRequestError, fetchCSRFToken } from './auth.js';

export type VoteSummary = {
  vote_count: number;
  vote_up_count: number;
  vote_down_count: number;
  my_vote: 'up' | 'down' | '';
};

export type FollowSummary = {
  following: boolean;
  follower_count: number;
};

export type NotificationItem = {
  id: number;
  type: string;
  sender_id: number;
  sender_name: string;
  article_id?: number;
  question_id?: number;
  answer_id?: number;
  comment_id?: number;
  reply_id?: number;
  content_deleted?: string;
  read: boolean;
  created_at: string;
};

export type EngagementPage<T> = {
  items: T[];
  page: number;
  page_size: number;
  has_more: boolean;
};

export type VoteTarget = 'question' | 'answer' | 'article' | 'comment';
export type FollowTarget = 'user' | 'question' | 'article' | 'topic';

type Payload = {
  data?: Record<string, unknown>;
  error?: { code?: string; message?: string };
};

export class EngagementRequestError extends Error {
  constructor(message: string, readonly status: number, readonly code: string) {
    super(message);
    this.name = 'EngagementRequestError';
  }
}

export async function getVoteState(kind: VoteTarget, id: number, fetchFn: typeof fetch = fetch): Promise<VoteSummary> {
  const payload = await request(`/api/v1/votes?type=${kind}&id=${id}`, { credentials: 'same-origin' }, fetchFn);
  return requireVote(payload);
}

export async function vote(kind: VoteTarget, id: number, type: 'up' | 'down', fetchFn: typeof fetch = fetch): Promise<VoteSummary> {
  const csrfToken = await fetchCSRFToken(fetchFn);
  const payload = await request('/api/v1/votes', {
    method: 'POST',
    credentials: 'same-origin',
    headers: { 'content-type': 'application/json', 'X-CSRF-Token': csrfToken },
    body: JSON.stringify({ votable_type: kind, votable_id: id, type })
  }, fetchFn);
  return requireVote(payload);
}

export async function unvote(kind: VoteTarget, id: number, fetchFn: typeof fetch = fetch): Promise<VoteSummary> {
  const csrfToken = await fetchCSRFToken(fetchFn);
  const payload = await request(`/api/v1/votes?type=${kind}&id=${id}`, {
    method: 'DELETE',
    credentials: 'same-origin',
    headers: { 'X-CSRF-Token': csrfToken }
  }, fetchFn);
  return requireVote(payload);
}

export async function getFollowState(kind: FollowTarget, id: number, fetchFn: typeof fetch = fetch): Promise<FollowSummary> {
  const payload = await request(`/api/v1/follows?type=${kind}&id=${id}`, { credentials: 'same-origin' }, fetchFn);
  return requireFollow(payload);
}

/** 列表页批量取关注状态，返回纯对象便于直接按编号查询。 */
export async function getFollowStates(
  kind: FollowTarget,
  ids: number[],
  fetchFn: typeof fetch = fetch
): Promise<Record<string, boolean>> {
  if (ids.length === 0) return {};
  const payload = await request(`/api/v1/follows?type=${kind}&ids=${ids.join(',')}`, { credentials: 'same-origin' }, fetchFn);
  const states = (payload.data as { states?: Record<string, boolean> } | undefined)?.states;
  return states && typeof states === 'object' ? states : {};
}

export async function follow(kind: FollowTarget, id: number, fetchFn: typeof fetch = fetch): Promise<FollowSummary> {
  const csrfToken = await fetchCSRFToken(fetchFn);
  const payload = await request('/api/v1/follows', {
    method: 'POST',
    credentials: 'same-origin',
    headers: { 'content-type': 'application/json', 'X-CSRF-Token': csrfToken },
    body: JSON.stringify({ followable_type: kind, followable_id: id })
  }, fetchFn);
  return requireFollow(payload);
}

export async function unfollow(kind: FollowTarget, id: number, fetchFn: typeof fetch = fetch): Promise<FollowSummary> {
  const csrfToken = await fetchCSRFToken(fetchFn);
  const payload = await request(`/api/v1/follows?type=${kind}&id=${id}`, {
    method: 'DELETE',
    credentials: 'same-origin',
    headers: { 'X-CSRF-Token': csrfToken }
  }, fetchFn);
  return requireFollow(payload);
}

export async function createReport(
  kind: VoteTarget | 'user',
  id: number,
  reason: string,
  fetchFn: typeof fetch = fetch
): Promise<void> {
  const csrfToken = await fetchCSRFToken(fetchFn);
  await request('/api/v1/reports', {
    method: 'POST',
    credentials: 'same-origin',
    headers: { 'content-type': 'application/json', 'X-CSRF-Token': csrfToken },
    body: JSON.stringify({ reportable_type: kind, reportable_id: id, reason })
  }, fetchFn);
}

export async function listNotifications(page = 1, pageSize = 20, fetchFn: typeof fetch = fetch): Promise<EngagementPage<NotificationItem>> {
  const payload = await request(`/api/v1/notifications?page=${page}&page_size=${pageSize}`, { credentials: 'same-origin' }, fetchFn);
  const data = payload.data;
  if (!data || (data.items !== undefined && data.items !== null && !Array.isArray(data.items))) {
    throw new EngagementRequestError('通知列表响应格式不正确', 200, 'invalid_response');
  }
  return {
    ...(data as unknown as EngagementPage<NotificationItem>),
    items: Array.isArray(data.items) ? data.items : []
  };
}

export async function unreadNotificationCount(fetchFn: typeof fetch = fetch): Promise<number> {
  const payload = await request('/api/v1/notifications/unread_count', { credentials: 'same-origin' }, fetchFn);
  const unread = payload.data?.unread;
  if (typeof unread !== 'number') {
    throw new EngagementRequestError('未读数量响应格式不正确', 200, 'invalid_response');
  }
  return unread;
}

export async function markNotificationsRead(ids: number[], fetchFn: typeof fetch = fetch): Promise<number> {
  const csrfToken = await fetchCSRFToken(fetchFn);
  const payload = await request('/api/v1/notifications/read', {
    method: 'POST',
    credentials: 'same-origin',
    headers: { 'content-type': 'application/json', 'X-CSRF-Token': csrfToken },
    body: JSON.stringify({ ids })
  }, fetchFn);
  return typeof payload.data?.updated === 'number' ? payload.data.updated : 0;
}

export async function deleteNotifications(ids: number[], fetchFn: typeof fetch = fetch): Promise<number> {
  const csrfToken = await fetchCSRFToken(fetchFn);
  const payload = await request('/api/v1/notifications/delete', {
    method: 'POST',
    credentials: 'same-origin',
    headers: { 'content-type': 'application/json', 'X-CSRF-Token': csrfToken },
    body: JSON.stringify({ ids })
  }, fetchFn);
  return typeof payload.data?.deleted === 'number' ? payload.data.deleted : 0;
}

function requireVote(payload: Payload): VoteSummary {
  const item = payload.data?.vote;
  if (!item || typeof (item as VoteSummary).vote_count !== 'number') {
    throw new EngagementRequestError('投票响应格式不正确', 200, 'invalid_response');
  }
  return item as VoteSummary;
}

function requireFollow(payload: Payload): FollowSummary {
  const item = payload.data?.follow;
  if (!item || typeof (item as FollowSummary).following !== 'boolean') {
    throw new EngagementRequestError('关注响应格式不正确', 200, 'invalid_response');
  }
  return item as FollowSummary;
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
    throw new EngagementRequestError(
      payload?.error?.message ?? '操作失败',
      response.status,
      payload?.error?.code ?? 'engagement_request_failed'
    );
  }
  return payload ?? {};
}

export { AuthRequestError };
