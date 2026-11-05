import { expect, type Page } from '@playwright/test';

/**
 * 打开账户菜单。SSR 已渲染按钮，但水合完成前点击没有处理器，
 * 因此带重试：未展开就再点一次，直到菜单真正出现。
 */
export async function openAccountMenu(page: Page) {
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
