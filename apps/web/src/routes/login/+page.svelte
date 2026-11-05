<script lang="ts">
  import { goto } from '$app/navigation';
  import VxButton from '#lib/components/ui/VxButton.svelte';
  import VxTextField from '#lib/components/ui/VxTextField.svelte';
  import { AuthRequestError, loginWithPassword } from '#lib/api/auth.js';

  // 渐进式迁移的第一个业务页面：布局、信息密度、文案规则保持不变，
  // 仅把原生 input 与旧 ui/Button 换成 Vx* 封装。旧组件暂保留以便回滚。
  let login = $state('');
  let password = $state('');
  let errorMessage = $state('');
  let loading = $state(false);

  async function submit(event: SubmitEvent) {
    event.preventDefault();
    if (loading) return;
    errorMessage = '';
    loading = true;
    try {
      await loginWithPassword(login.trim(), password);
      await goto('/');
    } catch (error) {
      errorMessage = error instanceof AuthRequestError ? error.message : '登录失败，请稍后重试';
    } finally {
      loading = false;
    }
  }
</script>

<svelte:head>
  <title>登录 · VOIDXO</title>
</svelte:head>

<header class="app-bar">
  <a class="brand" href="/" aria-label="VOIDXO 首页">VOIDXO</a>
  <nav aria-label="账户">
    <a class="nav-link" href="/">返回首页</a>
  </nav>
</header>

<main class="auth-page">
  <section class="auth-card" aria-labelledby="login-title">
    <h1 id="login-title">登录</h1>
    <form onsubmit={submit} novalidate>
      <VxTextField
        label="用户名或邮箱"
        name="login"
        autocomplete="username"
        required
        bind:value={login}
      />
      <VxTextField
        label="密码"
        name="password"
        type="password"
        autocomplete="current-password"
        required
        bind:value={password}
      />
      {#if errorMessage}
        <p class="form-error" role="alert">{errorMessage}</p>
      {/if}
      <VxButton type="submit" disabled={loading}>登录</VxButton>
    </form>
    <p class="aux-links">
      <a href="/register">注册新账号</a>
      <a href="/forgot-password">忘记密码</a>
    </p>
  </section>
</main>

<style>
  .auth-page {
    display: grid;
    place-items: start center;
    min-height: calc(100vh - 56px);
    padding: var(--vx-space-8) var(--vx-space-4);
  }
  .auth-card {
    width: min(100%, 380px);
    padding: var(--vx-space-6);
    background: var(--vx-color-surface);
    border: 1px solid var(--vx-color-border);
    border-radius: var(--vx-radius-card);
  }
  h1 { margin: 0 0 var(--vx-space-6); font-size: 24px; font-weight: 500; }
  form { display: grid; gap: var(--vx-space-4); }
  .form-error { color: var(--vx-color-error); font-size: 12px; }
  .aux-links { display: flex; justify-content: space-between; margin: var(--vx-space-4) 0 0; font-size: 13px; }
  .aux-links a { color: var(--vx-color-text-secondary); }
  .aux-links a:hover { color: var(--vx-color-text-primary); }
</style>
