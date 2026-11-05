import { describe, expect, it, vi } from 'vitest';
import { fetchProfile, updateProfile } from './profile';

const profile = {
  id: 7,
  username: 'alice',
  email: 'a@example.com',
  avatar_key: '',
  cover_key: '',
  headline: '',
  bio: '',
  blog: '',
  company: '',
  location: '',
  roles: ['user'],
  permissions: []
};

describe('profile API', () => {
  it('读取资料', async () => {
    const fetchFn = vi.fn().mockResolvedValue(new Response(JSON.stringify({ data: { user: profile } }), { status: 200 }));
    await expect(fetchProfile(fetchFn)).resolves.toMatchObject({ username: 'alice' });
  });

  it('保存资料前获取 CSRF 并发送 PATCH', async () => {
    const fetchFn = vi.fn()
      .mockResolvedValueOnce(new Response(JSON.stringify({ data: { csrf_token: 'a'.repeat(32) } }), { status: 200 }))
      .mockResolvedValueOnce(new Response(JSON.stringify({ data: { user: { ...profile, headline: '介绍' } } }), { status: 200 }));

    await expect(updateProfile({
      headline: '介绍', bio: '', blog: '', company: '', location: ''
    }, fetchFn)).resolves.toMatchObject({ headline: '介绍' });
    expect(fetchFn.mock.calls[1][1]).toMatchObject({
      method: 'PATCH',
      headers: expect.objectContaining({ 'X-CSRF-Token': 'a'.repeat(32) })
    });
  });

  it('未登录时保留 401 供页面跳转', async () => {
    const fetchFn = vi.fn().mockResolvedValue(new Response(JSON.stringify({ error: { code: 'authentication_required', message: '请先登录' } }), { status: 401 }));
    await expect(fetchProfile(fetchFn)).rejects.toMatchObject({ status: 401, code: 'authentication_required' });
  });
});
