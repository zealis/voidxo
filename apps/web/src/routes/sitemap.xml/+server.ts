import { fetchSitemapEntries, type SitemapEntry } from '#lib/api/sitemap.js';
import { buildSitemapXml } from '#lib/seo.js';
import type { RequestHandler } from './$types';

export const GET: RequestHandler = async ({ fetch, url }) => {
  let entries: SitemapEntry[] = [];
  try {
    entries = await fetchSitemapEntries(fetch);
  } catch {
    // 接口暂时不可用时仍输出静态页，避免整站 sitemap 变空。
    entries = [];
  }

  const xml = buildSitemapXml({ origin: url.origin, entries });
  return new Response(xml, {
    headers: {
      'content-type': 'application/xml; charset=utf-8',
      // 允许 CDN 缓存 5 分钟；内容事件会通过 Worker 清理页面缓存。
      'cache-control': 'public, max-age=0, s-maxage=300'
    }
  });
};
