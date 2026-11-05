// 进程内页面缓存：只保存匿名访问的公开页 HTML，key 为路径。
// 失效由 Worker 在处理内容事件后调用 /internal/revalidate 触发。
type Entry = {
  body: string;
  expiresAt: number;
};

const MAX_ENTRIES = 200;
const store = new Map<string, Entry>();

export function getCachedPage(key: string): string | null {
  const entry = store.get(key);
  if (!entry) {
    return null;
  }
  if (entry.expiresAt <= Date.now()) {
    store.delete(key);
    return null;
  }
  return entry.body;
}

export function setCachedPage(key: string, body: string, ttlMs: number): void {
  if (store.size >= MAX_ENTRIES) {
    // Map 保持插入顺序，超限时淘汰最早写入的一条。
    const oldest = store.keys().next();
    if (!oldest.done) {
      store.delete(oldest.value);
    }
  }
  store.set(key, { body, expiresAt: Date.now() + ttlMs });
}

// 返回被清除的条目数，便于再生成端点回报处理结果。
export function clearPageCache(): number {
  const size = store.size;
  store.clear();
  return size;
}

export function pageCacheSize(): number {
  return store.size;
}
