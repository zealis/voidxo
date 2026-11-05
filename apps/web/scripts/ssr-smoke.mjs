// 真实 API 环境下的逐页 SSR 冒烟回归。
//
// 前提：MySQL / Redis 容器健康，bin/api.exe 与 bin/worker.exe 已启动，
// 前端 dev server（npm run dev）在 5173。API 未就绪时脚本直接退出并提示，
// 不把「降级路径」误报成「真实数据回归」。
//
// 用法：node ./scripts/ssr-smoke.mjs
// 可通过 VOIDXO_WEB_URL / VOIDXO_API_URL / VOIDXO_SMOKE_IDS 覆盖默认值。
const WEB = process.env.VOIDXO_WEB_URL ?? 'http://127.0.0.1:5173';
const API = process.env.VOIDXO_API_URL ?? 'http://127.0.0.1:8080';
const IDS = process.env.VOIDXO_SMOKE_IDS
  ? process.env.VOIDXO_SMOKE_IDS.split(',').map((v) => v.trim()).filter(Boolean)
  : [];

const STATIC_PATHS = [
  '/', '/login', '/register', '/forgot-password', '/search', '/user',
  '/messages', '/notifications', '/questions/new',
  '/admin', '/admin/images', '/admin/options', '/admin/reports', '/admin/topics', '/admin/users'
];

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
dynamicPaths.push('/messages/1');

async function get(url) {
  try {
    const res = await fetch(url, {
      redirect: 'manual',
      signal: AbortSignal.timeout(15_000),
      headers: { accept: 'text/html' }
    });
    return { status: res.status, html: await res.text() };
  } catch (error) {
    return { status: 0, html: '', error: String(error) };
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
console.log(`API 就绪（${API}），前端就绪（${WEB}），开始逐页冒烟\n`);

const rows = [];
for (const path of [...STATIC_PATHS, ...dynamicPaths]) {
  const { status, html, error } = await get(WEB + path);
  const buttons = (html.match(/<mdui-button/g) ?? []).length;
  const fields = (html.match(/<mdui-text-field/g) ?? []).length;
  const serverError = /Internal Error|500 -|Application error/i.test(html);
  let verdict;
  if (status >= 500 || serverError) verdict = 'FAIL';
  else if (status >= 300 && status < 400) verdict = 'REDIRECT';
  else if (status === 404) verdict = 'MISSING';
  else if (status >= 200 && status < 300 && html.length > 200) verdict = 'OK';
  else verdict = 'THIN';
  rows.push({ path, status, verdict, bytes: html.length, buttons, fields, error });
}

const pad = (v, n) => String(v).padEnd(n);
console.log(pad('路径', 34), pad('状态', 6), pad('结论', 10), pad('字节', 8), pad('button', 8), 'field');
for (const r of rows) {
  console.log(pad(r.path, 34), pad(r.status, 6), pad(r.verdict, 10), pad(r.bytes, 8), pad(r.buttons, 8), r.fields);
}

const failed = rows.filter((r) => r.verdict === 'FAIL' || r.verdict === 'THIN');
const redirected = rows.filter((r) => r.verdict === 'REDIRECT');
const missing = rows.filter((r) => r.verdict === 'MISSING');
console.log(`\n合计 ${rows.length}：可通过 ${rows.length - failed.length - redirected.length - missing.length}，需登录 ${redirected.length}，无此数据 ${missing.length}，失败 ${failed.length}`);
if (redirected.length) console.log('需登录（未覆盖真实数据）：', redirected.map((r) => r.path).join(', '));
if (missing.length) console.log('动态 id 占位未命中（可设 VOIDXO_SMOKE_IDS 为真实 id）：', missing.map((r) => r.path).join(', '));
if (failed.length) {
  console.error('失败页面：', failed.map((r) => `${r.path}(${r.status}${r.error ? ' ' + r.error : ''})`).join(', '));
  process.exit(1);
}
