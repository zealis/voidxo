import { describe, expect, it, vi } from 'vitest';
import { getQuestion, listQuestions } from './content';

describe('content API', () => {
  it('读取问题列表', async () => {
    const fetchFn = vi.fn().mockResolvedValue(new Response(JSON.stringify({
      data: { items: [{ id: 1, title: '问题' }], page: 1, page_size: 10, has_more: false }
    }), { status: 200 }));
    await expect(listQuestions(1, 10, undefined, fetchFn)).resolves.toMatchObject({
      items: [{ id: 1, title: '问题' }],
      has_more: false
    });
  });

  it('读取问题详情并保留服务端渲染内容', async () => {
    const fetchFn = vi.fn().mockResolvedValue(new Response(JSON.stringify({
      data: { question: { id: 8, title: '问题', content_rendered: '<p>正文</p>' } }
    }), { status: 200 }));
    await expect(getQuestion(8, fetchFn)).resolves.toMatchObject({ id: 8, content_rendered: '<p>正文</p>' });
  });

  it('把 404 转成可展示错误', async () => {
    const fetchFn = vi.fn().mockResolvedValue(new Response(JSON.stringify({
      error: { code: 'content_not_found', message: '内容不存在' }
    }), { status: 404 }));
    await expect(getQuestion(99, fetchFn)).rejects.toMatchObject({ status: 404, code: 'content_not_found' });
  });
});
