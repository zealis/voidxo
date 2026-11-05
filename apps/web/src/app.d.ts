declare global {
  namespace App {
    // 请求级用户状态将在认证模块接入时放入 Locals，禁止使用模块级全局用户 Store。
    // interface Locals {}
  }
}

export {};
