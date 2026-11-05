export type PublicProfile = {
  id: number;
  username: string;
  avatar_key: string;
  cover_key: string;
  headline: string;
  bio: string;
  blog: string;
  company: string;
  location: string;
  follower_count: number;
  followee_count: number;
  article_count: number;
  question_count: number;
  answer_count: number;
  comment_count: number;
  created_at: string;
};

/** 列表卡片只需要公开摘要字段。 */
export type PublicUser = Pick<
  PublicProfile,
  'id' | 'username' | 'avatar_key' | 'cover_key' | 'headline' | 'follower_count'
>;

export type PublicUserPage = {
  items: PublicUser[];
  page: number;
  page_size: number;
  has_more: boolean;
};

export type UserContentItem = {
  kind: 'question' | 'answer' | 'article' | 'comment';
  id: number;
  title: string;
  excerpt: string;
  target_type?: string;
  target_id?: number;
  comment_count: number;
  vote_count: number;
  created_at: string;
};

export type UserContentKind = 'questions' | 'answers' | 'articles' | 'comments';

export type UserContentPage = {
  items: UserContentItem[];
  page: number;
  page_size: number;
  has_more: boolean;
};

type Payload = {
  data?: Record<string, unknown>;
  error?: { code?: string; message?: string };
};

export class UserRequestError extends Error {
  constructor(message: string, readonly status: number, readonly code: string) {
    super(message);
    this.name = 'UserRequestError';
  }
}

export async function fetchPublicProfile(id: number, fetchFn: typeof fetch = fetch): Promise<PublicProfile> {
  let response: Response;
  try {
    response = await fetchFn(`/api/v1/users/${id}`, { credentials: 'same-origin' });
  } catch {
    throw new UserRequestError('无法连接服务', 0, 'network_error');
  }
  const payload = (await response.json().catch(() => ({}))) as Payload;
  const user = payload.data?.user as PublicProfile | undefined;
  if (!response.ok || !user) {
    throw new UserRequestError(
      payload.error?.message ?? '用户不存在',
      response.status,
      payload.error?.code ?? 'invalid_response'
    );
  }
  return user;
}

export async function listUserContent(
  id: number,
  kind: UserContentKind,
  page = 1,
  pageSize = 20,
  fetchFn: typeof fetch = fetch
): Promise<UserContentPage> {
  const response = await fetchFn(`/api/v1/users/${id}/${kind}?page=${page}&page_size=${pageSize}`, {
    credentials: 'same-origin'
  });
  const payload = (await response.json().catch(() => ({}))) as Payload;
  const data = payload.data;
  if (!response.ok) {
    throw new UserRequestError(
      payload.error?.message ?? '内容加载失败',
      response.status,
      payload.error?.code ?? 'unknown_error'
    );
  }
  if (!data || (data.items !== undefined && data.items !== null && !Array.isArray(data.items))) {
    throw new UserRequestError('内容响应格式不正确', 200, 'invalid_response');
  }
  return {
    ...(data as unknown as UserContentPage),
    items: Array.isArray(data.items) ? data.items : []
  };
}

export async function listUsers(
  page = 1,
  pageSize = 20,
  sort: 'recommended' | 'new' = 'recommended',
  fetchFn: typeof fetch = fetch
): Promise<PublicUserPage> {
  let response: Response;
  try {
    response = await fetchFn(`/api/v1/users?page=${page}&page_size=${pageSize}&sort=${sort}`, {
      credentials: 'same-origin'
    });
  } catch {
    throw new UserRequestError('无法连接服务', 0, 'network_error');
  }
  const payload = (await response.json().catch(() => ({}))) as Payload;
  if (!response.ok) {
    throw new UserRequestError(
      payload.error?.message ?? '用户列表加载失败',
      response.status,
      payload.error?.code ?? 'unknown_error'
    );
  }
  const data = payload.data;
  if (!data || (data.items !== undefined && data.items !== null && !Array.isArray(data.items))) {
    throw new UserRequestError('用户列表响应格式不正确', 200, 'invalid_response');
  }
  return {
    ...(data as unknown as PublicUserPage),
    items: Array.isArray(data.items) ? (data.items as PublicUser[]) : []
  };
}
