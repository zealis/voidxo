import { describe, expect, it, vi } from 'vitest';
import { listContent, listImages, listUsers, updateOptions } from './admin';

describe('admin API', () => {
  it('内容列表为空时归一化为空数组', async () => {
    const fetchFn = vi.fn().mockResolvedValue(new Response(JSON.stringify({
      data: { items: null, page: 1, page_size: 20, has_more: false }
    }), { status: 200 }));

    const result = await listContent('question', { page: 2, deleted: 'all' }, fetchFn);
    expect(fetchFn).toHaveBeenCalledWith('/api/v1/admin/content/question?page=2&page_size=20&deleted=all', {
      credentials: 'same-origin'
    });
    expect(result.items).toEqual([]);
    expect(result.has_more).toBe(false);
  });

  it('用户筛选参数拼接到查询串', async () => {
    const fetchFn = vi.fn().mockResolvedValue(new Response(JSON.stringify({
      data: { items: [{ id: 1, username: 'admin', roles: [] }], page: 1, page_size: 20, has_more: false }
    }), { status: 200 }));

    const result = await listUsers({ query: 'admin', role: 'admin', status: 'active' }, fetchFn);
    expect(fetchFn.mock.calls[0][0]).toBe('/api/v1/admin/users?page=1&page_size=20&q=admin&role=admin&status=active');
    expect(result.items).toHaveLength(1);
  });

  it('接口错误转成可展示异常', async () => {
    // 写操作先取 CSRF 再提交，需要按顺序返回两个响应。
    const fetchFn = vi
      .fn()
      .mockResolvedValueOnce(new Response(JSON.stringify({
        data: { csrf_token: 'x'.repeat(40) }
      }), { status: 200 }))
      .mockResolvedValueOnce(new Response(JSON.stringify({
        error: { code: 'permission_denied', message: '不能修改自己的角色或状态' }
      }), { status: 403 }));

    await expect(updateOptions({ article_can_edit: 'true' }, fetchFn)).rejects.toMatchObject({
      status: 403,
      code: 'permission_denied',
      message: '不能修改自己的角色或状态'
    });
  });

  it('图片列表响应格式不正确时抛错', async () => {
    const fetchFn = vi.fn().mockResolvedValue(new Response(JSON.stringify({ data: { items: 'oops' } }), { status: 200 }));
    await expect(listImages({}, fetchFn)).rejects.toMatchObject({ code: 'invalid_response' });
  });
});
