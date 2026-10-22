import type {
  Run,
  RunHistoryPage,
  RunMessage,
  RunTraceEvent,
  SessionFile,
  SSEEventData,
} from './api/types'

type SnapshotStep = {
  id: string
  description: string
  status: 'pending' | 'running' | 'completed' | 'failed'
  result_summary?: string
  error_summary?: string
  artifact_refs?: string[]
}

type ExecutionSnapshot = {
  plan_id?: string
  plan_title?: string
  plan_goal?: string
  plan_language?: string
  plan_message?: string
  steps?: SnapshotStep[]
}

/**
 * 将历史 API 的 Run、消息和持久化追踪事件还原成统一 SSE 事件。
 * 每个 Run 的工具过程必须来自 trace；旧的无 trace Run 只使用执行快照恢复计划和步骤。
 */
export function historyToEvents(history: RunHistoryPage, files: SessionFile[]): SSEEventData[] {
  const fileMap = new Map(files.map((file) => [file.id, file]))
  const events: SSEEventData[] = []

  for (const item of [...history.items].reverse()) {
    const messages = [...item.messages].sort(compareMessages)
    const traceEvents = [...(item.events ?? [])].sort((left, right) => left.sequence - right.sequence)
    if (traceEvents.length > 0) {
      appendRunTimeline(events, item.run, messages, traceEvents, fileMap)
    } else {
      appendMessages(events, messages.filter((message) => message.role === 'user'), fileMap)
      appendSnapshotEvents(events, item.run)
      appendMessages(events, messages.filter((message) => message.role !== 'user'), fileMap)
    }
    appendPersistedFailure(events, item.run, traceEvents)
  }

  return events
}

/**
 * 追踪写入异常时，仍用 Run 终态保存的错误文本恢复用户可见错误。
 * 有 error trace 时以 trace 为准，避免刷新后重复展示。
 */
function appendPersistedFailure(events: SSEEventData[], run: Run, traceEvents: RunTraceEvent[]): void {
  if (run.status !== 'failed' || !run.error_message) return
  if (traceEvents.some((event) => event.event_type === 'error')) return
  events.push({ type: 'error', data: { message: run.error_message } } as SSEEventData)
}

type HistoryTimelineEntry = {
  timestamp: number
  order: number
  message?: RunMessage
  traceEvent?: RunTraceEvent
  append: () => void
}

function appendRunTimeline(
  events: SSEEventData[],
  run: Run,
  messages: RunMessage[],
  traceEvents: RunTraceEvent[],
  fileMap: Map<string, SessionFile>,
): void {
  const entries: HistoryTimelineEntry[] = []
  messages.forEach((message, index) => {
    entries.push({
      timestamp: parseTimestamp(message.created_at),
      order: index * 2,
      message,
      append: () => appendMessages(events, [message], fileMap),
    })
  })
  traceEvents.forEach((traceEvent, index) => {
    entries.push({
      timestamp: parseTimestamp(traceEvent.created_at),
      order: index * 2 + 1,
      traceEvent,
      append: () => appendTraceEvent(events, traceEvent),
    })
  })
  entries.sort((left, right) => left.timestamp - right.timestamp || left.order - right.order)
  reorderWaitingEntries(entries, run, messages, traceEvents)
  for (const entry of entries) entry.append()
}

function reorderWaitingEntries(
  entries: HistoryTimelineEntry[],
  run: Run,
  messages: RunMessage[],
  traceEvents: RunTraceEvent[],
): void {
  for (const traceEvent of traceEvents) {
    if (traceEvent.event_type !== 'wait') continue

    const targetMessage = findWaitingQuestion(run, traceEvent, messages)
    if (!targetMessage) continue

    const waitIndex = entries.findIndex((entry) => entry.traceEvent === traceEvent)
    const questionIndex = entries.findIndex((entry) => entry.message === targetMessage)
    if (waitIndex < 0 || questionIndex < 0 || waitIndex > questionIndex) continue

    const [waitEntry] = entries.splice(waitIndex, 1)
    const adjustedQuestionIndex = entries.findIndex((entry) => entry.message === targetMessage)
    entries.splice(adjustedQuestionIndex + 1, 0, waitEntry)
  }
}

function findWaitingQuestion(run: Run, traceEvent: RunTraceEvent, messages: RunMessage[]): RunMessage | undefined {
  const assistantMessages = messages.filter((message) => message.role === 'assistant')
  if (run.waiting_message_id) {
    const question = messages.find((message) => message.id === run.waiting_message_id)
    if (question) return question
  }

  const data = parseTraceData(traceEvent.data)
  const waitingMessage = typeof data?.message === 'string' ? data.message : undefined
  const waitTimestamp = parseTimestamp(traceEvent.created_at)
  if (waitingMessage) {
    const question = messages.find((message) => (
      message.content === waitingMessage && parseTimestamp(message.created_at) >= waitTimestamp
    ))
    if (question) return question
  }

  // 事件发出后，RunExecutor 才会持久化对应的问题消息；终态 Run 已清空 waiting_message_id。
  const nextAssistant = assistantMessages
    .filter((message) => parseTimestamp(message.created_at) >= waitTimestamp)
    .sort((left, right) => parseTimestamp(left.created_at) - parseTimestamp(right.created_at))[0]
  if (!nextAssistant) return undefined

  if (run.status === 'waiting_input') return nextAssistant
  return messages.some((message) => (
    message.role === 'user' && message.reply_to_message_id === nextAssistant.id
  )) ? nextAssistant : undefined
}

function parseTimestamp(value: string): number {
  const timestamp = Date.parse(value)
  return Number.isNaN(timestamp) ? Number.MAX_SAFE_INTEGER : timestamp
}

function compareMessages(left: RunMessage, right: RunMessage): number {
  return left.created_at.localeCompare(right.created_at) || left.id.localeCompare(right.id)
}

function appendMessages(
  events: SSEEventData[],
  messages: RunMessage[],
  fileMap: Map<string, SessionFile>,
): void {
  for (const message of messages) {
    events.push({
      type: 'message',
      data: {
        role: message.role,
        message: message.content,
        message_id: message.id,
        attachments: (message.attachments || []).map((id) => {
          const file = fileMap.get(id)
          return { file_id: id, filename: file?.filename || id, size: file?.size || 0 }
        }),
      },
    } as SSEEventData)
  }
}

function appendTraceEvent(events: SSEEventData[], traceEvent: RunTraceEvent): void {
  const data = parseTraceData(traceEvent.data)
  if (data === undefined) return
  events.push({ type: traceEvent.event_type, data: normalizeTraceData(traceEvent.event_type, data) } as SSEEventData)
}

function normalizeTraceData(eventType: RunTraceEvent['event_type'], data: Record<string, unknown>): Record<string, unknown> {
  if (eventType !== 'step') return data
  const rawStep = data.step
  const lifecycleStatus = typeof data.status === 'string' ? data.status : undefined
  const normalizedStatus = normalizeStepStatus(lifecycleStatus)
  if (!rawStep || typeof rawStep !== 'object' || Array.isArray(rawStep)) {
    return normalizedStatus ? { ...data, status: normalizedStatus } : data
  }
  const step = rawStep as Record<string, unknown>
  return {
    ...data,
    status: normalizedStatus || step.status,
    step: {
      ...step,
      status: normalizedStatus || step.status,
    },
  }
}

function normalizeStepStatus(status: string | undefined): 'pending' | 'running' | 'completed' | 'failed' | undefined {
  switch (status) {
    case 'started':
    case 'running':
      return 'running'
    case 'completed':
      return 'completed'
    case 'failed':
      return 'failed'
    case 'pending':
      return 'pending'
    default:
      return undefined
  }
}

function parseTraceData(data: unknown): Record<string, unknown> | undefined {
  if (typeof data === 'string') {
    try {
      const parsed: unknown = JSON.parse(data)
      return parsed && typeof parsed === 'object' && !Array.isArray(parsed)
        ? parsed as Record<string, unknown>
        : undefined
    } catch {
      return undefined
    }
  }
  return data && typeof data === 'object' && !Array.isArray(data)
    ? data as Record<string, unknown>
    : undefined
}

function appendSnapshotEvents(events: SSEEventData[], run: Run): void {
  const snapshot = run.execution_snapshot as ExecutionSnapshot | undefined
  if (!snapshot?.plan_id || !Array.isArray(snapshot.steps) || snapshot.steps.length === 0) return

  const steps = snapshot.steps.map((step) => ({
    id: step.id,
    description: step.description,
    status: step.status,
    result_summary: step.result_summary,
    error_summary: step.error_summary,
    artifact_refs: step.artifact_refs,
  }))
  events.push({
    type: 'plan',
    data: {
      plan: {
        id: snapshot.plan_id,
        title: snapshot.plan_title || '',
        goal: snapshot.plan_goal || '',
        language: snapshot.plan_language || '',
        message: snapshot.plan_message || '',
        steps,
      },
      status: 'created',
    },
  } as SSEEventData)
  for (const step of steps) {
    events.push({
      type: 'step',
      data: {
        step: {
          ...step,
          id: step.id,
          description: step.description,
          result: step.result_summary,
          error: step.error_summary,
        },
        status: step.status,
      },
    } as SSEEventData)
  }
}
