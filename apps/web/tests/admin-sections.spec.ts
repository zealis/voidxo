import { expect, test } from '@playwright/test';
import { AUTH_USER, loginViaApi, requireCredentials } from './support/login.js';

// 后台 10 个分区的真实浏览器验收（只读）。
// 只做页面渲染、导航与筛选提交，不触发任何写操作。
test.use({ trace: 'off', screenshot: 'off', video: 'off' });

const SECTIONS = [
  { path: '/admin', title: '仪表盘' },
  { path: '/admin/users', title: '用户' },
  { path: '/admin/questions', title: '问题' },
  { path: '/admin/answers', title: '回答' },
  { path: '/admin/articles', title: '文章' },
  { path: '/admin/comments', title: '评论' },
  { path: '/admin/topics', title: '话题' },
  { path: '/admin/reports', title: '举报' },
  { path: '/admin/options', title: '配置' },
  { path: '/admin/images', title: '图片' }
];

test.describe('后台 10 个分区（登录态、只读）', () => {
  requireCredentials();

  test.beforeEach(async ({ page }) => {
    await loginViaApi(page);
  });

  test('逐个分区直接访问：标题正确、无加载失败、无 5xx', async ({ page }) => {
    // 顺序打开 10 个后台页面；dev server 首次编译每个路由较慢，默认 30 秒不够。
    test.setTimeout(90_000);
    const serverErrors: string[] = [];
    page.on('response', (res) => {
      if (res.status() >= 500) serverErrors.push(`${res.status()} ${res.url()}`);
    });

    for (const section of SECTIONS) {
      await page.goto(section.path);
      await expect(page.locator('h1'), section.path).toHaveText(section.title);
      await expect(page.locator('p.error[role="alert"]'), section.path).toHaveCount(0);
    }

    expect(serverErrors).toEqual([]);
  });

  test('通过后台导航走完全部分区', async ({ page }) => {
    await page.goto('/admin');
    const nav = page.locator('nav.admin-nav');
    await expect(nav.getByRole('link')).toHaveCount(SECTIONS.length);

    for (const section of SECTIONS.slice(1)) {
      await nav.getByRole('link', { name: section.title, exact: true }).click();
      await expect(page).toHaveURL(new RegExp(`${section.path}$`));
      await expect(page.locator('h1')).toHaveText(section.title);
      await expect(nav.getByRole('link', { name: section.title, exact: true })).toHaveAttribute(
        'aria-current',
        'page'
      );
    }
  });

  test('用户筛选是只读 GET 提交，参数写入 URL', async ({ page }) => {
    await page.goto('/admin/users');
    // 后台筛选按钮是 mdui-button：必须等自定义元素注册完，否则点击没有提交行为。
    await page.waitForFunction(() => !!customElements.get('mdui-button'), undefined, { timeout: 15_000 });

    await page.locator('input[name="q"]').fill('admin');
    await page.locator('select[name="role"]').selectOption('admin');
    await page.locator('form.filters mdui-button[type="submit"]').click();

    await expect(page).toHaveURL(/[?&]q=admin/);
    await expect(page).toHaveURL(/[?&]role=admin/);
    await expect(page.locator('h1')).toHaveText('用户');
    await expect(page.locator('input[name="q"]')).toHaveValue('admin');
  });

  test('后台栏显示当前登录用户', async ({ page }) => {
    await page.goto('/admin');
    await expect(page.locator('.admin-bar .user')).toHaveText(AUTH_USER);
    await expect(page.locator('.admin-bar')).toContainText('管理后台');
  });
});
