import { expect, test, type Page } from '@playwright/test';
import { loginViaApi, requireCredentials } from './support/login.js';
import { api, pick, pickFirst } from './support/api.js';

// 写操作真实浏览器验收（可选）。
//
// 会向本地开发库写入内容，因此所有创建物标题统一带 `[e2e] 写操作验收` 前缀，
// 每个用例结束后由 sweep() 通过 API 删除本账号名下带前缀的残留；
// 用例内部也走一遍产品的删除路径，验证级联清理。
// 凭据由 ./support/login.ts 从环境变量读取，缺失时整组跳过。
//
// trace / 截图 / 录像关闭，避免登录请求体与会话值进入测试产物；必须在文件顶层。

test.use({ trace: 'off', screenshot: 'off', video: 'off' });
test.describe.configure({ mode: 'serial' });

const PREFIX = '[e2e] 写操作验收';

/** 等自定义元素注册完成，避免点到未升级的空壳。 */
async function waitForTags(page: Page, tags: string[]): Promise<void> {
  await page.waitForFunction((names) => names.every((tag) => !!customElements.get(tag)), tags, { timeout: 15_000 });
}

/** 等 Lit 首帧把内部 button 渲染出来，否则点击不会有任何行为。 */
async function waitForShadowButton(page: Page, selector: string): Promise<void> {
  await page.waitForFunction(
    (sel) => {
      const host = document.querySelector(sel) as HTMLElement | null;
      return !!host?.shadowRoot?.querySelector('button');
    },
    selector,
    { timeout: 15_000 }
  );
}

async function createQuestionViaApi(page: Page, title: string): Promise<number> {
  const topics = await api(page, 'GET', '/api/v1/topics?page=1&page_size=1');
  const items = pick<any[]>(topics.body, ['data', 'items']) ?? [];
  expect(items.length, '缺少话题时无法创建问题，请先导入基础数据').toBeGreaterThan(0);
  const created = await api(page, 'POST', '/api/v1/questions', {
    title,
    content_markdown: '写操作验收用的临时正文。',
    topic_ids: [items[0].id]
  });
  expect(created.status, JSON.stringify(created.body)).toBeLessThan(300);
  const id = pickFirst<number>(created.body, [
    ['data', 'question', 'id'],
    ['data', 'id']
  ]);
  expect(id, `创建问题应返回 id，实际响应 ${JSON.stringify(created.body)}`).toBeTruthy();
  return id as number;
}

/** 清理本账号名下所有带前缀的残留问题（连同回答与评论）。 */
async function sweep(page: Page): Promise<void> {
  const me = await api(page, 'GET', '/api/v1/auth/me');
  const userId = pickFirst<number>(me.body, [
    ['data', 'user', 'id'],
    ['data', 'id']
  ]);
  if (!userId) return;
  for (let pageNo = 1; pageNo <= 3; pageNo += 1) {
    const list = await api(page, 'GET', `/api/v1/users/${userId}/questions?page=${pageNo}&page_size=50`);
    const items = pick<any[]>(list.body, ['data', 'items']) ?? [];
    for (const item of items) {
      if (String(item?.title ?? '').startsWith(PREFIX)) {
        await api(page, 'DELETE', `/api/v1/questions/${item.id}`);
      }
    }
    if (items.length < 50) break;
  }
}

test.describe('写操作真实浏览器验收（可选）', () => {
  requireCredentials();
  test.setTimeout(120_000);

  test.beforeEach(async ({ page }) => {
    await loginViaApi(page);
  });

  test.afterEach(async ({ page }) => {
    await sweep(page);
  });

  test('W1：提问 → 回答 → 编辑回答 → 评论 → 回复 → 删除问题级联清理', async ({ page }) => {
    const stamp = Date.now();
    const title = `${PREFIX} ${stamp}`;

    // —— 提问 ——
    await page.goto('/questions/new');
    await waitForTags(page, ['mdui-text-field', 'mdui-button']);
    const fields = page.locator('form mdui-text-field');
    await expect(fields).toHaveCount(2);
    await fields.nth(0).locator('input').fill(title);
    await fields.nth(1).locator('textarea').fill('写操作验收用正文，**加粗**测试渲染。');

    const topicInput = page.locator('.topic-option input[type="checkbox"]').first();
    await expect(topicInput).toBeVisible();
    await topicInput.check();

    const publish = page.locator('form mdui-button[type="submit"]').first();
    await waitForShadowButton(page, 'form mdui-button[type="submit"]');
    await publish.click();

    await expect(page).toHaveURL(/\/questions\/\d+$/, { timeout: 20_000 });
    await expect(page.locator('h1.question-title')).toHaveText(title);
    await expect(page.locator('.question .prose')).toContainText('写操作验收用正文');
    await expect(page.locator('.question .prose strong')).toHaveText('加粗');
    const questionId = Number(page.url().match(/\/questions\/(\d+)$/)?.[1] ?? 0);
    expect(questionId).toBeGreaterThan(0);

    // —— 回答 ——
    const answerText = `回答正文 ${stamp}`;
    await page.locator('#answer-content').fill(answerText);
    await waitForShadowButton(page, 'form.answer-form mdui-button[type="submit"]');
    await page.locator('form.answer-form mdui-button[type="submit"]').click();

    const answer = page.locator('.answer-list .answer').first();
    await expect(answer.locator('.answer-prose')).toContainText(answerText, { timeout: 20_000 });
    await expect(page.locator('h2.answers-title')).toHaveText('1 个回答');

    // —— 编辑回答 ——
    await answer.locator('.answer-actions button', { hasText: '编辑' }).click();
    const answerEditor = page.locator('form.answer-edit textarea');
    await expect(answerEditor).toBeVisible();
    const editedText = `回答已编辑 ${stamp}`;
    await answerEditor.fill(editedText);
    await waitForShadowButton(page, 'form.answer-edit mdui-button[type="submit"]');
    await page.locator('form.answer-edit mdui-button[type="submit"]').click();
    await expect(answer.locator('.answer-prose')).toContainText(editedText, { timeout: 20_000 });
    await expect(answer.locator('.answer-prose')).not.toContainText(answerText);

    // —— 评论 + 回复 ——
    const commentText = `评论内容 ${stamp}`;
    await page.locator('#comment-content').fill(commentText);
    await waitForShadowButton(page, 'form.comment-form mdui-button[type="submit"]');
    await page.locator('form.comment-form mdui-button[type="submit"]').click();
    const comment = page.locator('.comment-list > li.comment').first();
    await expect(comment.locator('.comment-content')).toHaveText(commentText, { timeout: 20_000 });

    await comment.locator('button.link', { hasText: '回复' }).click();
    const replyText = `回复内容 ${stamp}`;
    await comment.locator('form.reply-form input').fill(replyText);
    await waitForShadowButton(page, 'li.comment form.reply-form mdui-button[type="submit"]');
    await comment.locator('form.reply-form mdui-button[type="submit"]').click();
    await expect(comment.locator('.reply-list .reply .comment-content')).toHaveText(replyText, { timeout: 20_000 });

    // —— 删除问题（级联回答与评论）——
    page.once('dialog', (dialog) => dialog.accept());
    await page.locator('.question-actions button.text-link.danger', { hasText: '删除' }).click();

    await expect(page).toHaveURL(/\/questions$/, { timeout: 20_000 });
    const deleted = await api(page, 'GET', `/api/v1/questions/${questionId}`);
    expect(deleted.status, '删除后详情接口应返回 404').toBe(404);

    const detail = await page.request.get(`/questions/${questionId}`);
    expect(detail.status(), '删除后详情页应 404').toBe(404);
  });

  test('W2：投票可撤销，净票数回到原值', async ({ page }) => {
    const questionId = await createQuestionViaApi(page, `${PREFIX} 投票 ${Date.now()}`);
    await page.goto(`/questions/${questionId}`);
    await waitForTags(page, ['mdui-button']);

    const vote = page.locator('.question-actions .vote');
    const up = vote.locator('.vote-button').first();
    const score = vote.locator('.vote-score');
    await expect(up).toHaveAttribute('aria-pressed', 'false');
    const before = Number((await score.textContent())?.trim() ?? '0');

    await up.click();
    await expect(up).toHaveAttribute('aria-pressed', 'true');
    await expect(score).toHaveText(String(before + 1));

    await up.click();
    await expect(up).toHaveAttribute('aria-pressed', 'false');
    await expect(score).toHaveText(String(before));
  });

  test('W3：关注可撤销，按钮状态同步', async ({ page }) => {
    const questionId = await createQuestionViaApi(page, `${PREFIX} 关注 ${Date.now()}`);
    await page.goto(`/questions/${questionId}`);
    await waitForTags(page, ['mdui-button']);

    const button = page.locator('.question-actions .follow-button');
    await expect(button).toHaveAttribute('aria-pressed', 'false');
    await expect(button).toContainText('关注');

    await button.click();
    await expect(button).toHaveAttribute('aria-pressed', 'true');
    await expect(button).toContainText('已关注');

    await button.click();
    await expect(button).toHaveAttribute('aria-pressed', 'false');
    await expect(button).toContainText('关注');
  });
});
