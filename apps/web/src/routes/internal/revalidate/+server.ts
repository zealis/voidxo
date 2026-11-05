import { json } from '@sveltejs/kit';
import { REVALIDATE_TOKEN } from '$app/env/private';
import { clearPageCache } from '#lib/server/page-cache.js';
import type { RequestHandler } from './$types';

// 供 Worker 在内容变更后调用，清空公开页 HTML 缓存。
// 未配置令牌时端点等同于不存在，避免默认暴露内部能力。
export const POST: RequestHandler = async ({ request }) => {
  if (!REVALIDATE_TOKEN) {
    return new Response('Not Found', { status: 404 });
  }
  if (request.headers.get('x-revalidate-token') !== REVALIDATE_TOKEN) {
    return json({ error: 'forbidden' }, { status: 403 });
  }
  const cleared = clearPageCache();
  return json({ cleared });
};
