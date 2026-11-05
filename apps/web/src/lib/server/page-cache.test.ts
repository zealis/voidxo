import { afterEach, describe, expect, it, vi } from 'vitest';
import { clearPageCache, getCachedPage, pageCacheSize, setCachedPage } from './page-cache';

afterEach(() => {
  clearPageCache();
  vi.useRealTimers();
});

describe('页面缓存', () => {
  it('写入后可命中，过期后失效', () => {
    vi.useFakeTimers();
    setCachedPage('/articles', '<html>a</html>', 1_000);
    expect(getCachedPage('/articles')).toBe('<html>a</html>');

    vi.advanceTimersByTime(1_001);
    expect(getCachedPage('/articles')).toBeNull();
  });

  it('清理返回被清除的条目数', () => {
    setCachedPage('/', '<html>home</html>', 60_000);
    setCachedPage('/articles', '<html>list</html>', 60_000);
    expect(pageCacheSize()).toBe(2);
    expect(clearPageCache()).toBe(2);
    expect(pageCacheSize()).toBe(0);
  });

  it('超过上限时淘汰最早写入的条目', () => {
    for (let index = 0; index < 205; index += 1) {
      setCachedPage(`/articles/${index}`, `<html>${index}</html>`, 60_000);
    }
    expect(pageCacheSize()).toBeLessThanOrEqual(200);
    expect(getCachedPage('/articles/0')).toBeNull();
    expect(getCachedPage('/articles/204')).toBe(`<html>204</html>`);
  });
});
