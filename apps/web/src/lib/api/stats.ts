export type SiteStats = {
  users: number;
  questions: number;
  answers: number;
  articles: number;
  comments: number;
  topics: number;
};

type Payload = {
  data?: { stats?: SiteStats };
};

// 公开统计接口；失败时返回 null，由页面决定是否展示。
export async function fetchSiteStats(fetchFn: typeof fetch = fetch): Promise<SiteStats | null> {
  try {
    const response = await fetchFn('/api/v1/stats', { credentials: 'omit' });
    if (!response.ok) return null;
    const payload = (await response.json()) as Payload;
    return payload.data?.stats ?? null;
  } catch {
    return null;
  }
}
