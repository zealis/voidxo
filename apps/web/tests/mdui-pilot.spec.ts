import { expect, test } from '@playwright/test';

const PILOT = '/mdui-pilot';

// 等自定义元素真正注册完再断言，避免测到未升级的空壳。
async function waitForTags(page: import('@playwright/test').Page, tags: string[]): Promise<void> {
  await page.waitForFunction(
    (names) => names.every((tag) => !!customElements.get(tag)),
    tags,
    { timeout: 15_000 }
  );
}

const BASE_TAGS = ['mdui-button', 'mdui-button-icon', 'mdui-text-field'];

function waitForUpgrade(page: import('@playwright/test').Page): Promise<void> {
  return waitForTags(page, BASE_TAGS);
}

test.describe('mdui 2 试点验收', () => {
  test('M2：禁用 JS 时 SSR 正文仍可读，且输出原生 mdui 标签', async ({ browser }) => {
    const context = await browser.newContext({ javaScriptEnabled: false });
    const page = await context.newPage();
    await page.goto(PILOT);
    await expect(page.locator('h1')).toHaveText('mdui 2 组件试点');
    await expect(page.getByText('本页用于验证 M1–M5 的接入方式', { exact: false })).toBeVisible();
    expect(await page.locator('mdui-button').count()).toBeGreaterThan(0);
    expect(await page.locator('mdui-text-field').count()).toBe(2);
    // 服务端不应注册组件或执行主题函数。
    const html = await page.content();
    expect(html).not.toContain('customElements.define');
    await context.close();
  });

  test('M2/M3：客户端完成升级，首屏主题 class 已就位', async ({ page }) => {
    await page.goto(PILOT);
    await waitForUpgrade(page);
    await expect(page.getByRole('status')).toContainText('已升级');
    const htmlClass = (await page.locator('html').getAttribute('class')) ?? '';
    expect(htmlClass).toMatch(/mdui-theme-(auto|light|dark)/);
  });

  test('M5：表单值同步与提交', async ({ page }) => {
    await page.goto(PILOT);
    await waitForUpgrade(page);

    // mdui 的 shadow root 是 open，Playwright 选择器可穿透到内部 input。
    await page.locator('mdui-text-field[name="displayName"] input').fill('张三');
    await page.locator('mdui-text-field[name="email"] input').fill('zhangsan@example.com');

    // 读回自定义元素的 value property，确认封装层已同步。
    // 页面内其它 mdui 组件也可能渲染自己的 text-field，限定到试点表单内。
    const values = await page.locator('form mdui-text-field').evaluateAll((nodes) =>
      nodes.map((node) => (node as HTMLElement & { value: string }).value)
    );
    expect(values).toEqual(['张三', 'zhangsan@example.com']);

    await page.locator('mdui-button[type="submit"]').click();
    await expect(page.locator('[aria-live="polite"]')).toContainText('张三');
    await expect(page.locator('[aria-live="polite"]')).toContainText('zhangsan@example.com');
  });

  test('M3：深浅色切换写入 mdui 主题 class', async ({ page }) => {
    await page.goto(PILOT);
    await waitForUpgrade(page);
    await page.getByRole('button', { name: '深色' }).click();
    await expect(page.locator('html')).toHaveClass(/mdui-theme-dark/);
    await page.getByRole('button', { name: '浅色' }).click();
    await expect(page.locator('html')).toHaveClass(/mdui-theme-light/);
  });

  test('无障碍：按钮可键盘聚焦、图标按钮有可访问名', async ({ page }) => {
    await page.goto(PILOT);
    await waitForUpgrade(page);
    // customElements.get 只代表注册完成，还要等 Lit 首帧把内部 button 渲染出来。
    // snackbar 内部也有 button-icon，用 aria-label 精确定位试点里的那个。
    const iconSelector = 'mdui-button-icon[aria-label="试点图标按钮"]';
    await page.waitForFunction((selector) => {
      const host = document.querySelector(selector);
      return !!host?.shadowRoot?.querySelector('button');
    }, iconSelector);
    const iconButton = page.locator(iconSelector);
    await expect(iconButton).toHaveAttribute('aria-label', '试点图标按钮');

    // Playwright 的 locator.focus() 对 delegatesFocus 自定义元素不生效，
    // 在页面内直接调用 focus()，分别检查 host 与 shadow 内的原生按钮。
    const focusState = await page.evaluate((selector) => {
      const host = document.querySelector(selector) as HTMLElement;
      host.focus();
      return {
        host: document.activeElement?.tagName?.toLowerCase() ?? '',
        inner: host.shadowRoot?.activeElement?.tagName?.toLowerCase() ?? ''
      };
    }, iconSelector);
    expect(focusState.host).toBe('mdui-button-icon');
    expect(focusState.inner).toBe('button');

    await page.keyboard.press('Enter');
    await expect(page.getByText('点击计数（验证 CustomEvent 转 Svelte 回调）：1')).toBeVisible();
  });

  test('M5：选择类控件（Checkbox / Switch / RadioGroup / Select）值同步', async ({ page }) => {
    await page.goto(PILOT);
    await waitForTags(page, [
      'mdui-checkbox', 'mdui-switch', 'mdui-radio-group', 'mdui-radio',
      'mdui-select', 'mdui-menu-item'
    ]);

    await page.locator('mdui-checkbox').click();
    await page.locator('mdui-switch').click();
    await page.locator('mdui-radio[value="pro"]').click();
    await page.locator('mdui-select').click();
    await page.locator('mdui-menu-item[value="go"]').click();

    const values = page.getByTestId('control-values');
    await expect(values).toContainText('同意：是');
    await expect(values).toContainText('通知：关');
    await expect(values).toContainText('方案：pro');
    await expect(values).toContainText('话题：go');
  });

  test('M6：对话框打开、关闭与状态回写', async ({ page }) => {
    await page.goto(PILOT);
    await waitForTags(page, ['mdui-dialog']);
    await expect(page.getByTestId('dialog-state')).toContainText('关闭');
    await page.getByRole('button', { name: '打开对话框' }).click();
    await expect(page.getByTestId('dialog-state')).toContainText('打开');
    await page.getByRole('button', { name: '取消' }).click();
    await expect(page.getByTestId('dialog-state')).toContainText('关闭');
  });

  test('M6：Snackbar 显示与 Tooltip 悬浮', async ({ page }) => {
    await page.goto(PILOT);
    await waitForTags(page, ['mdui-snackbar', 'mdui-tooltip']);
    await page.getByRole('button', { name: '显示 Snackbar' }).click();
    await expect(page.locator('mdui-snackbar')).toContainText('保存成功');
    await expect.poll(() =>
      page.locator('mdui-snackbar').evaluate((el) => (el as HTMLElement & { open: boolean }).open)
    ).toBe(true);

    await page.getByRole('button', { name: '悬停查看提示' }).hover();
    await expect.poll(() =>
      page.locator('mdui-tooltip').evaluate((el) => (el as HTMLElement & { open: boolean }).open)
    ).toBe(true);
  });

  test('M6：Tabs 切换同步到业务状态', async ({ page }) => {
    await page.goto(PILOT);
    await waitForTags(page, ['mdui-tabs', 'mdui-tab', 'mdui-tab-panel']);
    await expect(page.getByTestId('tab-state')).toContainText('当前选项卡：overview');
    await page.locator('mdui-tab[value="detail"]').click();
    await expect(page.getByTestId('tab-state')).toContainText('当前选项卡：detail');
    await expect(page.locator('mdui-tab-panel[value="detail"]')).toBeVisible();
  });
});
