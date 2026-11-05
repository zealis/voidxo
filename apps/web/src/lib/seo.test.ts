import { describe, expect, it } from 'vitest';
import {
  buildRobotsTxt,
  buildSitemapXml,
  canonicalUrl,
  isCacheablePath,
  isNoindex
} from './seo';

describe('SEO 规则', () => {
  it('私有与编辑页标记为 noindex', () => {
    for (const path of ['/admin', '/admin/users', '/messages', '/messages/2', '/notifications', '/user', '/search', '/login', '/articles/new', '/questions/1/edit', '/register', '/forgot-password']) {
      expect(isNoindex(path), path).toBe(true);
    }
    for (const path of ['/', '/articles', '/articles/1', '/questions/2']) {
      expect(isNoindex(path), path).toBe(false);
    }
  });

  it('canonical 去掉末尾斜杠', () => {
    expect(canonicalUrl('https://example.com', '/articles/')).toBe('https://example.com/articles');
    expect(canonicalUrl('https://example.com/', '/')).toBe('https://example.com/');
  });

  it('只有公开只读页面进入页面缓存', () => {
    expect(isCacheablePath('/')).toBe(true);
    expect(isCacheablePath('/articles')).toBe(true);
    expect(isCacheablePath('/articles/12')).toBe(true);
    expect(isCacheablePath('/questions/3')).toBe(true);
    expect(isCacheablePath('/articles/new')).toBe(false);
    expect(isCacheablePath('/search')).toBe(false);
    expect(isCacheablePath('/admin/users')).toBe(false);
  });

  it('站点地图包含静态页与内容页，并去重', () => {
    const xml = buildSitemapXml({
      origin: 'https://example.com',
      entries: [
        { type: 'article', id: 1, updated_at: '2026-11-05T00:00:00Z' },
        { type: 'question', id: 2, updated_at: 'not-a-date' }
      ]
    });
    expect(xml.startsWith('<?xml version="1.0" encoding="UTF-8"?>')).toBe(true);
    expect(xml).toContain('<loc>https://example.com/</loc>');
    expect(xml).toContain('<loc>https://example.com/articles</loc>');
    expect(xml).toContain('<loc>https://example.com/articles/1</loc>');
    expect(xml).toContain('<lastmod>2026-11-05T00:00:00.000Z</lastmod>');
    expect(xml).toContain('<loc>https://example.com/questions/2</loc>');
    // 无有效时间时不写 lastmod
    expect(xml).not.toContain('<lastmod>Invalid');
  });

  it('robots.txt 禁止私有分区并声明 sitemap', () => {
    const robots = buildRobotsTxt('https://example.com/');
    expect(robots).toContain('User-agent: *');
    expect(robots).toContain('Disallow: /admin');
    expect(robots).toContain('Disallow: /messages');
    expect(robots).toContain('Sitemap: https://example.com/sitemap.xml');
  });
});
