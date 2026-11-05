import { expect, test, type Page } from '@playwright/test';
import { PLAIN_USER, loginAsPlainUser, requirePlainCredentials } from './support/login.js';
import { api, pick } from './support/api.js';

// 普通用户权限边界验收（可选）。
//
// 此前所有带凭据用例都在管理员身份下跑，只看得到"允许"的一侧。这一组用普通用户
// 验证"拒绝"的一侧：后台页面与后台 API 都必须 403（已登录，不是 401），
// 改/删他人内容必须被拒且数据不变。全程只读，不写入任何数据。
//
// 凭据来自 VOIDXO_PLAIN_USER / VOIDXO_PLAIN_PASSWORD，缺失时整组跳过。

test.use({ trace: 'off', screenshot: 'off', video: 'off' });
test.describe.configure({ mode: 'serial' });

async function currentUserId(page: Page): Promise<number> {
  const me = await api(page, 'GET', '/api/v1/auth/me');
  const id = pick<number>(me.body, ['data', 'user', 'id']) ?? pick<number>(me.body, ['data', 'id']);
  expect(id, `无法取得当前用户 id：${JSON.stringify(me.body)}`).toBeTruthy();
  return id as number;
}

/**
 * 打开账户菜单。SSR 已渲染按钮，但水合完成前点击没有处理器，
 * 因此带重试：未展开就再点一次，直到菜单真正出现。
 */
async function openAccountMenu(page: Page) {
  const trigger = page.locator('button[aria-haspopup="menu"]');
  const menu = page.locator('.menu-panel');
  await expect(trigger).toBeVisible();
  await expect(async () => {
    if ((await trigger.getAttribute('aria-expanded')) !== 'true') {
      await trigger.click();
    }
    await expect(menu).toBeVisible({ timeout: 1_000 });
  }).toPass({ timeout: 15_000 });
  return menu;
}

test.describe('普通用户权限边界（可选）', () => {
  requirePlainCredentials();
  test.setTimeout(120_000);

  test.beforeEach(async ({ page }) => {
    await loginAsPlainUser(page);
  });

  test('P1：账户菜单不出现后台入口，直接访问后台返回 403', async ({ page }) => {
    await page.goto('/');
    const menu = await openAccountMenu(page);
    await expect(menu).toContainText(PLAIN_USER);
    await expect(menu.locator('a[href="/admin"]')).toHaveCount(0);
    await expect(menu).not.toContainText('管理后台');

    for (const path of ['/admin', '/admin/users', '/admin/options']) {
      const response = await page.goto(path);
      expect(response?.status(), `${path} 应对普通用户返回 403`).toBe(403);
    }
  });

  test('P2：后台 API 对普通用户返回 403（已登录，不是 401）', async ({ page }) => {
    const checks: Array<[string, 'GET' | 'PATCH' | 'POST']> = [
      ['/api/v1/admin/stats', 'GET'],
      ['/api/v1/admin/users?page=1&page_size=20', 'GET'],
      ['/api/v1/admin/content/questions?page=1&page_size=20', 'GET'],
      ['/api/v1/admin/topics?page=1&page_size=20', 'GET'],
      ['/api/v1/admin/options', 'GET']
    ];
    for (const [path, method] of checks) {
      const res = await api(page, method, path);
      expect(res.status, `${method} ${path} 应被拒绝`).toBe(403);
      expect(pick<string>(res.body, ['error', 'code']), `${method} ${path} 错误码`).toBe('permission_denied');
    }

    // 写接口同样拒绝，且不应产生副作用。
    const created = await api(page, 'POST', '/api/v1/admin/topics', { name: '越权话题', description: '' });
    expect(created.status).toBe(403);
  });

  test('P3：改/删他人内容被拒绝，内容保持不变', async ({ page }) => {
    const meId = await currentUserId(page);

    const list = await api(page, 'GET', '/api/v1/questions?page=1&page_size=20');
    const items = pick<any[]>(list.body, ['data', 'items']) ?? [];
    const target = items.find((item) => item?.user_id !== meId);
    expect(target, '需要至少一条他人创建的问题才能验证越权').toBeTruthy();

    const before = await api(page, 'GET', `/api/v1/questions/${target!.id}`);
    const beforeTitle = pick<string>(before.body, ['data', 'question', 'title']) ?? target!.title;

    const patch = await api(page, 'PATCH', `/api/v1/questions/${target!.id}`, {
      title: '越权修改标题',
      content_markdown: '越权修改正文'
    });
    expect(patch.status, '普通用户不能编辑他人问题').toBe(403);

    const remove = await api(page, 'DELETE', `/api/v1/questions/${target!.id}`);
    expect(remove.status, '普通用户不能删除他人问题').toBe(403);

    const after = await api(page, 'GET', `/api/v1/questions/${target!.id}`);
    expect(after.status, '内容应仍然存在').toBe(200);
    expect(pick<string>(after.body, ['data', 'question', 'title']), '标题不应被改动').toBe(beforeTitle);
  });

  test('P4：普通用户自己的页面与搜索仍可用', async ({ page }) => {
    for (const path of ['/user', '/messages', '/notifications', '/questions/new']) {
      const response = await page.goto(path);
      expect(response?.status(), `${path} 应对普通用户开放`).toBe(200);
    }

    const search = await api(page, 'GET', '/api/v1/search?q=' + encodeURIComponent('设计'));
    expect(search.status, '登录后搜索应可用').toBe(200);
  });
});
