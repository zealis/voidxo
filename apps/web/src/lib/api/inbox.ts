import { fetchCSRFToken } from './auth.js';

export type InboxRecipient = {
  id: number;
  username: string;
  avatar_key: string;
};

export type InboxMessage = {
  id: number;
  sender_id: number;
  sender_name: string;
  receiver_id: number;
  content_markdown: string;
  content_rendered: string;
  read: boolean;
  created_at: string;
};

export type InboxConversation = {
  user_id: number;
  username: string;
  avatar_key: string;
  last_excerpt: string;
  last_sender_id: number;
  last_at: string;
  unread: number;
};

export type InboxPage<T> = {
  items: T[];
  page: number;
  page_size: number;
  has_more: boolean;
};

export type InboxConversationPage = InboxPage<InboxMessage> & {
  counterpart: InboxRecipient;
};

type Payload = {
  data?: Record<string, unknown>;
  error?: { code?: string; message?: string };
};

export class InboxRequestError extends Error {
  constructor(message: string, readonly status: number, readonly code: string) {
    super(message);
    this.name = 'InboxRequestError';
  }
}

async function request(path: string, init: RequestInit, fetchFn: typeof fetch): Promise<Payload> {
  let response: Response;
  try {
    response = await fetchFn(path, init);
  } catch {
    throw new InboxRequestError('无法连接服务，请稍后重试', 0, 'network_error');
  }
  let payload: Payload = {};
  try {
    payload = (await response.json()) as Payload;
  } catch {
    payload = {};
  }
  if (!response.ok) {
    throw new InboxRequestError(
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

function requirePage<T>(payload: Payload, label: string): InboxPage<T> {
  const data = payload.data;
  if (!data || (data.items !== undefined && data.items !== null && !Array.isArray(data.items))) {
    throw new InboxRequestError(`${label}响应格式不正确`, 200, 'invalid_response');
  }
  return {
    ...(data as unknown as InboxPage<T>),
    items: Array.isArray(data.items) ? data.items : []
  };
}

export async function listConversations(
  page = 1,
  pageSize = 20,
  fetchFn: typeof fetch = fetch
): Promise<InboxPage<InboxConversation>> {
  const payload = await request(
    `/api/v1/inbox?page=${page}&page_size=${pageSize}`,
    { credentials: 'same-origin' },
    fetchFn
  );
  return requirePage<InboxConversation>(payload, '会话列表');
}

export async function listMessages(
  userId: number,
  page = 1,
  pageSize = 30,
  fetchFn: typeof fetch = fetch
): Promise<InboxConversationPage> {
  const payload = await request(
    `/api/v1/inbox/${userId}?page=${page}&page_size=${pageSize}`,
    { credentials: 'same-origin' },
    fetchFn
  );
  const data = payload.data;
  const counterpart = data?.counterpart as InboxRecipient | undefined;
  if (!counterpart || typeof counterpart.id !== 'number') {
    throw new InboxRequestError('会话响应格式不正确', 200, 'invalid_response');
  }
  const pageData = requirePage<InboxMessage>(payload, '会话消息');
  return { ...pageData, counterpart };
}

export async function sendMessage(
  receiverId: number,
  contentMarkdown: string,
  fetchFn: typeof fetch = fetch
): Promise<InboxMessage> {
  const payload = await write('/api/v1/inbox', 'POST', { receiver_id: receiverId, content_markdown: contentMarkdown }, fetchFn);
  const message = payload.data?.message as InboxMessage | undefined;
  if (!message || typeof message.id !== 'number') {
    throw new InboxRequestError('发送响应格式不正确', 200, 'invalid_response');
  }
  return message;
}

export async function markConversationRead(
  userId: number,
  fetchFn: typeof fetch = fetch
): Promise<{ updated: number; unread: number }> {
  const csrfToken = await fetchCSRFToken(fetchFn);
  const payload = await request(
    `/api/v1/inbox/${userId}/read`,
    { method: 'POST', credentials: 'same-origin', headers: { 'X-CSRF-Token': csrfToken } },
    fetchFn
  );
  return {
    updated: Number(payload.data?.updated ?? 0),
    unread: Number(payload.data?.unread ?? 0)
  };
}

export async function unreadMessageCount(fetchFn: typeof fetch = fetch): Promise<number> {
  const payload = await request('/api/v1/inbox/unread_count', { credentials: 'same-origin' }, fetchFn);
  return Number(payload.data?.unread ?? 0);
}

export async function findRecipient(username: string, fetchFn: typeof fetch = fetch): Promise<InboxRecipient> {
  const payload = await request(
    `/api/v1/inbox/recipient?username=${encodeURIComponent(username)}`,
    { credentials: 'same-origin' },
    fetchFn
  );
  const recipient = payload.data?.recipient as InboxRecipient | undefined;
  if (!recipient || typeof recipient.id !== 'number') {
    throw new InboxRequestError('用户查询响应格式不正确', 200, 'invalid_response');
  }
  return recipient;
}
