import type { Handle } from '@sveltejs/kit/hooks';
import { dev } from '$app/env';
import { API_ORIGIN } from '$app/env/private';
import { getCachedPage, setCachedPage } from '#lib/server/page-cache.js';
import { isCacheablePath } from '#lib/seo.js';

const STRIPPED_REQUEST_HEADERS = [
  'connection',
  'keep-alive',
  'proxy-authenticate',
  'proxy-authorization',
  'te',
  'trailer',
  'transfer-encoding',
  'upgrade',
  'host',
  'content-length',
  // 在配置可信代理之前，不把客户端自带的转发链传给 Go API。
  'forwarded',
  'x-forwarded-for',
  'x-forwarded-host',
  'x-forwarded-proto',
  'x-forwarded-port',
  'x-forwarded-scheme',
  'x-real-ip'
];

type HandleInput = Parameters<Handle>[0];

// handlePage 只缓存匿名访问的公开页 HTML，登录态与写操作直接透传。
async function handlePage(event: HandleInput['event'], resolve: HandleInput['resolve']) {
  const cacheable =
    !dev &&
    event.request.method === 'GET' &&
    isCacheablePath(event.url.pathname) &&
    !event.request.headers.get('cookie') &&
    !event.request.headers.get('authorization');

  if (!cacheable) {
    return resolve(event);
  }

  const cached = getCachedPage(event.url.pathname);
  if (cached) {
    return new Response(cached, {
      status: 200,
      headers: {
        'content-type': 'text/html; charset=utf-8',
        'cache-control': 'public, max-age=0, must-revalidate',
        'x-page-cache': 'hit'
      }
    });
  }

  const response = await resolve(event);
  if (response.status !== 200 || response.headers.has('set-cookie')) {
    return response;
  }
  const contentType = response.headers.get('content-type') ?? '';
  if (!contentType.includes('text/html')) {
    return response;
  }

  const body = await response.text();
  setCachedPage(event.url.pathname, body, PAGE_CACHE_TTL_MS);
  const headers = new Headers(response.headers);
  headers.delete('content-length');
  headers.set('x-page-cache', 'miss');
  return new Response(body, { status: response.status, headers });
}

const HOP_BY_HOP_RESPONSE_HEADERS = [
  'connection',
  'keep-alive',
  'proxy-authenticate',
  'proxy-authorization',
  'te',
  'trailer',
  'transfer-encoding',
  'upgrade',
  'content-length'
];

// 公开页 HTML 缓存时长；内容事件会通过 /internal/revalidate 提前清理。
const PAGE_CACHE_TTL_MS = 60_000;

export const handle: Handle = async ({ event, resolve }) => {
  if (!event.url.pathname.startsWith('/api/') && !event.url.pathname.startsWith('/health/')) {
    return handlePage(event, resolve);
  }

  const headers = new Headers(event.request.headers);
  for (const name of STRIPPED_REQUEST_HEADERS) headers.delete(name);
  // SvelteKit 是 Go API 的可信代理；只转发 adapter 解析出的客户端地址。
  try {
    headers.set('X-Forwarded-For', event.getClientAddress());
    headers.set('X-Forwarded-Proto', event.url.protocol.replace(':', ''));
  } catch {
    // adapter 无法解析客户端地址时，Go 侧回退到连接地址。
  }

  try {
    const response = await fetch(new URL(event.url.pathname + event.url.search, API_ORIGIN), {
      method: event.request.method,
      headers,
      body: event.request.method === 'GET' || event.request.method === 'HEAD' ? undefined : event.request.body,
      // Node.js 要求流式请求体启用 duplex。
      duplex: 'half',
      redirect: 'manual',
      signal: event.request.signal
    } as RequestInit);

    const responseHeaders = new Headers(response.headers);
    for (const name of HOP_BY_HOP_RESPONSE_HEADERS) responseHeaders.delete(name);
    return new Response(response.body, {
      status: response.status,
      statusText: response.statusText,
      headers: responseHeaders
    });
  } catch {
    return new Response(JSON.stringify({ error: { code: 'api_unavailable', message: '服务暂时不可用' } }), {
      status: 502,
      headers: { 'content-type': 'application/json; charset=utf-8', 'cache-control': 'no-store' }
    });
  }
};
