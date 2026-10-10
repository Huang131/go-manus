'use client'

import { useCallback, useEffect, useRef, useState } from 'react'
import { toast } from 'sonner'
import { sessionApi } from '@/lib/api/session'
import { runApi } from '@/lib/api/run'
import { StreamEndError } from '@/lib/api/fetch'
import { normalizeEvent } from '@/lib/session-events'
import type { Run, RunHistoryPage, SessionDetail, SSEEventData, SessionFile } from '@/lib/api/types'

export type UseSessionDetailResult = {
  session: SessionDetail | null
  files: SessionFile[]
  events: SSEEventData[]
  loading: boolean
  error: Error | null
  refresh: () => Promise<void>
  refreshFiles: () => Promise<void>
  sendMessage: (message: string, attachmentIds: string[], modelId?: string) => Promise<void>
  streaming: boolean
  lastSendError: Error | null
  streamingText: { messageId: string; text: string } | null
  cancelRun: () => Promise<void>
}

/**
 * 任务详情：Session 只提供容器信息，执行生命周期和事件流全部由 Run 驱动。
 */
export function useSessionDetail(sessionId: string | null): UseSessionDetailResult {
  const [session, setSession] = useState<SessionDetail | null>(null)
  const [files, setFiles] = useState<SessionFile[]>([])
  const [events, setEvents] = useState<SSEEventData[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<Error | null>(null)
  // 仅由"发送消息"链路产生的错误（HTTP/SSE），用于驱动"切换 Auto 重试"横幅；
  // 与 refresh 等其他来源的 error 区分开。
  const [lastSendError, setLastSendError] = useState<Error | null>(null)
  // 空流重连定时器：卸载/切会话时必须清除，否则产生孤儿 SSE 连接
  const runStreamTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null)
  // 流式中的 assistant 增量文本（独立于 events，避免每 token 触发 timeline 全量重算）
  const [streamingText, setStreamingText] = useState<{ messageId: string; text: string } | null>(null)
  const [streaming, setStreaming] = useState(false)
  const runStreamCleanupRef = useRef<(() => void) | null>(null)
  const activeRunIdRef = useRef<string | null>(null)
  const waitingMessageIdRef = useRef<string | null>(null)
  const lastEventIdRef = useRef<string | null>(null)
  const streamingDeltaIdsRef = useRef<Set<string>>(new Set())

  const runToSessionStatus = useCallback((run: Run | null): SessionDetail['status'] => {
    if (!run) return 'completed'
    if (run.status === 'waiting_input') return 'waiting'
    if (run.status === 'pending' || run.status === 'running' || run.status === 'cancelling') return 'running'
    return 'completed'
  }, [])

  const applyRun = useCallback((run: Run | null) => {
    if (!run) return
    activeRunIdRef.current = run.status === 'succeeded' || run.status === 'failed' || run.status === 'cancelled' || run.status === 'interrupted'
      ? null
      : run.id
    waitingMessageIdRef.current = run.status === 'waiting_input' ? (run.waiting_message_id || null) : null
    setSession((prev) => prev ? { ...prev, status: runToSessionStatus(run) } : prev)
    setStreaming(run.status === 'pending' || run.status === 'running')
  }, [runToSessionStatus])

  const appendEvent = useCallback((ev: SSEEventData) => {
    let evToAppend = ev
    if (ev.data && typeof ev.data === 'object' && ('event' in ev.data || 'type' in ev.data) && 'data' in ev.data) {
      const normalized = normalizeEvent(ev.data as { event?: string; type?: string; data?: unknown })
      if (normalized) evToAppend = { ...normalized, streamId: ev.streamId }
    }

    if (evToAppend.streamId) lastEventIdRef.current = evToAppend.streamId

    const payload = evToAppend.data as { event_id?: string } | undefined
    const streamId = evToAppend.streamId
    const eventId = payload?.event_id

    setEvents((prev) => {
      if (
        prev.some((item) => {
          const itemPayload = item.data as { event_id?: string } | undefined
          return (
            (streamId !== undefined && item.streamId === streamId) ||
            (eventId !== undefined && itemPayload?.event_id === eventId)
          )
        })
      ) {
        return prev
      }
      return [...prev, evToAppend]
    })

    // 更新会话标题
    if (evToAppend.type === 'title' && evToAppend.data && typeof (evToAppend.data as { title?: string }).title === 'string') {
      setSession((prev) =>
        prev ? { ...prev, title: (evToAppend.data as { title: string }).title } : null
      )
    }

    // 监听事件更新会话状态
    if (evToAppend.type === 'step') {
      // 后端 FullStepEvent 同时携带外层 status 与嵌套 step.status：
      // 两种形状都读取，避免其中一种形状下状态机不翻转
      const stepData = evToAppend.data as {
        status?: string
        step?: { status?: string }
      }
      const effectiveStepStatus = stepData.status ?? stepData.step?.status
      if (effectiveStepStatus === 'running') {
        setSession((prev) => prev ? { ...prev, status: 'running' } : null)
      }
      if (effectiveStepStatus === 'waiting') {
        setSession((prev) => prev ? { ...prev, status: 'waiting' } : null)
        setStreaming(false)
      }
    }

    // message_ask_user → 等待用户输入，切换为 waiting
    // chat 流中 tool_calling 事件携带原始字段 function_name（未归一化）
    if (evToAppend.type === 'tool_calling') {
      const toolData = evToAppend.data as { function_name?: string }
      if (toolData.function_name === 'message_ask_user') {
        setSession((prev) => prev ? { ...prev, status: 'waiting' } : null)
        setStreaming(false)
      }
    }

    // wait 事件 → 等待用户输入
    if (evToAppend.type === 'wait') {
      setSession((prev) => prev ? { ...prev, status: 'waiting' } : null)
      setStreaming(false)
    }

    // done 事件时更新为 completed
    if (evToAppend.type === 'done') {
      setSession((prev) => prev ? { ...prev, status: 'completed' } : null)
    }

    // error 事件时也可以认为任务结束
    if (evToAppend.type === 'error') {
      setSession((prev) => prev ? { ...prev, status: 'completed' } : null)
    }
    if (evToAppend.type === 'stream_error') {
      const message = (evToAppend.data as { message?: string })?.message
      if (message) setError(new Error(message))
    }
  }, [])

  // 流式增量路由：message_delta 进独立 state（O(1) 更新），
  // 不进 events 数组——那会每个 token 触发 timeline 全量重算 O(n²)。
  // 返回 true 表示该事件已被路由，调用方不要再 appendEvent。
  const routeStreamingDelta = useCallback((ev: SSEEventData): boolean => {
    if (ev.type !== 'message_delta') return false
    // 实时流与断线重放可能交叠，按 Redis streamId 去重后再拼接增量。
    if (ev.streamId) {
      if (streamingDeltaIdsRef.current.has(ev.streamId)) return true
      streamingDeltaIdsRef.current.add(ev.streamId)
      lastEventIdRef.current = ev.streamId
    }
    const d = ev.data as { message_id?: string; delta?: string }
    const mid = d?.message_id
    const delta = d?.delta
    if (!mid || typeof delta !== 'string') return false
    setStreamingText((prev) =>
      prev && prev.messageId === mid
        ? { messageId: mid, text: prev.text + delta }
        : { messageId: mid, text: delta }
    )
    return true
  }, [])

  const startRunStream = useCallback((runId: string) => {
    if (!sessionId || !runId) return
    if (runStreamCleanupRef.current) {
      runStreamCleanupRef.current()
      runStreamCleanupRef.current = null
    }
    runStreamCleanupRef.current = runApi.streamEvents(
      runId,
      lastEventIdRef.current || undefined,
      (ev) => {
        if (routeStreamingDelta(ev)) return
        appendEvent(ev)
        if (ev.type === 'message_done') setStreamingText(null)
        if (ev.type === 'wait') {
          void runApi.get(runId).then(applyRun).catch(() => undefined)
        }
        if (ev.type === 'done' || ev.type === 'error') {
          setStreaming(false)
          setStreamingText(null)
          void runApi.get(runId).then(applyRun).catch(() => undefined)
        }
      },
      (err) => {
        if (err.name === 'AbortError') {
          return
        }
        if (err instanceof StreamEndError) {
          runStreamCleanupRef.current = null
          void runApi.get(runId).then((run) => {
            applyRun(run)
            if (run.status === 'pending' || run.status === 'running' || run.status === 'cancelling') {
              runStreamTimerRef.current = setTimeout(() => startRunStream(runId), 500)
            }
          }).catch(() => undefined)
          return
        }
        runStreamCleanupRef.current = null
        setError(err)
        runStreamTimerRef.current = setTimeout(() => startRunStream(runId), 1000)
      }
    )
  }, [sessionId, appendEvent, applyRun, routeStreamingDelta])

  const stopRunStream = useCallback(() => {
    if (runStreamTimerRef.current) {
      clearTimeout(runStreamTimerRef.current)
      runStreamTimerRef.current = null
    }
    if (runStreamCleanupRef.current) {
      runStreamCleanupRef.current()
      runStreamCleanupRef.current = null
    }
  }, [])

  const normalizeFileList = useCallback((raw: unknown): SessionFile[] => {
    if (Array.isArray(raw)) return raw as SessionFile[]
    if (raw && typeof raw === 'object' && 'files' in raw && Array.isArray((raw as { files: unknown }).files)) {
      return (raw as { files: SessionFile[] }).files
    }
    if (raw && typeof raw === 'object' && 'data' in raw && Array.isArray((raw as { data: unknown }).data)) {
      return (raw as { data: SessionFile[] }).data
    }
    return []
  }, [])

  const refresh = useCallback(async () => {
    if (!sessionId) return
    setError(null)
    try {
      const [detail, fileListRaw, history] = await Promise.all([
        sessionApi.getSessionDetail(sessionId),
        sessionApi.getSessionFiles(sessionId),
        runApi.listBySession(sessionId),
      ])
      setFiles(normalizeFileList(fileListRaw))
      const historyEvents = historyToEvents(history, normalizeFileList(fileListRaw))
      setEvents(historyEvents)
      const active = history.items.find((item) => item.run.status === 'pending' || item.run.status === 'running' || item.run.status === 'waiting_input' || item.run.status === 'cancelling')
      applyRun(active?.run || null)
      setSession({ ...detail, status: runToSessionStatus(active?.run || null) })
    } catch (e) {
      setError(e instanceof Error ? e : new Error('加载失败'))
    } finally {
      setLoading(false)
    }
  }, [sessionId, normalizeFileList, applyRun, runToSessionStatus])

  const refreshFiles = useCallback(async () => {
    if (!sessionId) return
    try {
      const fileListRaw = await sessionApi.getSessionFiles(sessionId)
      setFiles(normalizeFileList(fileListRaw))
    } catch (e) {
      console.error('刷新文件列表失败:', e)
    }
  }, [sessionId, normalizeFileList])

  useEffect(() => {
    // 切会话：清理上一会话的消息流与状态，避免事件串台、游标串用
    stopRunStream()
    setEvents([])
    setLastSendError(null)
    setStreamingText(null)
    streamingDeltaIdsRef.current.clear()
    lastEventIdRef.current = ''
    if (!sessionId) {
      setLoading(false)
      setSession(null)
      setFiles([])
      setEvents([])
      setError(null)
      stopRunStream()
      return
    }
    setLoading(true)
    refresh().then(() => {
      if (activeRunIdRef.current) startRunStream(activeRunIdRef.current)
    })
    return () => {
      stopRunStream()
    }
  }, [sessionId, refresh, startRunStream, stopRunStream])

  // 组件卸载时清理消息流与重连定时器
  useEffect(() => {
    return () => {
      stopRunStream()
    }
  }, [stopRunStream])

  const sendMessage = useCallback(
    async (message: string, attachmentIds: string[], modelId?: string) => {
      if (!sessionId) return
      stopRunStream()
      setStreaming(true)
      setLastSendError(null)

      setSession((prev) => prev ? { ...prev, status: 'running' } : null)
      try {
        const waitingRunId = activeRunIdRef.current
        const waitingMessageId = waitingMessageIdRef.current
        // 新 Run 的 Redis Stream 与上一轮完全独立，不能携带旧游标续读。
        if (!waitingRunId || !waitingMessageId) {
          lastEventIdRef.current = null
          streamingDeltaIdsRef.current.clear()
          setStreamingText(null)
        }
        const run = waitingRunId && waitingMessageId
          ? await runApi.submitInput(waitingRunId, message, waitingMessageId, attachmentIds, modelId)
          : await runApi.create(sessionId, message, attachmentIds, modelId)
        activeRunIdRef.current = run.id
        waitingMessageIdRef.current = run.waiting_message_id || null
        await refresh()
        startRunStream(run.id)
      } catch (e) {
        const sendErr = e instanceof Error ? e : new Error('Run 创建失败')
        setError(sendErr)
        setLastSendError(sendErr)
        setStreaming(false)
        toast.error(`消息发送失败：${sendErr.message}`)
        throw sendErr
      }
    },
    [sessionId, refresh, startRunStream, stopRunStream]
  )

  const cancelRun = useCallback(async () => {
    if (!activeRunIdRef.current) return
    const run = await runApi.cancel(activeRunIdRef.current)
    applyRun(run)
  }, [applyRun])

  return {
    session,
    files,
    events,
    loading,
    error,
    refresh,
    refreshFiles,
    sendMessage,
    streaming,
    lastSendError,
    streamingText,
    cancelRun,
  }
}

function historyToEvents(history: RunHistoryPage, files: SessionFile[]): SSEEventData[] {
  const fileMap = new Map(files.map((file) => [file.id, file]))
  const events: SSEEventData[] = []
  for (const item of [...history.items].reverse()) {
    for (const message of item.messages) {
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
  return events
}
