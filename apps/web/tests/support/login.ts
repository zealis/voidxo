import { expect, test, type Page } from '@playwright/test';

// 登录态用例共用的凭据与登录逻辑。
// 凭据仅由 VOIDXO_SMOKE_USER / VOIDXO_SMOKE_PASSWORD 提供，不写入文件、不打印；
// 登录在页面内同源完成，只回传状态码，不回传响应体或任何会话值。
export const AUTH_USER = process.env.VOIDXO_SMOKE_USER ?? '';
export const AUTH_PASSWORD = process.env.VOIDXO_SMOKE_PASSWORD ?? '';
export const HAS_CREDENTIALS = Boolean(AUTH_USER && AUTH_PASSWORD);

export async function loginViaApi(page: Page): Promise<void> {
  await page.goto('/login');
  const status = await page.evaluate(
    async ({ user, password }) => {
      const csrfRes = await fetch('/api/v1/auth/csrf', { credentials: 'same-origin' });
      const payload = await csrfRes.json();
      const token = payload?.data?.csrf_token ?? '';
      const loginRes = await fetch('/api/v1/auth/login', {
        method: 'POST',
        credentials: 'same-origin',
        headers: { 'content-type': 'application/json', 'X-CSRF-Token': token },
        body: JSON.stringify({ login: user, password, device_id: 'playwright-auth-session' })
      });
      return { csrf: csrfRes.status, login: loginRes.status };
    },
    { user: AUTH_USER, password: AUTH_PASSWORD }
  );
  expect(status.csrf).toBe(200);
  expect(status.login).toBe(200);
}

/** 在 describe 内调用：缺少凭据时跳过整组。 */
export function requireCredentials(): void {
  test.skip(!HAS_CREDENTIALS, '设置 VOIDXO_SMOKE_USER / VOIDXO_SMOKE_PASSWORD 后才会运行');
}
