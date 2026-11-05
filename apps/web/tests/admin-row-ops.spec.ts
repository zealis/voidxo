import { expect, test, type Page } from '@playwright/test';
import { AUTH_USER, loginViaApi, requireCredentials } from './support/login.js';
import { api, pick, pickFirst } from './support/api.js';

// 后台行级操作验收（可选）。
//
// 覆盖此前只有“渲染 + 导航”覆盖的后台动作：
//   U1 自我保护（只读）：改自己的角色、停用自己都必须被后端拒绝；
//   U2 改他人角色（可逆）：改完立即改回，afterEach 再兜底还原；
//   T1 话题全生命周期：新建 → 编辑 → 删除，删除后列表隐藏、接口标记 deleted；
//   O1 配置页无改动时保存按钮禁用（只读）。
//
// 会改动本地开发数据：U2 临时改动 tester 的角色并还原；T1 会留下一条软删除话题
// （后台没有硬删除入口）。凭据缺失时整组跳过。

test.use({ trace: 'off', screenshot: 'off', video: 'off' });
test.describe.configure({ mode: 'serial' });

const TARGET_USER = 'tester';
const TOPIC_PREFIX = 'e2e话题';

type Baseline = { id: number; role: string; status: number };

let baseline: Baseline | null = null;

async function waitForTags(page: Page, tags: string[]): Promise<void> {
  await page.waitForFunction((names) => names.every((tag) => !!customElements.get(tag)), tags, { timeout: 15_000 });
}

/** 等范围内所有 mdui-button 都渲染出内部 button，否则点击不会有任何行为。 */
async function waitForShadowButtons(page: Page, selector: string): Promise<void> {
  await page.waitForFunction(
    (sel) => {
      const hosts = Array.from(document.querySelectorAll(sel)) as HTMLElement[];
      return hosts.length > 0 && hosts.every((host) => !!host.shadowRoot?.querySelector('button'));
    },
    selector,
    { timeout: 15_000 }
  );
}

function primaryRole(roles: string[]): string {
  if (roles?.includes('admin')) return 'admin';
  if (roles?.includes('moderator')) return 'moderator';
  return 'user';
}

async function loadBaseline(page: Page): Promise<Baseline | null> {
  const res = await api(page, 'GET', `/api/v1/admin/users?q=${TARGET_USER}&page=1&page_size=20`);
  const items = pick<any[]>(res.body, ['data', 'items']) ?? [];
  const found = items.find((item) => item?.username === TARGET_USER);
  if (!found) return null;
  return { id: found.id, role: primaryRole(found.roles), status: found.status };
}

/** 还原 tester 的角色与状态；用例中途失败也能保证不留下停用或降级。 */
async function restoreBaseline(page: Page): Promise<void> {
  if (!baseline) return;
  for (let attempt = 0; attempt < 3; attempt += 1) {
    await api(page, 'PATCH', `/api/v1/admin/users/${baseline.id}`, {
      role: baseline.role,
      status: baseline.status
    });
    const after = await loadBaseline(page);
    if (after?.role === baseline.role && after?.status === baseline.status) return;
  }
}

/** 清掉上次失败残留的活动话题（软删除过的无法硬删，只能留在“含已删除”里）。 */
async function sweepTopics(page: Page): Promise<void> {
  const res = await api(page, 'GET', '/api/v1/admin/topics?page=1&page_size=50&include_deleted=1&q=' + encodeURIComponent(TOPIC_PREFIX));
  const items = pick<any[]>(res.body, ['data', 'items']) ?? [];
  for (const item of items) {
    if (!item?.deleted && String(item?.name ?? '').startsWith(TOPIC_PREFIX)) {
      await api(page, 'DELETE', `/api/v1/admin/topics/${item.id}`);
    }
  }
}

test.describe('后台行级操作验收（可选）', () => {
  requireCredentials();
  test.setTimeout(120_000);

  test.beforeEach(async ({ page }) => {
    await loginViaApi(page);
    baseline = await loadBaseline(page);
  });

  test.afterEach(async ({ page }) => {
    await restoreBaseline(page);
    await sweepTopics(page);
  });

  test('U1：管理员不能改自己的角色或停用自己', async ({ page }) => {
    await page.goto(`/admin/users?q=${encodeURIComponent(AUTH_USER)}`);
    await waitForTags(page, ['mdui-button']);

    const row = page.locator('tbody tr', { hasText: AUTH_USER }).first();
    await expect(row).toBeVisible();

    const roleSelect = row.locator('select');
    await expect(roleSelect).toHaveValue('admin');
    await roleSelect.selectOption('user');

    const error = page.locator('.error[role="alert"]');
    await expect(error).toContainText('不能修改自己的角色或状态');
    await expect(row.locator('select')).toHaveValue('admin');

    await waitForShadowButtons(page, 'tbody mdui-button');
    page.once('dialog', (dialog) => dialog.accept());
    await row.locator('mdui-button', { hasText: '停用' }).click();

    await expect(error).toContainText('不能修改自己的角色或状态');
    await expect(row.locator('.badge')).toHaveText('正常');
  });

  test('U2：管理员可以改他人角色并改回', async ({ page }) => {
    expect(baseline, `本地缺少 ${TARGET_USER} 账号，无法验证改角色`).toBeTruthy();
    await page.goto(`/admin/users?q=${TARGET_USER}`);
    await waitForTags(page, ['mdui-button']);

    const roleSelect = page.locator('tbody tr', { hasText: TARGET_USER }).first().locator('select');
    await expect(roleSelect).toHaveValue(baseline!.role);

    const target = baseline!.role === 'moderator' ? 'user' : 'moderator';
    await roleSelect.selectOption(target);

    // selectOption 不等待页面里的异步 PATCH，必须轮询接口确认真正落库。
    await expect
      .poll(async () => (await loadBaseline(page))?.role, { timeout: 15_000, message: '角色变更应落库' })
      .toBe(target);
    await expect(page.locator('tbody tr', { hasText: TARGET_USER }).first().locator('select')).toHaveValue(target);

    await page.locator('tbody tr', { hasText: TARGET_USER }).first().locator('select').selectOption(baseline!.role);
    await expect
      .poll(async () => (await loadBaseline(page))?.role, { timeout: 15_000, message: '角色应还原' })
      .toBe(baseline!.role);
    await expect(page.locator('tbody tr', { hasText: TARGET_USER }).first().locator('select')).toHaveValue(baseline!.role);
  });

  test('T1：话题新建 → 编辑 → 删除', async ({ page }) => {
    const stamp = Date.now();
    const topicName = `${TOPIC_PREFIX}${String(stamp).slice(-10)}`;
    const description = `写操作验收 ${stamp}`;
    const editedDescription = `已编辑 ${stamp}`;

    await page.goto(`/admin/topics?q=${encodeURIComponent(topicName)}`);
    await waitForTags(page, ['mdui-button']);

    // 新建
    await page.locator('form.create input[type="text"]').fill(topicName);
    await page.locator('form.create textarea').fill(description);
    await waitForShadowButtons(page, 'form.create mdui-button');
    await page.locator('form.create mdui-button').click();

    await expect(page.locator('.success[role="status"]')).toHaveText('话题已创建');
    const row = page.locator('tbody tr', { hasText: topicName }).first();
    await expect(row).toBeVisible();
    await expect(row).toContainText(description);

    // 编辑（进入编辑态后名称变成输入框，行不能再按文本定位，改用编辑框反查该行）
    await waitForShadowButtons(page, 'tbody mdui-button');
    await row.locator('mdui-button', { hasText: '编辑' }).click();
    const editRow = page.locator('tbody tr').filter({ has: page.locator('input[aria-label="编辑话题名称"]') });
    await page.locator('input[aria-label="编辑话题名称"]').fill(topicName);
    await page.locator('textarea[aria-label="编辑话题描述"]').fill(editedDescription);
    await waitForShadowButtons(page, 'tbody mdui-button');
    await editRow.locator('mdui-button', { hasText: '保存' }).click();

    await expect(page.locator('.success[role="status"]')).toHaveText('话题已更新');
    await expect(page.locator('tbody tr', { hasText: topicName }).first()).toContainText(editedDescription);

    // 删除（软删，默认列表隐藏）
    page.once('dialog', (dialog) => dialog.accept());
    await page
      .locator('tbody tr', { hasText: topicName })
      .first()
      .locator('mdui-button', { hasText: '删除' })
      .click();
    await expect(page.locator('.success[role="status"]')).toHaveText('话题已删除');
    await expect(page.locator('tbody tr', { hasText: topicName })).toHaveCount(0);

    // 接口核对：仍在“含已删除”里，且 deleted 为真
    const listed = await api(
      page,
      'GET',
      `/api/v1/admin/topics?page=1&page_size=20&include_deleted=1&q=${encodeURIComponent(topicName)}`
    );
    const items = pick<any[]>(listed.body, ['data', 'items']) ?? [];
    const deletedOne = items.find((item) => item?.name === topicName);
    expect(deletedOne, '删除后应仍能在含已删除中查到').toBeTruthy();
    expect(deletedOne?.deleted, '删除应标记为已删除').toBe(true);
  });

  test('O1：配置页无改动时保存按钮禁用', async ({ page }) => {
    await page.goto('/admin/options');
    await waitForTags(page, ['mdui-button']);

    const save = page.locator('mdui-button', { hasText: '保存修改' }).first();
    await expect(save).toHaveJSProperty('disabled', true);
  });
});
