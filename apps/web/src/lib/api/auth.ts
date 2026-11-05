export type AuthUser = {
  id: number;
  username: string;
  email: string;
  avatar_key: string;
  roles: string[];
  permissions: string[];
};

export class AuthRequestError extends Error {
  constructor(message: string, readonly status: number, readonly code: string) {
    super(message);
    this.name = 'AuthRequestError';
  }
}

export async function fetchCSRFToken(fetchFn: typeof fetch = fetch): Promise<string> {
  const response = await fetchFn('/api/v1/auth/csrf', { credentials: 'same-origin' });
  const payload = await readPayload(response);
  const token = payload?.data?.csrf_token;
  if (!response.ok || typeof token !== 'string' || token.length < 32) {
    throw new AuthRequestError('安全令牌获取失败，请刷新页面后重试', response.status, 'csrf_unavailable');
  }
  return token;
}

export async function fetchCurrentUser(fetchFn: typeof fetch = fetch): Promise<AuthUser | null> {
  const response = await fetchFn('/api/v1/auth/me', { credentials: 'same-origin' });
  if (response.status === 401) return null;
  const payload = await readPayload(response);
  if (!response.ok) {
    throw new AuthRequestError(
      payload?.error?.message ?? '登录状态加载失败',
      response.status,
      payload?.error?.code ?? 'auth_unavailable'
    );
  }
  const user = payload?.data?.user;
  if (!user || typeof user.id !== 'number' || typeof user.username !== 'string') {
    throw new AuthRequestError('登录状态响应格式不正确', response.status, 'invalid_response');
  }
  return normalizeUser(user);
}

export async function logout(fetchFn: typeof fetch = fetch): Promise<void> {
  const csrfToken = await fetchCSRFToken(fetchFn);
  const response = await fetchFn('/api/v1/auth/logout', {
    method: 'POST',
    credentials: 'same-origin',
    headers: { 'X-CSRF-Token': csrfToken }
  });
  if (!response.ok) {
    const payload = await readPayload(response);
    throw new AuthRequestError(
      payload?.error?.message ?? '退出失败，请稍后重试',
      response.status,
      payload?.error?.code ?? 'logout_failed'
    );
  }
}

export async function changePassword(currentPassword: string, newPassword: string, fetchFn: typeof fetch = fetch): Promise<void> {
  const csrfToken = await fetchCSRFToken(fetchFn);
  const response = await fetchFn('/api/v1/auth/password', {
    method: 'POST',
    credentials: 'same-origin',
    headers: { 'content-type': 'application/json', 'X-CSRF-Token': csrfToken },
    body: JSON.stringify({ current_password: currentPassword, new_password: newPassword })
  });
  if (!response.ok) {
    const payload = await readPayload(response);
    throw new AuthRequestError(
      payload?.error?.message ?? '密码更新失败',
      response.status,
      payload?.error?.code ?? 'password_change_failed'
    );
  }
}

export async function loginWithPassword(
  login: string,
  password: string,
  fetchFn: typeof fetch = fetch
): Promise<AuthUser> {
  const csrfToken = await fetchCSRFToken(fetchFn);
  const response = await fetchFn('/api/v1/auth/login', {
    method: 'POST',
    credentials: 'same-origin',
    headers: { 'content-type': 'application/json', 'X-CSRF-Token': csrfToken },
    body: JSON.stringify({ login, password })
  });
  const payload = await readPayload(response);
  if (!response.ok) {
    throw new AuthRequestError(
      payload?.error?.message ?? '登录失败，请稍后重试',
      response.status,
      payload?.error?.code ?? 'login_failed'
    );
  }
  const user = payload?.data?.user;
  if (!user || typeof user.id !== 'number' || typeof user.username !== 'string') {
    throw new AuthRequestError('登录响应格式不正确', response.status, 'invalid_response');
  }
  return normalizeUser(user);
}

// 服务端已保证角色与权限为数组，这里再兜底历史会话中的 null。
function normalizeUser(user: AuthUser): AuthUser {
  return {
    ...user,
    avatar_key: typeof user.avatar_key === 'string' ? user.avatar_key : '',
    roles: Array.isArray(user.roles) ? user.roles : [],
    permissions: Array.isArray(user.permissions) ? user.permissions : []
  };
}

type Payload = {
  data?: Record<string, unknown> & { csrf_token?: unknown; user?: AuthUser };
  error?: { code?: string; message?: string };
};

async function readPayload(response: Response): Promise<Payload | null> {
  try {
    return (await response.json()) as Payload;
  } catch {
    return null;
  }
}
