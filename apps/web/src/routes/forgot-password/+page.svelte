<script lang="ts">
  import Button from '#lib/components/ui/Button.svelte';
  import { AuthRequestError } from '#lib/api/auth.js';
  import { resetPassword, sendEmailCode } from '#lib/api/account.js';

  let email = $state('');
  let code = $state('');
  let password = $state('');
  let confirm = $state('');
  let errorMessage = $state('');
  let notice = $state('');
  let loading = $state(false);
  let sending = $state(false);
  let countdown = $state(0);
  let done = $state(false);

  const emailValid = $derived(/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(email.trim()));

  $effect(() => {
    if (countdown <= 0) return;
    const timer = setTimeout(() => {
      countdown -= 1;
    }, 1000);
    return () => clearTimeout(timer);
  });

  async function sendCode() {
    if (sending || countdown > 0) return;
    if (!emailValid) {
      errorMessage = '请先填写正确的邮箱';
      return;
    }
    sending = true;
    errorMessage = '';
    notice = '';
    try {
      await sendEmailCode(email.trim(), 'reset');
      countdown = 60;
      notice = '如果该邮箱已注册，验证码会在几分钟内送达';
    } catch (error) {
      errorMessage = error instanceof AuthRequestError ? error.message : '验证码发送失败';
    } finally {
      sending = false;
    }
  }

  async function submit(event: SubmitEvent) {
    event.preventDefault();
    if (loading) return;
    errorMessage = '';
    if (password !== confirm) {
      errorMessage = '两次输入的密码不一致';
      return;
    }
    loading = true;
    try {
      await resetPassword({ email: email.trim(), code: code.trim(), newPassword: password });
      done = true;
    } catch (error) {
      errorMessage = error instanceof AuthRequestError ? error.message : '密码重置失败，请稍后重试';
    } finally {
      loading = false;
    }
  }
</script>

<svelte:head>
  <title>找回密码 · VOIDXO</title>
</svelte:head>

<header class="app-bar">
  <a class="brand" href="/" aria-label="VOIDXO 首页">VOIDXO</a>
  <nav aria-label="账户">
    <a class="nav-link" href="/login">返回登录</a>
  </nav>
</header>

<main class="auth-page">
  <section class="auth-card" aria-labelledby="reset-title">
    <h1 id="reset-title">找回密码</h1>
    {#if done}
      <p class="form-notice" role="status">密码已重置，全部旧会话已失效。</p>
      <p class="switch"><a href="/login">用新密码登录</a></p>
    {:else}
      <form onsubmit={submit} novalidate>
        <div class="field">
          <label for="email">注册邮箱</label>
          <input id="email" name="email" type="email" autocomplete="email" bind:value={email} required />
        </div>
        <div class="field">
          <label for="code">邮箱验证码</label>
          <div class="code-row">
            <input id="code" name="code" type="text" inputmode="numeric" maxlength="6" autocomplete="one-time-code" bind:value={code} required />
            <Button type="button" variant="outlined" disabled={sending || countdown > 0} onclick={sendCode}>
              {countdown > 0 ? `${countdown} 秒后重发` : sending ? '发送中…' : '发送验证码'}
            </Button>
          </div>
        </div>
        <div class="field">
          <label for="password">新密码</label>
          <input id="password" name="password" type="password" autocomplete="new-password" minlength="8" bind:value={password} required />
          <span class="hint">至少 8 个字符</span>
        </div>
        <div class="field">
          <label for="confirm">确认新密码</label>
          <input id="confirm" name="confirm" type="password" autocomplete="new-password" bind:value={confirm} required />
        </div>
        {#if notice}<p class="form-notice" role="status">{notice}</p>{/if}
        {#if errorMessage}<p class="form-error" role="alert">{errorMessage}</p>{/if}
        <Button type="submit" disabled={loading}>重置密码</Button>
      </form>
    {/if}
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
    width: min(100%, 420px);
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
  .code-row { display: flex; gap: var(--vx-space-2); align-items: center; }
  .code-row input { flex: 1; }
  .hint { color: var(--vx-color-text-secondary); font-size: 11px; }
  .form-error { color: var(--vx-color-error); font-size: 12px; }
  .form-notice { color: var(--vx-color-text-secondary); font-size: 12px; }
  .switch { margin: var(--vx-space-4) 0 0; font-size: 13px; color: var(--vx-color-text-secondary); }
  .switch a { color: var(--vx-color-text-primary); }
</style>
