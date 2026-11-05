import { describe, expect, it, vi } from 'vitest';
import { fetchPublicProfile, listUserContent } from './users';

describe('users API', () => {
  it('公开资料返回用户字段', async () => {
    const fetchFn = vi.fn().mockResolvedValue(new Response(JSON.stringify({
      data: { user: { id: 1, username: 'admin', question_count: 2 } }
    }), { status: 200 }));

    const profile = await fetchPublicProfile(1, fetchFn);
    expect(fetchFn).toHaveBeenCalledWith('/api/v1/users/1', { credentials: 'same-origin' });
    expect(profile.username).toBe('admin');
  });

  it('用户不存在时抛出 404', async () => {
    const fetchFn = vi.fn().mockResolvedValue(new Response(JSON.stringify({
      error: { code: 'not_found', message: '用户不存在' }
    }), { status: 404 }));

    await expect(fetchPublicProfile(999, fetchFn)).rejects.toMatchObject({ status: 404 });
  });

  it('内容列表空值归一化为空数组', async () => {
    const fetchFn = vi.fn().mockResolvedValue(new Response(JSON.stringify({
      data: { items: null, page: 1, page_size: 20, has_more: false }
    }), { status: 200 }));

    const page = await listUserContent(1, 'questions', 1, 20, fetchFn);
    expect(fetchFn.mock.calls[0][0]).toBe('/api/v1/users/1/questions?page=1&page_size=20');
    expect(page.items).toEqual([]);
  });

  it('内容响应格式不正确时抛错', async () => {
    const fetchFn = vi.fn().mockResolvedValue(new Response(JSON.stringify({ data: { items: 'oops' } }), { status: 200 }));
    await expect(listUserContent(1, 'articles', 1, 20, fetchFn)).rejects.toMatchObject({ code: 'invalid_response' });
  });
});
