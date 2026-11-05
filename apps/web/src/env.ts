import { defineEnvVars } from '@sveltejs/kit/env';

export const variables = defineEnvVars({
  REVALIDATE_TOKEN: {
    description: 'Worker 调用 /internal/revalidate 时使用的共享令牌，留空表示禁用该端点',
    schema: (value) => (value ?? '').trim()
  },
  API_ORIGIN: {
    description: 'Go API 内网地址，仅在 SvelteKit 服务端使用',
    schema: (value) => {
      const origin = value ?? 'http://127.0.0.1:8080';
      try {
        return new URL(origin).origin;
      } catch {
        throw new Error('API_ORIGIN 必须是合法的 HTTP(S) 地址');
      }
    }
  }
});
