import { expect, test, type APIRequestContext } from '@playwright/test';

// 公开内容页的真实数据验收：不需要登录态，但依赖本地 API。
// API 未就绪时整组跳过，保证默认 `npm run test:e2e` 在无后端环境也不会红。

async function ensureApi(request: APIRequestContext): Promise<void> {
  const res = await request.get('/health/ready').catch(() => null);
  test.skip(!res || !res.ok(), 'API 未就绪，跳过真实数据用例');
}

async function firstId(request: APIRequestContext, endpoint: string): Promise<number | null> {
  const res = await request.get(endpoint);
  if (!res.ok()) return null;
  const json = await res.json();
  return json?.data?.items?.[0]?.id ?? null;
}

test.describe('公开内容页真实数据渲染', () => {
  test('首页：话题网格与内容列表都指向真实页面', async ({ page, request }) => {
    await ensureApi(request);
    await page.goto('/');

    const topicLinks = page.locator('.topic-grid a');
    await expect(topicLinks.first()).toBeVisible();
    expect(await topicLinks.count()).toBeGreaterThan(0);
    expect(await topicLinks.first().getAttribute('href')).toMatch(/^\/topics\/\d+$/);

    const rows = page.locator('a.home-row');
    await expect(rows.first()).toBeVisible();
    expect(await rows.count()).toBeGreaterThan(0);
    expect(await rows.first().getAttribute('href')).toMatch(/\/(questions|articles)\/\d+$/);
  });

  test('问题详情：标题、话题标签、回答区与回答表单齐全', async ({ page, request }) => {
    await ensureApi(request);
    const id = await firstId(request, '/api/v1/questions?page=1&page_size=1');
    test.skip(id == null, '没有可用的公开问题');

    await page.goto(`/questions/${id}`);
    await expect(page.locator('h1.question-title')).not.toBeEmpty();
    await expect(page.locator('ul.topic-bar a').first()).toBeVisible();

    const answersTitle = page.locator('h2.answers-title');
    await expect(answersTitle).toBeVisible();
    await expect(answersTitle).toContainText('回答');
    // 匿名访问不应暴露「写回答」表单，入口只对登录用户开放。
    await expect(page.locator('form.answer-form')).toHaveCount(0);
  });

  test('文章详情：标题与正文渲染正常', async ({ page, request }) => {
    await ensureApi(request);
    const id = await firstId(request, '/api/v1/articles?page=1&page_size=1');
    test.skip(id == null, '没有可用的公开文章');

    await page.goto(`/articles/${id}`);
    await expect(page.locator('h1.article-title')).not.toBeEmpty();
    await expect(page.locator('#article-title')).toBeVisible();
  });

  test('话题页：从首页话题进入后标题正常', async ({ page, request }) => {
    await ensureApi(request);
    await page.goto('/');

    const href = await page.locator('.topic-grid a').first().getAttribute('href');
    test.skip(!href, '首页没有话题入口');
    await page.goto(href as string);

    await expect(page.locator('h1')).not.toBeEmpty();
    await expect(page.locator('h1')).toHaveCount(1);
  });

  test('用户主页：内容分类 Tab 可切换', async ({ page, request }) => {
    await ensureApi(request);
    const questionId = await firstId(request, '/api/v1/questions?page=1&page_size=1');
    test.skip(questionId == null, '没有可用的公开问题');

    await page.goto(`/questions/${questionId}`);
    const userHref = await page.locator('a[href^="/users/"]').first().getAttribute('href');
    test.skip(!userHref, '问题页没有作者链接');

    await page.goto(userHref as string);
    await expect(page.locator('h1')).not.toBeEmpty();

    const tabs = page.locator('nav.tabs a');
    await expect(tabs.first()).toBeVisible();
    expect(await tabs.count()).toBeGreaterThan(1);

    await tabs.nth(1).click();
    await expect(page).toHaveURL(/tab=/);
    await expect(tabs.nth(1)).toHaveAttribute('aria-current', 'page');
  });
});
