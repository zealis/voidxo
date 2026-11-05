import { AuthRequestError, fetchCSRFToken } from './auth.js';

export type ProfileUser = {
  id: number;
  username: string;
  email: string;
  avatar_key: string;
  cover_key: string;
  headline: string;
  bio: string;
  blog: string;
  company: string;
  location: string;
  roles: string[];
  permissions: string[];
};

export type ProfileUpdate = Pick<ProfileUser, 'headline' | 'bio' | 'blog' | 'company' | 'location'>;

export async function fetchProfile(fetchFn: typeof fetch = fetch): Promise<ProfileUser> {
  const response = await fetchFn('/api/v1/user', { credentials: 'same-origin' });
  const payload = await readPayload(response);
  if (!response.ok) {
    throw new AuthRequestError(
      payload?.error?.message ?? '资料加载失败',
      response.status,
      payload?.error?.code ?? 'profile_unavailable'
    );
  }
  return requireProfile(payload?.data?.user, response.status);
}

export async function updateProfile(value: ProfileUpdate, fetchFn: typeof fetch = fetch): Promise<ProfileUser> {
  const csrfToken = await fetchCSRFToken(fetchFn);
  const response = await fetchFn('/api/v1/user', {
    method: 'PATCH',
    credentials: 'same-origin',
    headers: { 'content-type': 'application/json', 'X-CSRF-Token': csrfToken },
    body: JSON.stringify(value)
  });
  const payload = await readPayload(response);
  if (!response.ok) {
    throw new AuthRequestError(
      payload?.error?.message ?? '资料保存失败',
      response.status,
      payload?.error?.code ?? 'profile_save_failed'
    );
  }
  return requireProfile(payload?.data?.user, response.status);
}

type Payload = {
  data?: { user?: unknown };
  error?: { code?: string; message?: string };
};

async function readPayload(response: Response): Promise<Payload | null> {
  try {
    return (await response.json()) as Payload;
  } catch {
    return null;
  }
}

function requireProfile(value: unknown, status: number): ProfileUser {
  if (!value || typeof value !== 'object' || typeof (value as ProfileUser).username !== 'string') {
    throw new AuthRequestError('资料响应格式不正确', status, 'invalid_response');
  }
  return value as ProfileUser;
}
