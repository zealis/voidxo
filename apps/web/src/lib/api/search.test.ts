import { describe, expect, it, vi } from 'vitest';
import { search } from './search';

describe('search API', () => {
  it('按关键词与类型查询并归一化空分组', async () => {
    const fetchFn = vi.fn().mockResolvedValue(new Response(JSON.stringify({
      data: {
        query: '技术',
        questions: { items: [{ id: 1, title: '问题' }], page: 1, page_size: 10, has_more: false },
        articles: null,
        topics: { items: null, page: 1, page_size: 10, has_more: false }
      }
    }), { status: 200 }));

    const result = await search('技术', 'question', 1, 10, fetchFn);
    expect(fetchFn).toHaveBeenCalledWith('/api/v1/search?q=%E6%8A%80%E6%9C%AF&type=question&page=1&page_size=10', {
      credentials: 'same-origin'
    });
    expect(result.questions.items).toHaveLength(1);
    expect(result.articles.items).toEqual([]);
    expect(result.topics.items).toEqual([]);
  });

  it('把限流错误转成可展示异常', async () => {
    const fetchFn = vi.fn().mockResolvedValue(new Response(JSON.stringify({
      error: { code: 'search_rate_limited', message: '搜索过于频繁，请稍后再试' }
    }), { status: 429 }));

    await expect(search('技术', 'all', 1, 10, fetchFn)).rejects.toMatchObject({
      status: 429,
      code: 'search_rate_limited'
    });
  });

  it('响应缺少 query 时视为格式错误', async () => {
    const fetchFn = vi.fn().mockResolvedValue(new Response(JSON.stringify({ data: {} }), { status: 200 }));
    await expect(search('技术', 'all', 1, 10, fetchFn)).rejects.toMatchObject({ code: 'invalid_response' });
  });
});
