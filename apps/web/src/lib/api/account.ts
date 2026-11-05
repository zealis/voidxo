import { AuthRequestError, fetchCSRFToken, type AuthUser } from './auth.js';

export type EmailPurpose = 'register' | 'reset';

type Payload = {
  data?: Record<string, unknown>;
  error?: { code?: string; message?: string };
};

async function readPayload(response: Response): Promise<Payload> {
  try {
    return (await response.json()) as Payload;
  } catch {
    return {};
  }
}

function failure(response: Response, payload: Payload, fallback: string): AuthRequestError {
  return new AuthRequestError(
    payload.error?.message ?? fallback,
    response.status,
    payload.error?.code ?? 'account_request_failed'
  );
}

// 发送验证码：后端无论邮箱是否存在都返回 202，前端只处理限流与格式错误。
export async function sendEmailCode(
  email: string,
  purpose: EmailPurpose,
  fetchFn: typeof fetch = fetch
): Promise<number> {
  const csrfToken = await fetchCSRFToken(fetchFn);
  const response = await fetchFn('/api/v1/email/code', {
    method: 'POST',
    credentials: 'same-origin',
    headers: { 'content-type': 'application/json', 'X-CSRF-Token': csrfToken },
    body: JSON.stringify({ email, purpose })
  });
  const payload = await readPayload(response);
  if (!response.ok) {
    throw failure(response, payload, '验证码发送失败，请稍后重试');
  }
  const expiresIn = Number(payload.data?.expires_in ?? 0);
  return Number.isFinite(expiresIn) ? expiresIn : 0;
}

export async function registerAccount(
  input: { username: string; email: string; password: string; code: string },
  fetchFn: typeof fetch = fetch
): Promise<AuthUser> {
  const csrfToken = await fetchCSRFToken(fetchFn);
  const response = await fetchFn('/api/v1/auth/register', {
    method: 'POST',
    credentials: 'same-origin',
    headers: { 'content-type': 'application/json', 'X-CSRF-Token': csrfToken },
    body: JSON.stringify(input)
  });
  const payload = await readPayload(response);
  const user = payload.data?.user as AuthUser | undefined;
  if (!response.ok || !user) {
    throw failure(response, payload, '注册失败，请稍后重试');
  }
  return {
    ...user,
    roles: Array.isArray(user.roles) ? user.roles : [],
    permissions: Array.isArray(user.permissions) ? user.permissions : []
  };
}

export async function resetPassword(
  input: { email: string; code: string; newPassword: string },
  fetchFn: typeof fetch = fetch
): Promise<void> {
  const csrfToken = await fetchCSRFToken(fetchFn);
  const response = await fetchFn('/api/v1/auth/password/reset', {
    method: 'POST',
    credentials: 'same-origin',
    headers: { 'content-type': 'application/json', 'X-CSRF-Token': csrfToken },
    body: JSON.stringify({ email: input.email, code: input.code, new_password: input.newPassword })
  });
  if (!response.ok) {
    const payload = await readPayload(response);
    throw failure(response, payload, '密码重置失败，请稍后重试');
  }
}
