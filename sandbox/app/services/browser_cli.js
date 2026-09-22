#!/usr/bin/env node
/**
 * 沙箱浏览器自动化 CLI。
 *
 * 由沙箱 Python 服务（BrowserService）以子进程方式调用，一个动作一次进程：
 *   node browser_cli.js '{"op":"navigate","url":"https://example.com"}'
 *
 * 约定：
 * 1. 入参为单个 JSON 字符串（argv[2]），字段使用 snake_case；
 * 2. stdout 只输出一行 JSON 结果，便于上层严格解析；stderr 仅用于意外崩溃；
 * 3. 通过 CDP 复用 Supervisor 托管的 Chrome，进程退出不关闭浏览器；
 * 4. 每个动作执行期间会收集页面控制台日志并追加到 CONSOLE_LOG，
 *    供 console_view 动作回溯（进程退出后监听失效，仅覆盖动作执行窗口）。
 */

const fs = require('fs');
const path = require('path');

// CDP 调试地址：Chrome 监听 8222（只绑回环），Node CLI 与 Chrome 同容器直连
const CDP_ADDRESS = process.env.SANDBOX_CDP_ADDRESS || 'http://127.0.0.1:8222';
// 控制台日志文件，与 Python 侧 BROWSER_CONSOLE_LOG 保持一致
const CONSOLE_LOG = process.env.SANDBOX_CONSOLE_LOG || '/tmp/browser/console.log';

const DEFAULT_TIMEOUT = 15000; // Playwright 默认动作超时
const NAV_TIMEOUT = 30000; // 页面导航超时
const MAX_TEXT = 120; // 元素文本截断长度
const MAX_LOGS = 200; // 单次动作收集的日志条数上限
const MAX_LOG_ITEM = 500; // 单条日志截断长度
const MAX_EXEC_RESULT = 8000; // console_exec 返回值截断长度
const MAX_VIEW_LINES = 1000; // console_view 最多回传行数
const MAX_SNAPSHOT_ELEMENTS = 200; // snapshot 最多回传元素数，超出截断并标记 truncated
// console_view 返回日志的总字节上限（超出从最旧一行丢弃），可用 SANDBOX_MAX_VIEW_BYTES 覆盖，
// 便于契约测试构造 >64KB 管道缓冲的大输出，验证 writeResult「刷盘回调再退出」不会截断
const MAX_VIEW_BYTES = parseInt(process.env.SANDBOX_MAX_VIEW_BYTES || '20000', 10);

// 可交互元素选择器：快照编号与点击/输入定位共用同一集合，保证 index 语义一致
const INTERACTIVE_SELECTOR = [
  'a[href]',
  'button',
  'input',
  'textarea',
  'select',
  '[role="button"]',
  '[role="link"]',
  '[role="tab"]',
  '[role="menuitem"]',
  '[role="checkbox"]',
  '[contenteditable="true"]',
  '[onclick]',
].join(',');

/** 业务动作错误：页面元素找不到、参数缺失等，上层应回 4xx 让模型自行修正 */
class ActionError extends Error {}

/**
 * 输出单行 JSON 结果并结束进程。
 *
 * 两个约束决定了不能写完就退出：
 * 1. 结果可能大于管道缓冲区（64KB，snapshot 元素多或 console_view 回传长日志时），
 *    此时 stdout 写入是异步的，立即 process.exit() 会把输出截断，上层只能拿到半截
 *    JSON 并报"返回了非法结果"；必须在写入回调里退出。
 * 2. 本进程持有一条 CDP 长连接，事件循环不会被清空，因此也不能只设 process.exitCode
 *    等待自然退出——那样进程会永久挂起，直到上层超时后才被 kill。
 */
function writeResult(text, code) {
  process.stdout.write(text, () => process.exit(code));
}

/** 输出成功结果并结束进程 */
function succeed(payload) {
  writeResult(JSON.stringify(Object.assign({ ok: true }, payload)) + '\n', 0);
}

/**
 * 输出失败结果并以非 0 退出码结束。
 * kind=action 表示页面/参数层面的失败（4xx），kind=env 表示沙箱环境不可用（5xx），
 * 上层据此决定是把错误回灌给模型还是上报服务异常。
 */
function fail(message, kind) {
  writeResult(JSON.stringify({ ok: false, kind: kind || 'env', error: String(message) }) + '\n', 1);
}

/** 延迟加载 Playwright：模块缺失时给出可诊断的错误而非进程级堆栈 */
function loadPlaywright() {
  try {
    return require('playwright-core');
  } catch (err) {
    throw new Error('沙箱缺少 playwright-core 依赖，请检查镜像构建: ' + err.message);
  }
}

/** 生成页面上下文内联的 DOM 辅助函数源码 */
function domHelpers() {
  return `
    const SEL = ${JSON.stringify(INTERACTIVE_SELECTOR)};
    const visible = (el) => {
      const rect = el.getBoundingClientRect();
      if (rect.width <= 0 || rect.height <= 0) return false;
      const style = window.getComputedStyle(el);
      return style.visibility !== 'hidden' && style.display !== 'none' && style.opacity !== '0';
    };
    const nodes = () => Array.from(document.querySelectorAll(SEL)).filter(visible);
    const label = (el) => ((el.innerText || el.value || el.getAttribute('aria-label')
      || el.getAttribute('placeholder') || el.getAttribute('title') || '') + '')
      .replace(/\\s+/g, ' ').trim().slice(0, ${MAX_TEXT});
  `;
}

/** 生成"按编号取元素"的页面表达式，编号来源与快照完全一致 */
function elementExpr(index) {
  return `(() => { ${domHelpers()} return nodes()[${index}] || null; })()`;
}

/** 生成页面快照表达式：可交互元素列表 + 滚动位置 */
function snapshotExpr() {
  return `(() => { ${domHelpers()}
    const elements = nodes().map((el, index) => {
      const item = { index: index, tag: el.tagName.toLowerCase() };
      const type = el.getAttribute('type');
      if (type) item.type = type;
      const text = label(el);
      if (text) item.text = text;
      const href = el.getAttribute('href');
      if (href) item.href = href;
      if (typeof el.value === 'string' && el.value) item.value = el.value.slice(0, ${MAX_TEXT});
      if (el.checked) item.checked = true;
      if (el.disabled) item.disabled = true;
      return item;
    });
    return {
      elements: elements,
      scroll: {
        y: Math.round(window.scrollY),
        height: document.body ? document.body.scrollHeight : 0,
        viewport: window.innerHeight
      }
    };
  })()`;
}

/** 收集动作执行窗口内的控制台日志，并追加到日志文件 */
function attachLogCollector(page) {
  const logs = [];
  const push = (line) => {
    if (logs.length >= MAX_LOGS) return;
    logs.push(String(line).slice(0, MAX_LOG_ITEM));
  };
  page.on('console', (msg) => push(`[${msg.type()}] ${msg.text()}`));
  page.on('pageerror', (err) => push(`[pageerror] ${err.message}`));
  page.on('requestfailed', (req) => push(`[requestfailed] ${req.url()} ${(req.failure() || {}).errorText || ''}`));
  return {
    logs,
    flush: () => {
      if (!logs.length) return;
      try {
        fs.mkdirSync(path.dirname(CONSOLE_LOG), { recursive: true });
        fs.appendFileSync(CONSOLE_LOG, logs.join('\n') + '\n');
      } catch (err) {
        process.stderr.write('write console log failed: ' + err.message + '\n');
      }
    },
  };
}

/** 打开（复用）CDP 页面：优先使用最新打开的标签页，保证与人工/VNC 视角一致 */
async function openPage(chromium) {
  const browser = await chromium.connectOverCDP(CDP_ADDRESS, { timeout: DEFAULT_TIMEOUT });
  const context = browser.contexts()[0] || (await browser.newContext());
  context.setDefaultTimeout(DEFAULT_TIMEOUT);
  const pages = context.pages();
  const page = pages.length > 0 ? pages[pages.length - 1] : await context.newPage();
  await page.bringToFront();
  return page;
}

/** 按编号/选择器定位元素，均支持 actionability 等待 */
async function locate(page, params) {
  if (params.selector) {
    return page.locator(String(params.selector)).first();
  }
  if (typeof params.index === 'number') {
    const handle = await page.evaluateHandle(elementExpr(params.index));
    const element = handle.asElement();
    if (!element) {
      throw new ActionError(`元素编号 ${params.index} 不存在或不可见，请先 snapshot 获取最新元素列表`);
    }
    return element;
  }
  return null;
}

/** 页面摘要：大部分动作都回传 URL 与标题，让模型无需额外 snapshot 即可确认状态 */
async function pageInfo(page) {
  let title = '';
  try {
    title = await page.title();
  } catch (err) {
    title = '';
  }
  return { url: page.url(), title: title };
}

/** navigate：导航超时不视为硬失败，页面可能已可用 */
async function opNavigate(params) {
  const chromium = loadPlaywright().chromium;
  const page = await openPage(chromium);
  const collector = attachLogCollector(page);
  try {
    const warnings = [];
    try {
      await page.goto(String(params.url), { waitUntil: 'domcontentloaded', timeout: NAV_TIMEOUT });
    } catch (err) {
      if (!/Timeout/i.test(err.message)) throw err;
      warnings.push(`导航等待超时（${NAV_TIMEOUT}ms），页面内容可能未完全加载`);
    }
    await page.waitForLoadState('domcontentloaded').catch(() => {});
    const info = await pageInfo(page);
    succeed(Object.assign(info, { warnings: warnings, logs: collector.logs.slice(-20) }));
  } finally {
    collector.flush();
  }
}

/** snapshot：返回带编号的可交互元素列表，编号可直接用于 click/input */
async function opSnapshot() {
  const chromium = loadPlaywright().chromium;
  const page = await openPage(chromium);
  const collector = attachLogCollector(page);
  try {
    const data = await page.evaluate(snapshotExpr());
    // 输出预算：元素数量设上限，超出截断并标记，避免超大页面撑爆管道与上下文
    let truncated = false;
    if (data.elements.length > MAX_SNAPSHOT_ELEMENTS) {
      data.elements = data.elements.slice(0, MAX_SNAPSHOT_ELEMENTS);
      truncated = true;
    }
    const info = await pageInfo(page);
    succeed(Object.assign(info, data, { truncated: truncated, logs: collector.logs.slice(-20) }));
  } finally {
    collector.flush();
  }
}

/** screenshot：PNG 写入指定路径，由上层读取二进制，避免 base64 走 stdout */
async function opScreenshot(params) {
  const chromium = loadPlaywright().chromium;
  const page = await openPage(chromium);
  const collector = attachLogCollector(page);
  try {
    const out = String(params.path || '');
    if (!out) throw new ActionError('screenshot 缺少 path 参数');
    const buffer = await page.screenshot({ fullPage: Boolean(params.full_page) });
    fs.mkdirSync(path.dirname(out), { recursive: true });
    fs.writeFileSync(out, buffer);
    const info = await pageInfo(page);
    succeed(Object.assign(info, { path: out, bytes: buffer.length, logs: collector.logs.slice(-20) }));
  } finally {
    collector.flush();
  }
}

/** click：支持编号、选择器、坐标三种方式 */
async function opClick(params) {
  const chromium = loadPlaywright().chromium;
  const page = await openPage(chromium);
  const collector = attachLogCollector(page);
  try {
    if (typeof params.x === 'number' && typeof params.y === 'number') {
      await page.mouse.click(params.x, params.y);
    } else {
      const target = await locate(page, params);
      if (!target) throw new ActionError('click 需要 index、selector 或坐标之一');
      await target.scrollIntoViewIfNeeded();
      await target.click({ timeout: DEFAULT_TIMEOUT });
    }
    await page.waitForTimeout(500); // 留出页面响应（跳转/弹层）的短暂窗口
    const info = await pageInfo(page);
    succeed(Object.assign(info, { logs: collector.logs.slice(-20) }));
  } finally {
    collector.flush();
  }
}

/** input：编号/选择器走 fill（自动聚焦清空），坐标走鼠标点击+键盘输入 */
async function opInput(params) {
  const chromium = loadPlaywright().chromium;
  const page = await openPage(chromium);
  const collector = attachLogCollector(page);
  try {
    const text = String(params.text == null ? '' : params.text);
    if (typeof params.x === 'number' && typeof params.y === 'number') {
      await page.mouse.click(params.x, params.y);
      if (text) await page.keyboard.type(text);
    } else {
      const target = await locate(page, params);
      if (!target) throw new ActionError('input 需要 index、selector 或坐标之一');
      await target.scrollIntoViewIfNeeded();
      await target.fill(text, { timeout: DEFAULT_TIMEOUT });
    }
    if (params.press_enter) {
      await page.keyboard.press('Enter');
    }
    await page.waitForTimeout(500);
    const info = await pageInfo(page);
    succeed(Object.assign(info, { logs: collector.logs.slice(-20) }));
  } finally {
    collector.flush();
  }
}

/** press_key：模拟按键（如 Enter、Escape、Tab、ArrowDown） */
async function opPressKey(params) {
  const chromium = loadPlaywright().chromium;
  const page = await openPage(chromium);
  const collector = attachLogCollector(page);
  try {
    await page.keyboard.press(String(params.key));
    await page.waitForTimeout(300);
    const info = await pageInfo(page);
    succeed(Object.assign(info, { logs: collector.logs.slice(-20) }));
  } finally {
    collector.flush();
  }
}

/** scroll：按方向滚动一屏或直达端点 */
async function opScroll(params) {
  const chromium = loadPlaywright().chromium;
  const page = await openPage(chromium);
  const collector = attachLogCollector(page);
  try {
    const up = String(params.direction || 'down').toLowerCase() === 'up';
    const toEnd = Boolean(params.to_end);
    const expression = toEnd
      ? up
        ? 'window.scrollTo(0, 0)'
        : 'window.scrollTo(0, document.body.scrollHeight)'
      : up
        ? 'window.scrollBy(0, -window.innerHeight)'
        : 'window.scrollBy(0, window.innerHeight)';
    await page.evaluate(`(() => { ${expression}; })()`);
    await page.waitForTimeout(300);
    const position = await page.evaluate(
      '({ y: Math.round(window.scrollY), height: document.body ? document.body.scrollHeight : 0, viewport: window.innerHeight })'
    );
    const info = await pageInfo(page);
    succeed(Object.assign(info, { scroll: position, logs: collector.logs.slice(-20) }));
  } finally {
    collector.flush();
  }
}

/** console_exec：在页面上下文执行 JavaScript，用于读取页面状态或触发交互 */
async function opConsoleExec(params) {
  const chromium = loadPlaywright().chromium;
  const page = await openPage(chromium);
  const collector = attachLogCollector(page);
  try {
    const code = String(params.javascript || '');
    if (!code.trim()) throw new ActionError('console_exec 缺少 javascript 参数');
    let result = await page.evaluate(`(() => { ${code} })()`);
    let serialized = JSON.stringify(result === undefined ? null : result);
    let truncated = false;
    if (serialized && serialized.length > MAX_EXEC_RESULT) {
      serialized = serialized.slice(0, MAX_EXEC_RESULT);
      truncated = true;
    }
    succeed({ result: serialized, truncated: truncated, logs: collector.logs.slice(-20) });
  } finally {
    collector.flush();
  }
}

/** console_view：读取历史控制台日志尾部，无需连接浏览器 */
function opConsoleView(params) {
  const want = Number(params.max_lines);
  const limit = Number.isFinite(want) && want > 0 ? Math.min(want, MAX_VIEW_LINES) : 100;
  let all = [];
  try {
    all = fs.readFileSync(CONSOLE_LOG, 'utf8').split('\n').filter((line) => line.trim() !== '');
  } catch (err) {
    if (err.code !== 'ENOENT') throw err;
    all = [];
  }
  // 输出预算：在行数上限的基础上，再按总字节数从最旧一行起裁剪，
  // 避免单行超长日志把返回体积撑爆管道与上下文
  let tail = all.slice(-limit);
  let truncated = false;
  while (tail.length > 1 && Buffer.byteLength(tail.join('\n')) > MAX_VIEW_BYTES) {
    tail = tail.slice(1);
    truncated = true;
  }
  succeed({ lines: tail, total: all.length, truncated: truncated });
}

/** 动作分发表：新增动作只需在此登记并在 Python 侧补端点 */
const OPERATIONS = {
  navigate: opNavigate,
  snapshot: opSnapshot,
  screenshot: opScreenshot,
  click: opClick,
  input: opInput,
  press_key: opPressKey,
  scroll: opScroll,
  console_exec: opConsoleExec,
  console_view: opConsoleView,
};

async function main() {
  const raw = process.argv[2];
  if (!raw) {
    fail('缺少入参 JSON', 'action');
    return;
  }
  let params;
  try {
    params = JSON.parse(raw);
  } catch (err) {
    fail('入参 JSON 解析失败: ' + err.message, 'action');
    return;
  }
  const handler = OPERATIONS[String(params.op)];
  if (!handler) {
    fail('不支持的动作: ' + params.op, 'action');
    return;
  }
  await handler(params);
}

main().catch((err) => {
  // 动作类错误（元素找不到、参数缺失）与 Playwright 动作超时归为 4xx，
  // 其余（缺少依赖、CDP 连不上、脚本崩溃）归为环境不可用
  const action = err instanceof ActionError || (err && err.name === 'TimeoutError');
  fail((err && err.message) || err, action ? 'action' : 'env');
});