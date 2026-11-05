export type SitemapEntry = {
  type: string;
  id: number;
  updated_at: string;
};

type Payload = {
  data?: { items?: SitemapEntry[] | null };
};

// 站点地图条目来自 Go API 的公开接口，失败时由调用方决定是否降级为静态页。
export async function fetchSitemapEntries(
  fetchFn: typeof fetch = fetch,
  limit = 5000
): Promise<SitemapEntry[]> {
  const response = await fetchFn(`/api/v1/sitemap?limit=${limit}`, { credentials: 'omit' });
  if (!response.ok) {
    throw new Error(`sitemap request failed with ${response.status}`);
  }
  const payload = (await response.json()) as Payload;
  const items = payload.data?.items;
  return Array.isArray(items) ? items : [];
}
