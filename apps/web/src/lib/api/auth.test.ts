import { describe, expect, it, vi } from 'vitest';
import { AuthRequestError, loginWithPassword } from './auth';

describe('loginWithPassword', () => {
  it('先获取 CSRF，再提交登录', async () => {
    const fetchFn = vi.fn()
      .mockResolvedValueOnce(new Response(JSON.stringify({ data: { csrf_token: 'a'.repeat(32) } }), { status: 200 }))
      .mockResolvedValueOnce(new Response(JSON.stringify({ data: { user: { id: 7, username: 'alice', email: 'a@example.com', roles: [], permissions: [] } } }), { status: 200 }));

    await expect(loginWithPassword('alice', 'correct-password', fetchFn)).resolves.toMatchObject({ id: 7, username: 'alice' });
    expect(fetchFn).toHaveBeenCalledTimes(2);
    expect(fetchFn.mock.calls[1][1]).toMatchObject({
      method: 'POST',
      headers: { 'X-CSRF-Token': 'a'.repeat(32) }
    });
  });

  it('把服务端错误转成可展示的错误', async () => {
    const fetchFn = vi.fn()
      .mockResolvedValueOnce(new Response(JSON.stringify({ data: { csrf_token: 'a'.repeat(32) } }), { status: 200 }))
      .mockResolvedValueOnce(new Response(JSON.stringify({ error: { code: 'invalid_credentials', message: '账号或密码错误' } }), { status: 401 }));

    await expect(loginWithPassword('alice', 'wrong', fetchFn)).rejects.toMatchObject({
      name: 'AuthRequestError',
      code: 'invalid_credentials',
      status: 401
    } satisfies Partial<AuthRequestError>);
  });
});
