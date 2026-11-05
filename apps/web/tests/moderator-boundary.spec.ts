import { expect, test, type Page } from '@playwright/test';
import { MOD_USER, loginAsModerator, requireModeratorCredentials } from './support/login.js';
import { api, pick } from './support/api.js';
import { openAccountMenu } from './support/ui.js';

// 版主（moderator）权限边界验收（可选）。
//
// 产品决定：版主只在前台处理内容，不进入后台。
// 因此这一组锁定"能力有、入口无"的现状，防止权限被意外放大或收窄：
//   M1 权限集合精确匹配 content.moderate + report.manage；
//   M2 账户菜单无后台入口，/admin 系列一律 403；
//   M3 前台他人内容出现编辑/删除入口（content.moderate 生效），但不实际点击；
//   M4 后台内容 API 与举报 API 可用，用户/配置/话题 API 一律 403。
//
// 已知缺口（按产品决定保留）：report.manage 只在 /admin/reports 有页面，
// 版主进不去后台，所以该权限目前没有可用入口，举报由管理员处理。
//
// 全程只读，不点击任何写操作。凭据来自 VOIDXO_MOD_USER / VOIDXO_MOD_PASSWORD。

test.use({ trace: 'off', screenshot: 'off', video: 'off' });
test.describe.configure({ mode: 'serial' });

async function currentUserId(page: Page): Promise<number> {
  const me = await api(page, 'GET', '/api/v1/auth/me');
  const id = pick<number>(me.body, ['data', 'user', 'id']) ?? pick<number>(me.body, ['data', 'id']);
  expect(id, `无法取得当前用户 id：${JSON.stringify(me.body)}`).toBeTruthy();
  return id as number;
}

test.describe('版主权限边界（可选）', () => {
  requireModeratorCredentials();
  test.setTimeout(120_000);

  test.beforeEach(async ({ page }) => {
    await loginAsModerator(page);
  });

  test('M1：权限集合锁定为内容审核与举报处理', async ({ page }) => {
    const me = await api(page, 'GET', '/api/v1/auth/me');
    const permissions =
      pick<string[]>(me.body, ['data', 'user', 'permissions']) ?? pick<string[]>(me.body, ['data', 'permissions']) ?? [];
    expect([...permissions].sort(), '版主权限集合被改动').toEqual(['content.moderate', 'report.manage']);
  });

  test('M2：账户菜单无后台入口，后台页面一律 403', async ({ page }) => {
    await page.goto('/');
    const menu = await openAccountMenu(page);
    await expect(menu).toContainText(MOD_USER);
    await expect(menu.locator('a[href="/admin"]')).toHaveCount(0);
    await expect(menu).not.toContainText('管理后台');

    for (const path of ['/admin', '/admin/reports', '/admin/questions']) {
      const response = await page.goto(path);
      expect(response?.status(), `${path} 应对版主返回 403（无 admin.access）`).toBe(403);
    }
  });

  test('M3：前台他人内容出现编辑与删除入口', async ({ page }) => {
    const meId = await currentUserId(page);
    const list = await api(page, 'GET', '/api/v1/questions?page=1&page_size=20');
    const items = pick<any[]>(list.body, ['data', 'items']) ?? [];
    const target = items.find((item) => item?.user_id !== meId);
    expect(target, '需要至少一条他人创建的问题').toBeTruthy();

    await page.goto(`/questions/${target!.id}`);
    await expect(page.locator('h1.question-title')).toBeVisible();

    const actions = page.locator('.question-actions');
    await expect(actions.locator('a.text-link', { hasText: '编辑' })).toHaveCount(1);
    await expect(actions.locator('button.text-link.danger', { hasText: '删除' })).toHaveCount(1);
  });

  test('M4：后台内容与举报 API 可用，其余后台 API 一律 403', async ({ page }) => {
    // content.moderate：内容列表可读（kind 是单数 question/answer/article/comment）
    const content = await api(page, 'GET', '/api/v1/admin/content/question?page=1&page_size=5');
    expect(content.status, '版主应能读取内容列表').toBe(200);

    // report.manage：举报列表可读（虽然目前没有页面入口）
    const reports = await api(page, 'GET', '/api/v1/reports?page=1&page_size=5');
    expect(reports.status, '版主应能读取举报列表').toBe(200);

    // 用户、配置、话题（content.manage）都不在版主职责内
    const forbidden: Array<[string, 'GET' | 'PATCH' | 'POST']> = [
      ['/api/v1/admin/users?page=1&page_size=5', 'GET'],
      ['/api/v1/admin/options', 'GET'],
      ['/api/v1/admin/stats', 'GET'],
      ['/api/v1/admin/topics?page=1&page_size=5', 'GET']
    ];
    for (const [path, method] of forbidden) {
      const res = await api(page, method, path);
      expect(res.status, `${method} ${path} 应被拒绝`).toBe(403);
      expect(pick<string>(res.body, ['error', 'code']), `${method} ${path} 错误码`).toBe('permission_denied');
    }

    const patchUser = await api(page, 'PATCH', '/api/v1/admin/users/1', { role: 'user' });
    expect(patchUser.status, '版主不能改用户角色').toBe(403);

    const createTopic = await api(page, 'POST', '/api/v1/admin/topics', { name: '版主越权话题', description: '' });
    expect(createTopic.status, '版主没有 content.manage，不能建话题').toBe(403);
  });
});
