import { defineConfig } from '@playwright/test';

// M8：组件验收在真实浏览器进行，jsdom 无法覆盖 Shadow DOM 与自定义元素升级。
// 复用本机 Chrome（channel: 'chrome'），无需下载 Playwright 自带浏览器。
export default defineConfig({
  testDir: './tests',
  timeout: 30_000,
  expect: { timeout: 10_000 },
  fullyParallel: false,
  // 所有用例共用同一个 dev server：多 worker 并行会互相抢路由编译，
  // 曾导致自定义元素注册超过等待上限而随机失败。串行执行换取稳定。
  workers: 1,
  reporter: [['list']],
  use: {
    baseURL: 'http://127.0.0.1:5173',
    channel: 'chrome',
    headless: true,
    trace: 'retain-on-failure',
    screenshot: 'only-on-failure'
  },
  webServer: {
    command: 'npm run dev',
    url: 'http://127.0.0.1:5173',
    reuseExistingServer: true,
    timeout: 60_000
  }
});
