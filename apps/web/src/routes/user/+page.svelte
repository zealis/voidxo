<script lang="ts">
  import { untrack } from 'svelte';
  import { goto, invalidateAll } from '$app/navigation';
  import { MediaRequestError, uploadAvatar } from '#lib/api/media.js';
  import Avatar from '#lib/components/Avatar.svelte';
  import Button from '#lib/components/ui/Button.svelte';
  import { AuthRequestError, changePassword } from '#lib/api/auth.js';
  import { updateProfile } from '#lib/api/profile.js';
  import type { PageProps } from './$types';

  let { data }: PageProps = $props();
  let avatarUploading = $state(false);
  let avatarError = $state('');
  let avatarMessage = $state('');

  async function handleAvatarChange(event: Event) {
    const input = event.currentTarget as HTMLInputElement;
    const file = input.files?.[0];
    if (!file || avatarUploading) return;
    avatarError = '';
    avatarMessage = '';
    avatarUploading = true;
    try {
      await uploadAvatar(file);
      avatarMessage = '头像已更新';
      await invalidateAll();
    } catch (cause) {
      avatarError = cause instanceof MediaRequestError ? cause.message : '头像上传失败';
    } finally {
      avatarUploading = false;
      input.value = '';
    }
  }

  const initial = untrack(() => data.profile);
  let headline = $state(initial?.headline ?? '');
  let bio = $state(initial?.bio ?? '');
  let blog = $state(initial?.blog ?? '');
  let company = $state(initial?.company ?? '');
  let place = $state(initial?.location ?? '');
  let saving = $state(false);
  let message = $state('');
  let errorMessage = $state('');
  let currentPassword = $state('');
  let newPassword = $state('');
  let confirmPassword = $state('');
  let passwordSaving = $state(false);
  let passwordError = $state('');

  async function save(event: SubmitEvent) {
    event.preventDefault();
    if (saving || !data.profile) return;
    saving = true;
    message = '';
    errorMessage = '';
    try {
      await updateProfile({ headline, bio, blog, company, location: place });
      message = '已保存';
    } catch (error) {
      if (error instanceof AuthRequestError && error.status === 401) {
        await goto('/login');
        return;
      }
      errorMessage = error instanceof AuthRequestError ? error.message : '保存失败，请稍后重试';
    } finally {
      saving = false;
    }
  }

  async function updatePassword(event: SubmitEvent) {
    event.preventDefault();
    if (passwordSaving) return;
    passwordError = '';
    if (newPassword !== confirmPassword) {
      passwordError = '两次输入的新密码不一致';
      return;
    }
    passwordSaving = true;
    try {
      await changePassword(currentPassword, newPassword);
      await goto('/login');
    } catch (error) {
      if (error instanceof AuthRequestError && error.status === 401) {
        await goto('/login');
        return;
      }
      passwordError = error instanceof AuthRequestError ? error.message : '密码更新失败，请稍后重试';
    } finally {
      passwordSaving = false;
    }
  }
</script>

<svelte:head>
  <title>个人资料 · VOIDXO</title>
</svelte:head>

<header class="app-bar">
  <a class="brand" href="/" aria-label="VOIDXO 首页">VOIDXO</a>
  <nav aria-label="账户">
    <a class="nav-link" href="/">返回首页</a>
  </nav>
</header>

<main class="page-container">
  <section class="profile-panel" aria-labelledby="profile-title">
    <div class="profile-heading">
      <div class="avatar-block">
        <Avatar
          name={data.profile?.username ?? '?'}
          avatarKey={data.profile?.avatar_key ?? ''}
          size={64}
          alt="当前头像"
        />
        <label class="avatar-upload">
          更换头像
          <input type="file" accept="image/png,image/jpeg" onchange={handleAvatarChange} />
        </label>
      </div>
      <div>
        <h1 id="profile-title">{data.profile?.username ?? '个人资料'}</h1>
        <p>{data.profile?.email ?? ''}</p>
        {#if avatarError}<p class="form-error" role="alert">{avatarError}</p>{/if}
        {#if avatarMessage}<p class="success-message" role="status">{avatarMessage}</p>{/if}
      </div>
    </div>

    {#if data.loadError}
      <div class="error-message" role="alert">
        <span>资料加载失败，请刷新重试。</span>
        <Button variant="outlined" onclick={() => window.location.reload()}>重试</Button>
      </div>
    {:else if data.profile}
      <form onsubmit={save} novalidate>
        <div class="field">
          <label for="headline">一句话介绍</label>
          <input id="headline" maxlength="40" bind:value={headline} />
        </div>
        <div class="field">
          <label for="bio">个人简介</label>
          <textarea id="bio" maxlength="160" rows="3" bind:value={bio}></textarea>
        </div>
        <div class="field">
          <label for="blog">个人主页</label>
          <input id="blog" type="url" maxlength="255" bind:value={blog} />
        </div>
        <div class="field">
          <label for="company">公司名称</label>
          <input id="company" maxlength="255" bind:value={company} />
        </div>
        <div class="field">
          <label for="location">地址</label>
          <input id="location" maxlength="255" bind:value={place} />
        </div>
        <div class="form-actions">
          <Button type="submit" disabled={saving}>保存</Button>
          {#if message}<span class="success-message" role="status">{message}</span>{/if}
          {#if errorMessage}<span class="form-error" role="alert">{errorMessage}</span>{/if}
        </div>
      </form>
      <section class="password-panel" aria-labelledby="password-title">
        <h2 id="password-title">修改密码</h2>
        <form onsubmit={updatePassword} novalidate>
          <div class="field">
            <label for="current-password">当前密码</label>
            <input id="current-password" type="password" autocomplete="current-password" bind:value={currentPassword} required />
          </div>
          <div class="field">
            <label for="new-password">新密码</label>
            <input id="new-password" type="password" minlength="8" maxlength="128" autocomplete="new-password" bind:value={newPassword} required />
          </div>
          <div class="field">
            <label for="confirm-password">确认新密码</label>
            <input id="confirm-password" type="password" minlength="8" maxlength="128" autocomplete="new-password" bind:value={confirmPassword} required />
          </div>
          <div class="form-actions">
            <Button type="submit" disabled={passwordSaving}>更新密码</Button>
            {#if passwordError}<span class="form-error" role="alert">{passwordError}</span>{/if}
          </div>
        </form>
      </section>
    {/if}
  </section>
</main>

<style>
  .profile-panel { max-width: 560px; padding: var(--vx-space-6) 0; }
  .profile-heading { margin-bottom: var(--vx-space-6); }
  h1 { margin: 0 0 var(--vx-space-1); font-size: 24px; font-weight: 500; }
  .profile-heading p { color: var(--vx-color-text-secondary); }
  form { display: grid; gap: var(--vx-space-4); }
  .field { display: grid; gap: var(--vx-space-1); }
  label { color: var(--vx-color-text-secondary); font-size: 12px; }
  input, textarea {
    width: 100%;
    padding: 8px 12px;
    color: var(--vx-color-text-primary);
    font: inherit;
    background: var(--vx-color-surface);
    border: 1px solid var(--vx-color-border);
    border-radius: var(--vx-radius-control);
    resize: vertical;
  }
  input { min-height: var(--vx-input-height); }
  input:focus-visible, textarea:focus-visible {
    border-color: var(--vx-color-text-primary);
    outline: 2px solid var(--vx-color-focus);
    outline-offset: 1px;
  }
  .form-actions { display: flex; align-items: center; gap: var(--vx-space-3); }
  .success-message { color: var(--vx-color-text-secondary); font-size: 12px; }
  .form-error { color: var(--vx-color-error); font-size: 12px; }
  .password-panel {
    margin-top: var(--vx-space-8);
    padding-top: var(--vx-space-6);
    border-top: 1px solid var(--vx-color-border);
  }
  .password-panel h2 { margin: 0 0 var(--vx-space-4); font-size: 16px; font-weight: 500; }
  .avatar-block { display: flex; align-items: center; gap: var(--vx-space-3); }
  .avatar-upload { color: var(--vx-color-text-secondary); font-size: 12px; cursor: pointer; }
  .avatar-upload input { display: none; }
</style>
