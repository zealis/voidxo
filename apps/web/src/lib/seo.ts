// 与 SEO 相关的纯函数集中在此，便于单测与在 hooks、页面之间复用。

// 这些分区属于私有或交互页，不参与索引。
const NOINDEX_PREFIXES = ['/admin', '/messages', '/notifications', '/user', '/search', '/login', '/register', '/forgot-password', '/internal'];

export function isNoindex(pathname: string): boolean {
  if (/\/(new|edit)$/.test(pathname)) {
    return true;
  }
  return NOINDEX_PREFIXES.some((prefix) => pathname === prefix || pathname.startsWith(`${prefix}/`));
}

// 去掉末尾斜杠并拼接站点源，避免同一内容出现多个 canonical。
export function canonicalUrl(origin: string, pathname: string): string {
  const base = origin.replace(/\/+$/, '');
  const path = pathname === '/' ? '/' : pathname.replace(/\/+$/, '');
  return `${base}${path}`;
}

// 只有公开的只读页面才进入页面缓存，登录态与写操作一律直出。
export function isCacheablePath(pathname: string): boolean {
  if (pathname === '/' || pathname === '/articles') {
    return true;
  }
  return /^\/articles\/\d+$/.test(pathname) || /^\/questions\/\d+$/.test(pathname);
}

export type SitemapEntry = {
  type: string;
  id: number;
  updated_at: string;
};

export type SitemapSource = {
  origin: string;
  entries: SitemapEntry[];
  staticPaths?: string[];
};

const STATIC_PATHS = ['/', '/articles'];

function escapeXML(value: string): string {
  return value
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
    .replace(/'/g, '&apos;');
}

function pathForEntry(entry: SitemapEntry): string {
  return entry.type === 'article' ? `/articles/${entry.id}` : `/questions/${entry.id}`;
}

function lastmodFor(value: string): string {
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return '';
  }
  return date.toISOString();
}

// 生成标准 sitemap.xml；同一条目只保留一次，静态页排在最前。
export function buildSitemapXml({ origin, entries, staticPaths = STATIC_PATHS }: SitemapSource): string {
  const urls: string[] = [];
  const seen = new Set<string>();

  for (const path of staticPaths) {
    const loc = canonicalUrl(origin, path);
    if (seen.has(loc)) continue;
    seen.add(loc);
    urls.push(`  <url><loc>${escapeXML(loc)}</loc><changefreq>daily</changefreq></url>`);
  }

  for (const entry of entries) {
    const loc = canonicalUrl(origin, pathForEntry(entry));
    if (seen.has(loc)) continue;
    seen.add(loc);
    const lastmod = lastmodFor(entry.updated_at);
    urls.push(
      lastmod
        ? `  <url><loc>${escapeXML(loc)}</loc><lastmod>${lastmod}</lastmod></url>`
        : `  <url><loc>${escapeXML(loc)}</loc></url>`
    );
  }

  return [
    '<?xml version="1.0" encoding="UTF-8"?>',
    '<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">',
    ...urls,
    '</urlset>',
    ''
  ].join('\n');
}

// robots.txt 只放行公开内容，私有分区与接口全部禁止抓取。
export function buildRobotsTxt(origin: string): string {
  const base = origin.replace(/\/+$/, '');
  return [
    'User-agent: *',
    'Allow: /',
    'Disallow: /admin',
    'Disallow: /api/',
    'Disallow: /health/',
    'Disallow: /internal/',
    'Disallow: /messages',
    'Disallow: /notifications',
    'Disallow: /user',
    'Disallow: /search',
    'Disallow: /login',
    'Disallow: /register',
    'Disallow: /forgot-password',
    '',
    `Sitemap: ${base}/sitemap.xml`,
    ''
  ].join('\n');
}
