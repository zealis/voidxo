import { expect, test, type Page } from '@playwright/test';

// 登录态用例共用的凭据与登录逻辑。
// 两套凭据，均只从环境变量读取、不写入文件、不打印：
//   VOIDXO_SMOKE_USER / VOIDXO_SMOKE_PASSWORD —— 管理员，后台与写操作用例使用；
//   VOIDXO_PLAIN_USER / VOIDXO_PLAIN_PASSWORD —— 普通用户，权限边界用例使用。
// 登录在页面内同源完成，只回传状态码，不回传响应体或任何会话值。
export const AUTH_USER = process.env.VOIDXO_SMOKE_USER ?? '';
export const AUTH_PASSWORD = process.env.VOIDXO_SMOKE_PASSWORD ?? '';
export const HAS_CREDENTIALS = Boolean(AUTH_USER && AUTH_PASSWORD);

export const PLAIN_USER = process.env.VOIDXO_PLAIN_USER ?? '';
export const PLAIN_PASSWORD = process.env.VOIDXO_PLAIN_PASSWORD ?? '';
export const HAS_PLAIN_CREDENTIALS = Boolean(PLAIN_USER && PLAIN_PASSWORD);

export const MOD_USER = process.env.VOIDXO_MOD_USER ?? '';
export const MOD_PASSWORD = process.env.VOIDXO_MOD_PASSWORD ?? '';
export const HAS_MOD_CREDENTIALS = Boolean(MOD_USER && MOD_PASSWORD);

export async function loginAs(page: Page, user: string, password: string, device = 'playwright-auth-session'): Promise<void> {
  await page.goto('/login');
  const status = await page.evaluate(
    async ({ login, secret, deviceId }) => {
      const csrfRes = await fetch('/api/v1/auth/csrf', { credentials: 'same-origin' });
      const payload = await csrfRes.json();
      const token = payload?.data?.csrf_token ?? '';
      const loginRes = await fetch('/api/v1/auth/login', {
        method: 'POST',
        credentials: 'same-origin',
        headers: { 'content-type': 'application/json', 'X-CSRF-Token': token },
        body: JSON.stringify({ login, password: secret, device_id: deviceId })
      });
      return { csrf: csrfRes.status, login: loginRes.status };
    },
    { login: user, secret: password, deviceId: device }
  );
  expect(status.csrf).toBe(200);
  expect(status.login).toBe(200);
}

/** 管理员登录。 */
export function loginViaApi(page: Page): Promise<void> {
  return loginAs(page, AUTH_USER, AUTH_PASSWORD);
}

/** 普通用户登录。 */
export function loginAsPlainUser(page: Page): Promise<void> {
  return loginAs(page, PLAIN_USER, PLAIN_PASSWORD, 'playwright-plain-session');
}

/** 版主登录。 */
export function loginAsModerator(page: Page): Promise<void> {
  return loginAs(page, MOD_USER, MOD_PASSWORD, 'playwright-mod-session');
}

/** 在 describe 内调用：缺少管理员凭据时跳过整组。 */
export function requireCredentials(): void {
  test.skip(!HAS_CREDENTIALS, '设置 VOIDXO_SMOKE_USER / VOIDXO_SMOKE_PASSWORD 后才会运行');
}

/** 在 describe 内调用：缺少普通用户凭据时跳过整组。 */
export function requirePlainCredentials(): void {
  test.skip(!HAS_PLAIN_CREDENTIALS, '设置 VOIDXO_PLAIN_USER / VOIDXO_PLAIN_PASSWORD 后才会运行');
}

/** 在 describe 内调用：缺少版主凭据时跳过整组。 */
export function requireModeratorCredentials(): void {
  test.skip(!HAS_MOD_CREDENTIALS, '设置 VOIDXO_MOD_USER / VOIDXO_MOD_PASSWORD 后才会运行');
}
