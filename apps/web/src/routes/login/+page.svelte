<script lang="ts">
  import { goto } from '$app/navigation';
  import Button from '#lib/components/ui/Button.svelte';
  import { AuthRequestError, loginWithPassword } from '#lib/api/auth.js';

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
      <div class="field">
        <label for="login">用户名或邮箱</label>
        <input id="login" name="login" type="text" autocomplete="username" bind:value={login} required />
      </div>
      <div class="field">
        <label for="password">密码</label>
        <input id="password" name="password" type="password" autocomplete="current-password" bind:value={password} required />
      </div>
      {#if errorMessage}
        <p class="form-error" role="alert">{errorMessage}</p>
      {/if}
      <Button type="submit" disabled={loading}>登录</Button>
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
  .field { display: grid; gap: var(--vx-space-1); }
  label { color: var(--vx-color-text-secondary); font-size: 12px; }
  input {
    min-height: var(--vx-input-height);
    padding: 0 var(--vx-space-3);
    color: var(--vx-color-text-primary);
    background: var(--vx-color-surface);
    border: 1px solid var(--vx-color-border);
    border-radius: var(--vx-radius-control);
  }
  input:focus-visible {
    border-color: var(--vx-color-text-primary);
    outline: 2px solid var(--vx-color-focus);
    outline-offset: 1px;
  }
  .form-error { color: var(--vx-color-error); font-size: 12px; }
  .aux-links { display: flex; justify-content: space-between; margin: var(--vx-space-4) 0 0; font-size: 13px; }
  .aux-links a { color: var(--vx-color-text-secondary); }
  .aux-links a:hover { color: var(--vx-color-text-primary); }
</style>
