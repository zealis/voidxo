import { buildRobotsTxt } from '#lib/seo.js';
import type { RequestHandler } from './$types';

export const GET: RequestHandler = async ({ url }) => {
  return new Response(buildRobotsTxt(url.origin), {
    headers: {
      'content-type': 'text/plain; charset=utf-8',
      'cache-control': 'public, max-age=3600'
    }
  });
};
