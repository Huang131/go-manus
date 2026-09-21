package sandbox

// 沙箱内置 Chrome 的 CDP 调试地址，由 Supervisor 守护进程暴露
const cdpAddress = "http://127.0.0.1:9222"

// playwrightScript Playwright 脚本模板（通过 heredoc 传递，兼容多行脚本）
// 复用 Supervisor 管理的 Chrome，不在脚本内关闭浏览器
const playwrightScript = `cat << 'PLAYWRIGHT_EOF' | node
const { chromium } = require('playwright');
(async () => {
    const browser = await chromium.connectOverCDP('%s');
    const context = browser.contexts()[0] || await browser.newContext();
    const page = context.pages()[0] || await context.newPage();
    try { %s } finally { /* Chrome 由 Supervisor 管理，不在此关闭 */ }
})().catch(err => { console.error(err); process.exitCode = 1; });
PLAYWRIGHT_EOF`

// maxDownloadBytes 下载文件大小上限（64MB），避免一次性读爆 API 进程内存
const maxDownloadBytes = 64 << 20
