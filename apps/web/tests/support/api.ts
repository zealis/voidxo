import type { Page } from '@playwright/test';

// 浏览器上下文内的同源 API 调用工具，供需要“先读后写/写完核对”的用例复用。
// Cookie 复用 page 的上下文；写操作自动取 CSRF，不打印任何凭据或响应体。

export type ApiResponse = { status: number; body: unknown };

export function pick<T = unknown>(body: unknown, path: string[]): T | undefined {
  let current: any = body;
  for (const key of path) {
    if (current === null || current === undefined || typeof current !== 'object') return undefined;
    current = current[key];
  }
  return current as T;
}

/** 同一份数据在不同接口上可能多包一层 user / question，按候选路径依次尝试。 */
export function pickFirst<T = unknown>(body: unknown, paths: string[][]): T | undefined {
  for (const path of paths) {
    const value = pick<T>(body, path);
    if (value !== undefined && value !== null) return value;
  }
  return undefined;
}

export async function api(
  page: Page,
  method: 'GET' | 'POST' | 'PATCH' | 'DELETE',
  path: string,
  data?: unknown
): Promise<ApiResponse> {
  const headers: Record<string, string> = { accept: 'application/json' };
  if (method !== 'GET') {
    const csrfRes = await page.request.get('/api/v1/auth/csrf');
    const csrfBody = await csrfRes.json();
    headers['X-CSRF-Token'] = pick<string>(csrfBody, ['data', 'csrf_token']) ?? '';
    if (data !== undefined) headers['content-type'] = 'application/json';
  }
  const response = await page.request.fetch(path, { method, headers, data: data as never });
  let body: unknown = null;
  try {
    body = await response.json();
  } catch {
    body = null;
  }
  return { status: response.status(), body };
}
