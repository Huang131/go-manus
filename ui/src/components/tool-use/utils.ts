import type { ToolEvent } from '@/lib/api/types'

export type ToolKind =
  | 'message'
  | 'bash'
  | 'file'
  | 'search'
  | 'browser'
  | 'mcp'
  | 'a2a'
  | 'default'

export function getArg(args: Record<string, unknown>, ...keys: string[]): string {
  if (!args || typeof args !== 'object') return ''
  for (const k of keys) {
    const v = args[k]
    if (typeof v === 'string') return v
  }
  return ''
}

export function truncate(s: string, max: number): string {
  return s.length > max ? s.slice(0, max) + '…' : s
}

export function getToolKind(data: ToolEvent | null | undefined): ToolKind {
  if (!data) return 'default'
  const name = (data.name ?? '').toLowerCase()
  const fn = (data.function ?? '').toLowerCase()

  if (data.function === 'message_notify_user' || data.function === 'message_ask_user') {
    return 'message'
  }
  if (name === 'shell' || name.includes('bash') || fn === 'shell_execute' || fn === 'run' || fn === 'execute' || fn === 'run_command') {
    return 'bash'
  }
  if (name === 'file' || name.includes('file')) {
    return 'file'
  }
  if (name === 'mcp' || name.startsWith('mcp_')) {
    return 'mcp'
  }
  // 剔除原先fn中关于search_web的包含匹配, 避免mcp工具中也存在搜索工具，识别错误
  if (fn === 'search_web' || name === 'search') {
    return 'search'
  }
  if (name === 'browser' || name.includes('browser') || fn.startsWith('browser_')) {
    return 'browser'
  }
  if (name === 'a2a' || name.includes('a2a')) {
    return 'a2a'
  }
  return 'default'
}

/**
 * 根据工具 name/function 和 args 生成事件列表中的人性化提示
 */
export function getFriendlyToolLabel(data: ToolEvent | null | undefined): string {
  if (!data) return '—'
  const name = (data.name ?? '').toLowerCase()
  const fn = (data.function ?? '').toLowerCase()
  const args = data.args && typeof data.args === 'object' ? data.args : {}

  if (data.function === 'message_notify_user' || data.function === 'message_ask_user') {
    const text = typeof args.text === 'string' ? args.text : ''
    return text || '—'
  }

  const filepath = getArg(args, 'filepath', 'path', 'pathname')
  const dirPath = getArg(args, 'dir_path', 'directory', 'dir')
  const query = getArg(args, 'query', 'q')
  const command = getArg(args, 'command', 'cmd', 'script')
  const url = getArg(args, 'url', 'href', 'link')
  const key = getArg(args, 'key')
  // 内置工具（shell/file/browser）是 action 分发模型：动作名由 args.action 承载，
  // function 恒为工具名，因此优先按 action 匹配，其次兼容旧的 *_xxx 函数名。
  const action = getArg(args, 'action')

  if (name === 'file') {
    switch (action || fn) {
      case 'read':
      case 'read_file':
        return filepath ? `正在读取文件 ${truncate(filepath, 60)}` : '正在读取文件'
      case 'write':
      case 'write_file':
        return filepath ? `正在写入文件 ${truncate(filepath, 60)}` : '正在写入文件'
      case 'replace':
      case 'replace_in_file':
        return filepath ? `正在替换文件内容 ${truncate(filepath, 60)}` : '正在替换文件内容'
      case 'search':
      case 'search_in_file':
        return filepath ? `正在在文件中搜索 ${truncate(filepath, 60)}` : '正在在文件中搜索'
      case 'find_files':
        return dirPath ? `正在查找文件 ${truncate(dirPath, 60)}` : '正在查找文件'
      case 'list':
      case 'list_files':
        return dirPath ? `正在列出目录 ${truncate(dirPath, 60)}` : '正在列出目录'
      case 'exists':
        return filepath ? `正在检查文件是否存在 ${truncate(filepath, 60)}` : '正在检查文件'
      case 'delete':
        return filepath ? `正在删除文件 ${truncate(filepath, 60)}` : '正在删除文件'
      default:
        return filepath ? `正在访问文件 ${truncate(filepath, 60)}` : dirPath ? `正在访问目录 ${truncate(dirPath, 60)}` : '正在访问文件'
    }
  }

  if (name === 'browser' || fn.startsWith('browser_')) {
    switch (action || fn) {
      case 'navigate':
      case 'browser_navigate':
      case 'browser_restart':
        return url ? `正在打开页面 ${truncate(url, 80)}` : '正在打开页面'
      case 'snapshot':
      case 'view':
      case 'browser_view':
        return '正在查看当前页面'
      case 'click':
      case 'browser_click':
        return '正在点击页面元素'
      case 'input':
      case 'browser_input':
        return '正在输入内容'
      case 'press_key':
      case 'browser_press_key':
        return key ? `正在按键 ${key}` : '正在按键'
      case 'scroll':
        return '正在滚动页面'
      case 'scroll_up':
      case 'browser_scroll_up':
        return '正在向上滚动页面'
      case 'scroll_down':
      case 'browser_scroll_down':
        return '正在向下滚动页面'
      case 'screenshot':
        return '正在截图'
      case 'console_exec':
      case 'browser_console_exec':
        return '正在执行控制台脚本'
      case 'console_view':
      case 'browser_console_view':
        return '正在查看控制台输出'
      default:
        return url ? `正在打开页面 ${truncate(url, 80)}` : '正在使用浏览器访问页面'
    }
  }

  if (name === 'search' || fn === 'search_web' || fn.includes('search_web')) {
    return query ? `正在搜索 ${truncate(query, 60)}` : '正在搜索'
  }

  if (name === 'shell') {
    switch (action || fn) {
      case 'exec':
      case 'shell_execute':
        return command ? `正在执行命令 ${truncate(command, 60)}` : '正在执行命令'
      case 'read':
      case 'shell_read_output':
        return '正在查看命令输出'
      case 'wait':
      case 'shell_wait':
        return '正在等待命令完成'
      case 'write':
      case 'shell_write_input':
        return '正在向命令输入内容'
      case 'kill':
      case 'shell_kill_process':
        return '正在终止进程'
      default:
        return command ? `正在执行命令 ${truncate(command, 60)}` : '正在执行命令'
    }
  }

  if (name.includes('bash') || fn === 'run' || fn === 'execute' || fn === 'run_command') {
    const cmd = command || (typeof args.input === 'string' ? args.input : '')
    return cmd ? `正在执行命令 ${truncate(cmd, 60)}` : '正在执行命令'
  }

  if (name === 'a2a') {
    switch (fn) {
      case 'get_remote_agent_cards':
        return '正在获取远程 Agent 列表'
      case 'call_remote_agent':
        return query ? `正在调用远程 Agent：${truncate(query, 40)}` : '正在调用远程 Agent'
      default:
        return '正在调用 Agent'
    }
  }

  if (name === 'mcp' || name.startsWith('mcp_')) {
    if (fn.includes('search_web') || fn.includes('search')) {
      return query ? `正在搜索 ${truncate(query, 60)}` : '正在搜索'
    }
    return '正在通过 MCP 服务执行操作'
  }

  return '正在执行操作'
}
