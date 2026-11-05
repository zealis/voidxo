import { describe, expect, it, vi } from 'vitest';
import { fetchSiteStats } from './stats';

describe('stats API', () => {
  it('返回聚合数量', async () => {
    const fetchFn = vi.fn().mockResolvedValue(new Response(JSON.stringify({
      data: { stats: { users: 3, questions: 5, answers: 2, articles: 1, comments: 4, topics: 2 } }
    }), { status: 200 }));

    const stats = await fetchSiteStats(fetchFn);
    expect(fetchFn).toHaveBeenCalledWith('/api/v1/stats', { credentials: 'omit' });
    expect(stats?.questions).toBe(5);
  });

  it('接口失败时返回 null 而不抛错', async () => {
    const fetchFn = vi.fn().mockResolvedValue(new Response('{}', { status: 500 }));
    expect(await fetchSiteStats(fetchFn)).toBeNull();
  });

  it('网络异常时返回 null', async () => {
    const fetchFn = vi.fn().mockRejectedValue(new Error('offline'));
    expect(await fetchSiteStats(fetchFn)).toBeNull();
  });
});
