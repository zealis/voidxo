import { expect, test, type Page } from '@playwright/test';

// 登录态真实浏览器回归（可选）。
//
// 凭据仅由 VOIDXO_SMOKE_USER / VOIDXO_SMOKE_PASSWORD 环境变量提供，不写入文件、
// 不打印；未提供时整组跳过，默认 `npm run test:e2e` 不需要任何凭据。
// 关闭 trace / 截图 / 录像，避免登录请求体或会话值进入测试产物。
const USER = process.env.VOIDXO_SMOKE_USER ?? '';
const PASSWORD = process.env.VOIDXO_SMOKE_PASSWORD ?? '';

// 必须放在文件顶层：这三项会强制新 worker，放在 describe 内会被 Playwright 拒绝。
test.use({ trace: 'off', screenshot: 'off', video: 'off' });

test.describe('登录态真实浏览器回归（可选）', () => {
  test.skip(!USER || !PASSWORD, '设置 VOIDXO_SMOKE_USER / VOIDXO_SMOKE_PASSWORD 后才会运行');

  // 在页面内同源完成 CSRF + login，Cookie 交给浏览器上下文持有；
  // 只返回状态码，不回传响应体或任何会话值。
  async function login(page: Page): Promise<void> {
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
      { user: USER, password: PASSWORD }
    );
    expect(status.csrf).toBe(200);
    expect(status.login).toBe(200);
  }

  test.beforeEach(async ({ page }) => {
    await login(page);
  });

  test('首页登录态：账户菜单出现，登录/注册 CTA 消失', async ({ page }) => {
    await page.goto('/');
    await expect(page.locator('button[aria-haspopup="menu"]')).toBeVisible();
    await expect(page.locator('a.appbar-action[href="/login"]')).toHaveCount(0);
    await expect(page.locator('a.appbar-action[href="/register"]')).toHaveCount(0);
  });

  test('后台用户分区渲染真实表格，并显示当前登录用户', async ({ page }) => {
    await page.goto('/admin/users');
    await expect(page.locator('h1')).toHaveText('用户');
    await expect(page.locator('table tbody tr').first()).toBeVisible();
    await expect(page.locator('.admin-bar .user')).toHaveText(USER);
    await expect(page.locator('p.error')).toHaveCount(0);
  });

  test('后台分区导航可点击切换，且切换过程无 5xx', async ({ page }) => {
    const serverErrors: string[] = [];
    page.on('response', (res) => {
      if (res.status() >= 500) serverErrors.push(`${res.status()} ${res.url()}`);
    });
    await page.goto('/admin');
    await expect(page.locator('h1')).toHaveText('仪表盘');
    await page.locator('nav.admin-nav').getByRole('link', { name: '话题' }).click();
    await expect(page).toHaveURL(/\/admin\/topics$/);
    await expect(page.locator('h1')).toBeVisible();
    await expect(page.locator('nav.admin-nav a.active')).toHaveText('话题');
    expect(serverErrors).toEqual([]);
  });

  test('私信与通知页在登录态正常渲染，不出现加载失败提示', async ({ page }) => {
    await page.goto('/messages');
    await expect(page.locator('h1')).toHaveText('私信');
    await expect(page.locator('p.error')).toHaveCount(0);
    await page.goto('/notifications');
    await expect(page.locator('h1')).toHaveText('通知');
    await expect(page.locator('p.error')).toHaveCount(0);
  });

  test('登录态搜索页可用，已登录访问认证页会跳转', async ({ page }) => {
    await page.goto('/search?q=VOIDXO');
    await expect(page).toHaveTitle(/搜索/);
    await expect(page.locator('p.error')).toHaveCount(0);
    await page.goto('/login');
    await expect(page).not.toHaveURL(/\/login$/);
  });
});
