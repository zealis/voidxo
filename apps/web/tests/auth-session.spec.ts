import { expect, test } from '@playwright/test';
import { AUTH_USER, loginViaApi, requireCredentials } from './support/login.js';

// 登录态真实浏览器回归（可选）。
//
// 凭据与登录逻辑在 ./support/login.ts：仅读环境变量、不打印、不回传会话值。
// 未提供凭据时整组跳过；trace / 截图 / 录像关闭，避免登录请求体进入测试产物。

// 必须放在文件顶层：这三项会强制新 worker，放在 describe 内会被 Playwright 拒绝。
test.use({ trace: 'off', screenshot: 'off', video: 'off' });

test.describe('登录态真实浏览器回归（可选）', () => {
  requireCredentials();

  test.beforeEach(async ({ page }) => {
    await loginViaApi(page);
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
    await expect(page.locator('.admin-bar .user')).toHaveText(AUTH_USER);
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
