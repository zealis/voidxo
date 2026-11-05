// 真实 API 环境下的逐页 SSR 冒烟回归。
//
// 前提：MySQL / Redis 容器健康，bin/api.exe 与 bin/worker.exe 已启动，
// 前端 dev server（npm run dev）在 5173。API 未就绪时脚本直接退出并提示，
// 不把「降级路径」误报成「真实数据回归」。
//
// 用法：
//   npm run smoke:ssr
//     匿名模式：公开页检查 200，私有页确认正确重定向到登录。
//   VOIDXO_SMOKE_USER=<用户名> VOIDXO_SMOKE_PASSWORD=<密码> npm run smoke:ssr
//     追加管理员会话：先走 CSRF + login，再带 Cookie 覆盖私有页真实数据。
//
// 凭据仅通过环境变量在进程内传递，不写入文件、不打印、不进入任何日志或报告。
// 可通过 VOIDXO_WEB_URL / VOIDXO_API_URL / VOIDXO_SMOKE_IDS 覆盖默认值。
const WEB = process.env.VOIDXO_WEB_URL ?? 'http://127.0.0.1:5173';
const API = process.env.VOIDXO_API_URL ?? 'http://127.0.0.1:8080';
const AUTH_USER = process.env.VOIDXO_SMOKE_USER ?? '';
const AUTH_PASSWORD = process.env.VOIDXO_SMOKE_PASSWORD ?? '';
const AUTH_MODE = Boolean(AUTH_USER && AUTH_PASSWORD);
const IDS = process.env.VOIDXO_SMOKE_IDS
  ? process.env.VOIDXO_SMOKE_IDS.split(',').map((v) => v.trim()).filter(Boolean)
  : [];
const TIMEOUT = 15_000;

const STATIC_PATHS = [
  '/', '/login', '/register', '/forgot-password', '/search', '/user',
  '/messages', '/notifications', '/questions/new',
  '/admin', '/admin/images', '/admin/options', '/admin/reports', '/admin/topics', '/admin/users'
];

// 无凭据时重定向到登录、有凭据时必须渲染真实内容的页面。
// 标题取自各页面 <svelte:head>，用于区分「正常渲染」与「空壳/降级」。
const PRIVATE_TITLES = {
  '/search': '搜索',
  '/user': '个人资料',
  '/messages': '私信',
  '/notifications': '通知',
  '/questions/new': '提问',
  '/admin': '仪表盘 · 管理后台',
  '/admin/users': '用户 · 管理后台',
  '/admin/topics': '话题 · 管理后台',
  '/admin/options': '配置 · 管理后台',
  '/admin/reports': '举报 · 管理后台',
  '/admin/images': '图片 · 管理后台'
};

function cookiePairs(response) {
  const raw = typeof response.headers.getSetCookie === 'function'
    ? response.headers.getSetCookie()
    : [response.headers.get('set-cookie')].filter(Boolean);
  return raw.map((value) => value.split(';')[0].trim()).filter(Boolean);
}

async function get(url, cookie) {
  try {
    const res = await fetch(url, {
      redirect: 'manual',
      signal: AbortSignal.timeout(TIMEOUT),
      headers: {
        accept: 'text/html',
        ...(cookie ? { cookie } : {})
      }
    });
    return { status: res.status, html: await res.text() };
  } catch (error) {
    return { status: 0, html: '', error: String(error) };
  }
}

function verdict({ status, html, error }) {
  if (error || status >= 500 || /Internal Error|500 -|Application error/i.test(html)) return 'FAIL';
  if (status >= 300 && status < 400) return 'REDIRECT';
  if (status === 404) return 'MISSING';
  if (status >= 200 && status < 300 && html.length > 200) return 'OK';
  return 'THIN';
}

// 未显式指定 id 时，从公开列表接口取真实 id，避免用占位 id 误判 404。
async function firstContentId(endpoint) {
  try {
    const res = await fetch(`${API}${endpoint}`, { signal: AbortSignal.timeout(8000) });
    if (!res.ok) return null;
    const json = await res.json();
    return json?.data?.items?.[0]?.id ?? null;
  } catch {
    return null;
  }
}

async function establishSession() {
  const csrfRes = await fetch(`${WEB}/api/v1/auth/csrf`, { signal: AbortSignal.timeout(TIMEOUT) });
  if (!csrfRes.ok) {
    console.error(`CSRF 获取失败：HTTP ${csrfRes.status}`);
    process.exit(3);
  }
  const csrfJson = await csrfRes.json();
  const token = csrfJson?.data?.csrf_token ?? csrfJson?.csrf_token;
  const csrfCookies = cookiePairs(csrfRes);
  if (!token || !csrfCookies.length) {
    console.error('CSRF 响应缺少令牌或 Cookie，无法登录。');
    process.exit(3);
  }

  const loginRes = await fetch(`${WEB}/api/v1/auth/login`, {
    method: 'POST',
    redirect: 'manual',
    signal: AbortSignal.timeout(TIMEOUT),
    headers: {
      'content-type': 'application/json',
      'X-CSRF-Token': token,
      cookie: csrfCookies.join('; ')
    },
    body: JSON.stringify({ login: AUTH_USER, password: AUTH_PASSWORD, device_id: 'ssr-smoke' })
  });
  if (loginRes.status !== 200) {
    const body = await loginRes.json().catch(() => ({}));
    console.error(`登录失败：HTTP ${loginRes.status} ${body?.error?.code ?? ''} ${body?.error?.message ?? ''}`);
    process.exit(3);
  }
  const jar = [...csrfCookies, ...cookiePairs(loginRes)].join('; ');

  const meRes = await fetch(`${WEB}/api/v1/auth/me`, {
    signal: AbortSignal.timeout(TIMEOUT),
    headers: { cookie: jar, accept: 'application/json' }
  });
  if (meRes.status !== 200) {
    console.error(`会话校验失败：/api/v1/auth/me -> ${meRes.status}`);
    process.exit(3);
  }
  const me = await meRes.json();
  const user = me?.data?.user ?? me?.user ?? {};
  const roles = Array.isArray(user.roles) ? user.roles : [];
  console.log(`管理员会话已建立：用户 ${user.username ?? '(未知)'}，角色 ${roles.join('/') || '(无)'}`);
  if (!roles.includes('admin')) {
    console.error('该账号不是 admin，无法覆盖后台页面。');
    process.exit(3);
  }
  return jar;
}

// 私信详情页的 id 是对方 user_id；有真实会话时取真实 id，否则保留占位并标注。
async function resolveCounterpart(jar) {
  try {
    const res = await fetch(`${WEB}/api/v1/inbox?page=1&page_size=1`, {
      signal: AbortSignal.timeout(TIMEOUT),
      headers: { cookie: jar, accept: 'application/json' }
    });
    if (!res.ok) return null;
    const json = await res.json();
    const first = json?.data?.items?.[0];
    if (!first) return null;
    return first.counterpart?.id ?? first.counterpart_id ?? first.user?.id ?? first.user_id ?? null;
  } catch {
    return null;
  }
}

const health = await get(`${API}/health/ready`);
if (health.status !== 200) {
  console.error(`API 未就绪：${API}/health/ready -> ${health.status} ${health.error ?? ''}`);
  console.error('请先启动 bin/api.exe 与 bin/worker.exe，再运行本脚本。');
  process.exit(2);
}
const webProbe = await get(WEB + '/login');
if (webProbe.status === 0) {
  console.error(`前端未就绪：${WEB} -> ${webProbe.error ?? '连接失败'}`);
  console.error('请先运行 npm run dev（或 preview），再运行本脚本。');
  process.exit(2);
}
console.log(`API 就绪（${API}），前端就绪（${WEB}）\n`);

let dynamicPaths = [];
if (IDS.length) {
  dynamicPaths = IDS.flatMap((id) => [`/articles/${id}`, `/questions/${id}`]);
} else {
  const [questionId, articleId] = await Promise.all([
    firstContentId('/api/v1/questions?page=1&page_size=1'),
    firstContentId('/api/v1/articles?page=1&page_size=1')
  ]);
  if (questionId != null) dynamicPaths.push(`/questions/${questionId}`);
  if (articleId != null) dynamicPaths.push(`/articles/${articleId}`);
  if (dynamicPaths.length) console.log(`自动获取真实内容 id：问题 ${questionId}，文章 ${articleId}\n`);
}

let sessionCookie = null;
let conversationPath = '/messages/1';
if (AUTH_MODE) {
  sessionCookie = await establishSession();
  const counterpart = await resolveCounterpart(sessionCookie);
  if (counterpart != null) {
    conversationPath = `/messages/${counterpart}`;
    console.log(`自动获取真实私信对象 id：${counterpart}\n`);
  } else {
    console.log('私信会话列表为空，/messages/{id} 使用占位 id 检查异常兜底路径\n');
  }
}
const paths = [...STATIC_PATHS, ...dynamicPaths, conversationPath];

async function runPaths(cookie) {
  const rows = [];
  for (const path of paths) {
    const result = await get(WEB + path, cookie);
    rows.push({
      path,
      ...result,
      verdict: verdict(result),
      buttons: (result.html.match(/<mdui-button/g) ?? []).length,
      fields: (result.html.match(/<mdui-text-field/g) ?? []).length
    });
  }
  return rows;
}

const anon = await runPaths(null);
const pad = (v, n) => String(v).padEnd(n);
const failures = [];

if (AUTH_MODE) {
  const auth = await runPaths(sessionCookie);
  const authByPath = new Map(auth.map((r) => [r.path, r]));
  console.log(pad('路径', 30), pad('匿名', 8), pad('登录态', 8), pad('结论', 12), pad('字节', 8), '标题');
  for (const a of anon) {
    const b = authByPath.get(a.path);
    const hint = PRIVATE_TITLES[a.path];
    const isPrivate = Object.hasOwn(PRIVATE_TITLES, a.path) || a.path === conversationPath;
    const titleOk = hint ? b.html.includes(hint) : b.html.length > 200;
    let state;
    if (isPrivate) {
      state = b.verdict === 'OK' && titleOk ? 'OK' : 'FAIL';
      if (state === 'FAIL') {
        failures.push(`${a.path}(匿名 ${a.status} → 登录态 ${b.status}${hint && !titleOk ? ' 标题未命中' : ''}${b.error ? ' ' + b.error : ''})`);
      }
    } else {
      state = b.verdict === 'FAIL' || b.verdict === 'THIN' ? 'FAIL' : b.verdict;
      if (state === 'FAIL') failures.push(`${a.path}(登录态 ${b.status})`);
    }
    console.log(
      pad(a.path, 30), pad(a.status, 8), pad(b.status, 8), pad(state, 12),
      pad(b.html.length, 8), hint ? (titleOk ? '命中' : '未命中') : '-'
    );
  }
  const privatePaths = [...Object.keys(PRIVATE_TITLES), conversationPath];
  const beforeRedirect = anon.filter((r) => privatePaths.includes(r.path) && r.verdict === 'REDIRECT').length;
  const privateOk = privatePaths.filter((path) => {
    const r = authByPath.get(path);
    const hint = PRIVATE_TITLES[path];
    return r?.verdict === 'OK' && (!hint || r.html.includes(hint));
  }).length;
  const okCount = [...authByPath.values()].filter((r) => r.verdict === 'OK').length;
  const loggedInRedirects = auth.filter((r) => r.verdict === 'REDIRECT').length;
  console.log(`\n私有页 ${privatePaths.length} 个：登录前 ${beforeRedirect} 个重定向，登录后 ${privateOk} 个 200 且渲染正常`);
  console.log(`全站 ${paths.length} 条路径：登录态 ${okCount} 个 200、${loggedInRedirects} 个已登录跳转、失败 ${failures.length}`);
} else {
  console.log(pad('路径', 34), pad('状态', 6), pad('结论', 10), pad('字节', 8), pad('button', 8), 'field');
  for (const r of anon) {
    console.log(pad(r.path, 34), pad(r.status, 6), pad(r.verdict, 10), pad(r.bytes ?? r.html.length, 8), pad(r.buttons, 8), r.fields);
  }
  for (const r of anon) {
    if (r.verdict === 'FAIL' || r.verdict === 'THIN') failures.push(`${r.path}(${r.status}${r.error ? ' ' + r.error : ''})`);
  }
  const redirected = anon.filter((r) => r.verdict === 'REDIRECT');
  const missing = anon.filter((r) => r.verdict === 'MISSING');
  console.log(`\n合计 ${anon.length}：可通过 ${anon.length - failures.length - redirected.length - missing.length}，需登录 ${redirected.length}，无此数据 ${missing.length}，失败 ${failures.length}`);
  if (redirected.length) console.log('需登录（未覆盖真实数据，可设置管理员凭据复跑）：', redirected.map((r) => r.path).join(', '));
  if (missing.length) console.log('动态 id 占位未命中（可设 VOIDXO_SMOKE_IDS 为真实 id）：', missing.map((r) => r.path).join(', '));
}

if (failures.length) {
  console.error('失败页面：', failures.join(', '));
  process.exit(1);
}
