<script lang="ts">
  import { onMount } from 'svelte';
  import VxButton from '#lib/components/ui/VxButton.svelte';
  import VxIconButton from '#lib/components/ui/VxIconButton.svelte';
  import VxTextField from '#lib/components/ui/VxTextField.svelte';
  import VxCheckbox from '#lib/components/ui/VxCheckbox.svelte';
  import VxSwitch from '#lib/components/ui/VxSwitch.svelte';
  import VxRadioGroup from '#lib/components/ui/VxRadioGroup.svelte';
  import VxSelect from '#lib/components/ui/VxSelect.svelte';
  import VxDialog from '#lib/components/ui/VxDialog.svelte';
  import VxSnackbar from '#lib/components/ui/VxSnackbar.svelte';
  import VxTooltip from '#lib/components/ui/VxTooltip.svelte';
  import VxTabs from '#lib/components/ui/VxTabs.svelte';
  import VxChip from '#lib/components/ui/VxChip.svelte';
  import VxBadge from '#lib/components/ui/VxBadge.svelte';
  import VxCard from '#lib/components/ui/VxCard.svelte';
  import type { VxOption, VxTabItem } from '#lib/components/ui/types.js';
  import { ensureMduiComponents, whenDefined } from '#lib/mdui/registry.js';
  import { applyTheme, isVxTheme, THEME_STORAGE_KEY, type VxTheme } from '#lib/mdui/theme.js';

  let displayName = $state('');
  let email = $state('');
  let note = $state('');
  let submitted = $state('');
  let currentTheme = $state<VxTheme>('auto');
  let upgraded = $state('等待升级');
  let clickCount = $state(0);

  let agreed = $state(false);
  let notifications = $state(true);
  let plan = $state('free');
  let topic = $state('');
  let tabValue = $state('overview');
  let dialogOpen = $state(false);
  let snackbarOpen = $state(false);
  let chipSelected = $state(false);

  const planOptions: VxOption[] = [
    { value: 'free', label: '免费版' },
    { value: 'pro', label: '专业版' },
    { value: 'team', label: '团队版' }
  ];
  const topicOptions: VxOption[] = [
    { value: 'go', label: 'Go 后端' },
    { value: 'svelte', label: 'SvelteKit 前端' },
    { value: 'mysql', label: 'MySQL' },
    { value: 'deprecated', label: '已停用选项', disabled: true }
  ];
  const tabItems: VxTabItem[] = [
    { value: 'overview', label: '概览', content: '概览面板：沿用旧项目的信息密度与文案规则。' },
    { value: 'detail', label: '明细', content: '明细面板：组件形态按 mdui 2 / MD3 落地。' },
    { value: 'about', label: '关于', content: '关于面板：正文为原生 HTML，SSR 可读。' }
  ];

  onMount(() => {
    try {
      const saved = localStorage.getItem(THEME_STORAGE_KEY);
      if (isVxTheme(saved)) currentTheme = saved;
    } catch {
      // 忽略隐私模式。
    }
    void ensureMduiComponents(['button', 'button-icon', 'text-field']).then(async () => {
      await whenDefined('mdui-button');
      upgraded = '已升级（customElements 已注册）';
    });
  });

  function submit(event: SubmitEvent) {
    event.preventDefault();
    submitted = `姓名：${displayName || '（空）'} · 邮箱：${email || '（空）'}`;
  }

  function switchTheme(theme: VxTheme) {
    currentTheme = theme;
    applyTheme(theme);
  }
</script>

<svelte:head>
  <title>mdui 2 组件试点 · VOIDXO</title>
</svelte:head>

<!-- 以下正文全部是原生 HTML：即使 JS 未加载、mdui 未升级，内容与链接依然可读可索引。 -->
<main class="pilot">
  <h1>mdui 2 组件试点</h1>
  <p class="lead">
    本页用于验证 M1–M5 的接入方式：按需导入、SSR 只输出标签、客户端升级、
    Token 映射到 mdui 色板、以及 Vx* 封装层的事件与表单值同步。
    正文与表单标签均为原生 HTML，不依赖组件升级后才可见。
  </p>
  <p class="status" role="status">组件状态：{upgraded}</p>

  <section aria-labelledby="buttons-title">
    <h2 id="buttons-title">按钮与图标按钮</h2>
    <div class="row">
      <VxButton variant="filled" onclick={() => (clickCount += 1)}>主要操作</VxButton>
      <VxButton variant="tonal">次要操作</VxButton>
      <VxButton variant="outlined">描边</VxButton>
      <VxButton variant="text">文本</VxButton>
      <VxButton disabled>禁用</VxButton>
      <VxButton href="/">链接态</VxButton>
      <VxIconButton ariaLabel="试点图标按钮" onclick={() => (clickCount += 1)}>
        <svg viewBox="0 0 24 24" width="20" height="20" aria-hidden="true">
          <path fill="currentColor" d="M12 2 2 12h3v8h6v-6h2v6h6v-8h3z" />
        </svg>
      </VxIconButton>
    </div>
    <p class="hint">点击计数（验证 CustomEvent 转 Svelte 回调）：{clickCount}</p>
  </section>

  <section aria-labelledby="form-title">
    <h2 id="form-title">文本字段与表单提交</h2>
    <form class="demo-form" onsubmit={submit} novalidate>
      <VxTextField label="姓名" name="displayName" autocomplete="name" bind:value={displayName} />
      <VxTextField label="邮箱" name="email" type="email" autocomplete="email" bind:value={email} />
      <VxTextField label="备注（多行）" rows={4} maxlength={200} bind:value={note} />
      <div class="row">
        <VxButton type="submit" variant="filled">提交</VxButton>
        <VxButton type="reset" variant="text" onclick={() => { displayName = ''; email = ''; note = ''; submitted = ''; }}>重置</VxButton>
      </div>
    </form>
    <p class="hint" data-testid="field-values">
      姓名：{displayName || '（空）'} · 邮箱：{email || '（空）'} · 备注：{note || '（空）'}
    </p>
    <p class="hint" aria-live="polite">{submitted || '尚未提交'}</p>
  </section>

  <section aria-labelledby="get-form-title">
    <h2 id="get-form-title">原生 GET 表单提交</h2>
    <form class="get-form" method="get">
      <input type="search" name="probe" value="probe-value" aria-label="GET 表单探针" />
      <VxButton type="submit" variant="outlined">GET 提交</VxButton>
    </form>
    <p class="hint">提交后地址栏应带 ?probe=probe-value</p>
  </section>

  <section aria-labelledby="controls-title">
    <h2 id="controls-title">选择类控件</h2>
    <div class="stack">
      <VxCheckbox bind:checked={agreed} name="agreed">我已阅读并同意条款</VxCheckbox>
      <VxSwitch bind:checked={notifications} name="notifications">接收邮件通知</VxSwitch>
      <VxRadioGroup bind:value={plan} options={planOptions} name="plan" />
      <VxSelect bind:value={topic} options={topicOptions} label="关注话题" name="topic" />
    </div>
    <p class="hint" data-testid="control-values">
      同意：{agreed ? '是' : '否'} · 通知：{notifications ? '开' : '关'} ·
      方案：{plan} · 话题：{topic || '（空）'}
    </p>
  </section>

  <section aria-labelledby="feedback-title">
    <h2 id="feedback-title">对话框、Snackbar 与 Tooltip</h2>
    <div class="row">
      <VxButton onclick={() => (dialogOpen = true)}>打开对话框</VxButton>
      <VxButton variant="tonal" onclick={() => (snackbarOpen = true)}>显示 Snackbar</VxButton>
      <VxTooltip content="这是 mdui 的 tooltip 提示文本">
        <VxButton variant="outlined">悬停查看提示</VxButton>
      </VxTooltip>
    </div>
    <p class="hint" data-testid="dialog-state">对话框状态：{dialogOpen ? '打开' : '关闭'}</p>
  </section>

  <section aria-labelledby="tabs-title">
    <h2 id="tabs-title">选项卡</h2>
    <VxTabs bind:value={tabValue} items={tabItems} />
    <p class="hint" data-testid="tab-state">当前选项卡：{tabValue}</p>
  </section>

  <section aria-labelledby="display-title">
    <h2 id="display-title">标签、徽标与卡片</h2>
    <div class="row">
      <VxChip href="/topics/1">Go 后端</VxChip>
      <VxChip variant="filter" selectable bind:selected={chipSelected}>SvelteKit</VxChip>
      <VxBadge>3</VxBadge>
    </div>
    <VxCard variant="outlined">
      <div class="card-body">卡片内容仍由业务组件控制，页面信息密度不变。</div>
    </VxCard>
    <p class="hint" data-testid="chip-state">筛选标签：{chipSelected ? '选中' : '未选中'}</p>
  </section>

  <section aria-labelledby="theme-title">
    <h2 id="theme-title">深浅色切换</h2>
    <div class="row">
      <VxButton variant={currentTheme === 'auto' ? 'filled' : 'outlined'} onclick={() => switchTheme('auto')}>跟随系统</VxButton>
      <VxButton variant={currentTheme === 'light' ? 'filled' : 'outlined'} onclick={() => switchTheme('light')}>浅色</VxButton>
      <VxButton variant={currentTheme === 'dark' ? 'filled' : 'outlined'} onclick={() => switchTheme('dark')}>深色</VxButton>
    </div>
    <p class="hint">当前主题 class：mdui-theme-{currentTheme}</p>
  </section>

  <p class="foot"><a href="/">返回首页</a></p>
</main>

<VxDialog bind:open={dialogOpen} headline="试点对话框" description="验证打开、关闭与焦点返回。">
  <p>对话框正文使用原生 HTML，服务端可读；关闭后焦点应回到触发按钮。</p>
  {#snippet actions()}
    <VxButton variant="text" onclick={() => (dialogOpen = false)}>取消</VxButton>
    <VxButton variant="filled" onclick={() => (dialogOpen = false)}>确认</VxButton>
  {/snippet}
</VxDialog>

<VxSnackbar
  bind:open={snackbarOpen}
  message="保存成功"
  action="撤销"
  closeable
  onActionClick={() => (snackbarOpen = false)}
/>

<style>
  .pilot { max-width: 760px; margin: 0 auto; padding: var(--vx-space-8) var(--vx-space-4); }
  h1 { margin: 0 0 var(--vx-space-3); font-size: var(--vx-font-size-headline); font-weight: 600; }
  h2 { margin: 0 0 var(--vx-space-4); font-size: var(--vx-font-size-subtitle); font-weight: 600; }
  .lead { color: var(--vx-color-text-secondary); font-size: var(--vx-font-size-body); line-height: 1.7; }
  .status { margin-top: var(--vx-space-3); color: var(--vx-color-primary); font-size: var(--vx-font-size-small); }
  section { margin-top: var(--vx-space-8); padding-top: var(--vx-space-6); border-top: 1px solid var(--vx-color-divider); }
  .row { display: flex; flex-wrap: wrap; align-items: center; gap: var(--vx-space-3); }
  .stack { display: grid; gap: var(--vx-space-4); max-width: 420px; }
  form { display: grid; gap: var(--vx-space-4); max-width: 420px; }
  .hint { margin-top: var(--vx-space-4); color: var(--vx-color-text-tertiary); font-size: var(--vx-font-size-small); }
  .card-body { margin-top: var(--vx-space-4); padding: var(--vx-space-4); font-size: var(--vx-font-size-small); }
  .foot { margin-top: var(--vx-space-8); font-size: var(--vx-font-size-small); }
  .foot a { color: var(--vx-color-primary); }
</style>
