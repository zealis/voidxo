import { describe, expect, it, vi } from 'vitest';
import { registerAccount, resetPassword, sendEmailCode } from './account';

const csrfResponse = () =>
  new Response(JSON.stringify({ data: { csrf_token: 'x'.repeat(40) } }), { status: 200 });

describe('account API', () => {
  it('发送验证码返回有效期', async () => {
    const fetchFn = vi
      .fn()
      .mockResolvedValueOnce(csrfResponse())
      .mockResolvedValueOnce(new Response(JSON.stringify({ data: { expires_in: 600 } }), { status: 202 }));

    const expiresIn = await sendEmailCode('a@b.com', 'register', fetchFn);
    expect(fetchFn.mock.calls[1][0]).toBe('/api/v1/email/code');
    expect(fetchFn.mock.calls[1][1]).toMatchObject({ method: 'POST' });
    expect(expiresIn).toBe(600);
  });

  it('发送过于频繁时抛出可展示异常', async () => {
    const fetchFn = vi
      .fn()
      .mockResolvedValueOnce(csrfResponse())
      .mockResolvedValueOnce(new Response(JSON.stringify({
        error: { code: 'email_code_rate_limited', message: '验证码发送过于频繁，请稍后再试' }
      }), { status: 429 }));

    await expect(sendEmailCode('a@b.com', 'reset', fetchFn)).rejects.toMatchObject({
      status: 429,
      code: 'email_code_rate_limited'
    });
  });

  it('注册成功返回归一化的用户', async () => {
    const fetchFn = vi
      .fn()
      .mockResolvedValueOnce(csrfResponse())
      .mockResolvedValueOnce(new Response(JSON.stringify({
        data: { user: { id: 7, username: 'newbie', email: 'a@b.com', roles: null, permissions: null } }
      }), { status: 201 }));

    const user = await registerAccount({ username: 'newbie', email: 'a@b.com', password: 'GoodPass!2026', code: '123456' }, fetchFn);
    expect(user.roles).toEqual([]);
    expect(user.permissions).toEqual([]);
  });

  it('注册冲突映射为 409', async () => {
    const fetchFn = vi
      .fn()
      .mockResolvedValueOnce(csrfResponse())
      .mockResolvedValueOnce(new Response(JSON.stringify({
        error: { code: 'account_exists', message: '该用户名或邮箱已被使用' }
      }), { status: 409 }));

    await expect(registerAccount({ username: 'x', email: 'a@b.com', password: 'GoodPass!2026', code: '1' }, fetchFn))
      .rejects.toMatchObject({ status: 409, code: 'account_exists' });
  });

  it('重置密码提交 new_password 字段', async () => {
    const fetchFn = vi
      .fn()
      .mockResolvedValueOnce(csrfResponse())
      .mockResolvedValueOnce(new Response(JSON.stringify({ data: { reset: true } }), { status: 200 }));

    await resetPassword({ email: 'a@b.com', code: '123456', newPassword: 'Rotated!2026' }, fetchFn);
    const body = JSON.parse(fetchFn.mock.calls[1][1].body as string);
    expect(body).toEqual({ email: 'a@b.com', code: '123456', new_password: 'Rotated!2026' });
  });
});
