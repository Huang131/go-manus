/**
 * API 统一响应格式
 */
export type ApiResponse<T = unknown> = {
  code: number;
  msg: string;
  data: T | null;
};

/**
 * 会话状态
 */
export type SessionStatus = "pending" | "running" | "waiting" | "completed";

/**
 * 执行状态
 */
export type ExecutionStatus = "pending" | "running" | "completed" | "failed";

/**
 * 工具事件状态
 */
export type ToolEventStatus = "calling" | "called";

// ==================== 配置模块类型 ====================

/**
 * LLM 模型条目（多模型管理）
 */
export type LLMModel = {
  id: string;
  name: string;
  provider: string;
  base_url: string;
  api_key?: string;
  model_name: string;
  temperature: number;
  max_tokens: number;
  tags: string[];
  is_default: boolean;
  is_enabled: boolean;
  sort_order: number;
  created_at: string;
  updated_at: string;
};

/**
 * 多模型列表响应
 */
export type LLMModelsData = {
  models: LLMModel[];
};

/**
 * 模型连接测试结果
 */
export type LLMModelTestResponse = {
  model_name: string;
  latency_ms: number;
  content: string;
};

/**
 * Agent 通用配置
 */
export type AgentConfig = {
  max_iterations?: number;
  max_retries?: number;
  max_search_results?: number;
  [key: string]: unknown;
};

/**
 * MCP 服务器列表项（GET 响应）
 */
export type ListMCPServerItem = {
  name: string;
  enabled: boolean;
  command?: string;
  args?: string[];
  env?: Record<string, string>;
};

/**
 * MCP 服务器列表响应
 */
export type MCPServersData = {
  servers: ListMCPServerItem[];
};

/**
 * MCP 服务器配置（POST 请求体中单个服务器的配置）
 */
export type MCPServerConfig = {
  name: string;
  enabled?: boolean;
  env?: Record<string, string>;
  command?: string;
  args?: string[];
};

/**
 * MCP 配置（POST 新增 MCP 服务的请求体）
 */
export type MCPConfig = {
  servers: MCPServerConfig[];
};

/**
 * A2A 服务器列表项（GET 响应）
 */
export type ListA2AServerItem = {
  id: string;
  enabled: boolean;
  url: string;
};

/**
 * A2A 服务器列表响应
 */
export type A2AServersData = {
  servers: ListA2AServerItem[];
};

export type A2AConfig = {
  servers: ListA2AServerItem[];
};

/**
 * 新增 A2A 服务器请求参数
 */
export type CreateA2AServerParams = A2AConfig;

// ==================== 文件模块类型 ====================

/**
 * 文件信息
 */
export type FileInfo = {
  id: string;
  filename: string;
  filepath: string;
  key: string;
  extension: string;
  content_type: string;
  size: number;
  [key: string]: unknown;
};

/**
 * 文件上传请求参数
 */
export type FileUploadParams = {
  file: File;
  session_id?: string;
};

// ==================== 会话模块类型 ====================

/**
 * 会话信息
 */
export type Session = {
  id: string;  // 后端返回 id，前端也使用 id 保持一致
  title: string;
  latest_message: string;
  latest_message_at: string;
  status: SessionStatus;
  unread_message_count: number;
  [key: string]: unknown;
};

/**
 * 会话列表响应
 */
export type SessionsData = {
  sessions: Session[];
};

/**
 * 创建会话请求参数
 */
export type CreateSessionParams = {
  title?: string;
  [key: string]: unknown;
};

/**
 * 聊天消息
 */
export type ChatMessage = {
  role: "user" | "assistant" | "system";
  message: string;
  attachments?: Array<{
    file_id: string;
    filename: string;
    [key: string]: unknown;
  }>;
  [key: string]: unknown;
};

/**
 * 聊天请求参数
 * message 为空时用于流式拉取未完成任务的事件列表
 */
export type ChatParams = {
  message?: string;
  attachments?: string[];
  event_id?: string;
  [key: string]: unknown;
};

/**
 * 会话详情（含事件列表，与 chat 流式响应格式一致）
 */
export type SessionDetail = Session & {
  events?: SSEEventData[];
};

/**
 * 计划步骤
 */
export type PlanStep = {
  id: string;
  description: string;
  status: ExecutionStatus;
  [key: string]: unknown;
};

/**
 * 计划事件
 */
export type PlanEvent = {
  steps: PlanStep[];
  [key: string]: unknown;
};

/**
 * 步骤事件
 */
export type StepEvent = {
  id: string;
  status: ExecutionStatus;
  description: string;
  [key: string]: unknown;
};

/**
 * 工具调用事件
 */
export type ToolEvent = {
  name: string;
  function: string;
  args: Record<string, unknown>;
  content?: unknown;
  /**
   * 仅 UI 展示用的数据，来自后端 ToolResult.display。
   * 大体积产物（如浏览器截图）已在后端落对象存储，这里只会出现文件引用
   * （如 screenshot: { file_id, filename, mime_type, size }），不会出现 base64。
   */
  display?: Record<string, unknown>;
  status?: ToolEventStatus;
  [key: string]: unknown;
};

/**
 * 工具执行结果
 */
export type ToolResult = {
  success: boolean;
  message: string;
  data?: unknown;
  /** 仅 UI 展示用的数据（大体积产物已落存储，这里只有文件引用），不进入 LLM 上下文 */
  display?: Record<string, unknown>;
};

/**
 * 工具调用中事件（SSE tool_calling）
 */
export type ToolCallingEvent = {
  name?: string;
  tool_call_id: string;
  function_name: string;
  arguments: Record<string, unknown>;
  [key: string]: unknown;
};

/**
 * 工具调用完成事件（SSE tool_called）
 */
export type ToolCalledEvent = {
  name?: string;
  tool_call_id: string;
  function_name: string;
  arguments: Record<string, unknown>;
  result?: ToolResult;
  [key: string]: unknown;
};

/**
 * SSE 事件类型
 */
export type SSEEventType =
  | "message"
  | "message_delta"
  | "message_done"
  | "stream_error"
  | "title"
  | "plan"
  | "step"
  | "tool_calling"
  | "tool_called"
  | "wait"
  | "done"
  | "error";

/**
 * SSE 事件数据
 */
export type SSEEventData =
  | ({ type: "message"; data: ChatMessage } & SSEEventMeta)
  | ({ type: "message_delta"; data: MessageDeltaEvent } & SSEEventMeta)
  | ({ type: "message_done"; data: MessageDoneEvent } & SSEEventMeta)
  | ({ type: "stream_error"; data: { message: string } } & SSEEventMeta)
  | ({ type: "title"; data: { title: string } } & SSEEventMeta)
  | ({ type: "plan"; data: PlanEvent } & SSEEventMeta)
  | ({ type: "step"; data: StepEvent } & SSEEventMeta)
  | ({ type: "tool_calling"; data: ToolCallingEvent } & SSEEventMeta)
  | ({ type: "tool_called"; data: ToolCalledEvent } & SSEEventMeta)
  | ({ type: "shell_output"; data: ShellOutputEvent } & SSEEventMeta)
  | ({ type: "wait"; data: Record<string, unknown> } & SSEEventMeta)
  | ({ type: "done"; data: Record<string, unknown> } & SSEEventMeta)
  | ({ type: "error"; data: { error?: string; message?: string } } & SSEEventMeta);

export type SSEEventMeta = {
  /** Redis Stream ID，唯一用于 SSE 断线续读。 */
  streamId?: string;
};

/** 长命令运行期间的控制台输出增量快照（console 为全量记录，直接替换渲染） */
export type ShellOutputEvent = {
  session_id: string;
  console: Array<{ ps1: string; command: string; output: string }>;
};

export type MessageDeltaEvent = {
  message_id: string;
  delta: string;
  sequence: number;
};

export type MessageDoneEvent = {
  message_id: string;
  content: string;
  finish_reason?: string;
};

/**
 * SSE 事件处理器
 */
export type SSEEventHandler = (event: SSEEventData) => void;

/**
 * 会话文件信息
 */
export type SessionFile = FileInfo;

/**
 * 查看文件内容请求参数
 */
export type ViewFileParams = {
  filepath: string;
  [key: string]: unknown;
};

/**
 * 查看 Shell 输出请求参数
 */
export type ViewShellParams = {
  shell_session_id: string;
  [key: string]: unknown;
};
