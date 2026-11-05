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
    expect(await page.locator('mdui-text-field').count()).toBe(3);
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
    expect(values).toEqual(['张三', 'zhangsan@example.com', '']);

    await page.locator('mdui-button[type="submit"]').click();
    await expect(page.locator('[aria-live="polite"]')).toContainText('张三');
    await expect(page.locator('[aria-live="polite"]')).toContainText('zhangsan@example.com');
  });

  test('M5：多行文本框（rows）渲染 textarea 并同步值', async ({ page }) => {
    await page.goto(PILOT);
    await waitForUpgrade(page);

    const multiline = page.locator('form mdui-text-field').nth(2);
    const textarea = multiline.locator('textarea');
    await expect(textarea).toHaveCount(1);

    await textarea.fill('第一行\n第二行');
    await expect(multiline).toHaveJSProperty('value', '第一行\n第二行');
    await expect(page.getByTestId('field-values')).toContainText('备注：第一行');
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

  test('M3：禁用 JS 时品牌色板已静态生效，无需等待客户端 setColorScheme', async ({ browser }) => {
    const context = await browser.newContext({ javaScriptEnabled: false });
    const page = await context.newPage();
    await page.goto(PILOT);
    const colors = await page.evaluate(() => {
      const root = getComputedStyle(document.documentElement);
      return {
        primaryLight: root.getPropertyValue('--mdui-color-primary-light').trim(),
        primaryDark: root.getPropertyValue('--mdui-color-primary-dark').trim(),
        vxSurface: root.getPropertyValue('--vx-color-surface').trim()
      };
    });
    // #1a73e8 经 MD3 色板生成后的浅色主色；深色主色必须同时就位以支持主题切换。
    expect(colors.primaryLight).toBe('0, 91, 192');
    expect(colors.primaryDark).toBe('173, 199, 255');
    expect(colors.vxSurface).not.toBe('');
    await context.close();
  });

  test('扩展组件：Chip 选中态、Badge 与 Card 渲染', async ({ page }) => {
    await page.goto(PILOT);
    await waitForTags(page, ['mdui-chip', 'mdui-badge', 'mdui-card']);
    await expect(page.locator('mdui-badge')).toContainText('3');
    await expect(page.locator('mdui-card')).toContainText('卡片内容仍由业务组件控制');
    await expect(page.getByTestId('chip-state')).toContainText('未选中');
    await page.locator('mdui-chip[variant="filter"]').click();
    await expect(page.getByTestId('chip-state')).toContainText('选中');
  });

  test('业务页面迁移：/login 换用 Vx* 后文案与表单同步仍成立', async ({ page }) => {
    await page.goto('/login');
    await waitForTags(page, ['mdui-text-field', 'mdui-button']);
    // 目标边界：布局、信息密度、文案不变。
    await expect(page.locator('h1')).toHaveText('登录');
    await expect(page.getByText('注册新账号')).toBeVisible();
    await expect(page.getByText('忘记密码')).toBeVisible();
    await expect(page.locator('a[href="/"]').first()).toBeVisible();

    await page.locator('mdui-text-field[name="login"] input').fill('alice');
    await page.locator('mdui-text-field[name="password"] input').fill('secret');
    const values = await page.locator('form mdui-text-field').evaluateAll((nodes) =>
      nodes.map((node) => (node as HTMLElement & { value: string }).value)
    );
    expect(values).toEqual(['alice', 'secret']);
    await expect(page.locator('mdui-button[type="submit"]')).toBeVisible();
  });

  test('认证页迁移：/register 文案、字段数量与值同步', async ({ page }) => {
    await page.goto('/register');
    await waitForTags(page, ['mdui-text-field', 'mdui-button']);
    await expect(page.locator('h1')).toHaveText('注册');
    await expect(page.getByText('3 至 20 个字符，可用中英文、数字、下划线与连字符')).toBeVisible();
    await expect(page.getByText('至少 8 个字符')).toBeVisible();
    await expect(page.locator('form mdui-text-field')).toHaveCount(5);
    await page.locator('mdui-text-field[name="username"] input').fill('alice');
    const value = await page
      .locator('mdui-text-field[name="username"]')
      .evaluate((el) => (el as HTMLElement & { value: string }).value);
    expect(value).toBe('alice');
    await expect(page.locator('mdui-button[type="submit"]')).toContainText('注册并登录');
    await expect(page.locator('mdui-button[type="button"]')).toContainText('发送验证码');
  });

  test('认证页迁移：/forgot-password 文案与字段同步', async ({ page }) => {
    await page.goto('/forgot-password');
    await waitForTags(page, ['mdui-text-field', 'mdui-button']);
    await expect(page.locator('h1')).toHaveText('找回密码');
    await expect(page.locator('form mdui-text-field')).toHaveCount(4);
    await page.locator('mdui-text-field[name="code"] input').fill('123456');
    const value = await page
      .locator('mdui-text-field[name="code"]')
      .evaluate((el) => (el as HTMLElement & { value: string }).value);
    expect(value).toBe('123456');
    await expect(page.locator('mdui-button[type="submit"]')).toContainText('重置密码');
  });

  test('批量迁移：认证页禁用 JS 时正文与链接仍可读', async ({ browser }) => {
    const context = await browser.newContext({ javaScriptEnabled: false });
    const page = await context.newPage();
    for (const path of ['/login', '/register', '/forgot-password']) {
      const response = await page.goto(path);
      expect(response?.status()).toBe(200);
      expect(await page.locator('h1').count()).toBe(1);
      expect(await page.locator('mdui-button').count()).toBeGreaterThan(0);
      expect(await page.locator('mdui-text-field').count()).toBeGreaterThan(0);
    }
    await context.close();
  });
});
