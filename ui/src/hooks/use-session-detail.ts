'use client'

import { useCallback, useEffect, useRef, useState } from 'react'
import { toast } from 'sonner'
import { sessionApi } from '@/lib/api/session'
import { normalizeEvent, normalizeEvents } from '@/lib/session-events'
import type { SessionDetail, SSEEventData, SessionFile } from '@/lib/api/types'

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
}

/**
 * 任务详情：拉取会话详情与文件列表，管理事件列表；
 * 未完成任务会通过 chat 空 body 流式拉取事件，发送消息时通过 chat 带 body 流式追加事件。
 */
export function useSessionDetail(
  sessionId: string | null,
  initialSkipEmptyStream?: boolean
): UseSessionDetailResult {
  const [session, setSession] = useState<SessionDetail | null>(null)
  const [files, setFiles] = useState<SessionFile[]>([])
  const [events, setEvents] = useState<SSEEventData[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<Error | null>(null)
  // 仅由"发送消息"链路产生的错误（HTTP/SSE），用于驱动"切换 Auto 重试"横幅；
  // 与 refresh 等其他来源的 error 区分开。
  const [lastSendError, setLastSendError] = useState<Error | null>(null)
  // 空流重连定时器：卸载/切会话时必须清除，否则产生孤儿 SSE 连接
  const emptyStreamTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null)
  const [streaming, setStreaming] = useState(false)
  const [skipEmptyStream, setSkipEmptyStream] = useState(initialSkipEmptyStream || false)
  const emptyStreamCleanupRef = useRef<(() => void) | null>(null)
  const messageStreamCleanupRef = useRef<(() => void) | null>(null)
  const isSendMessageRef = useRef(false)
  const lastEventIdRef = useRef<string | null>(null)

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
      const stepData = evToAppend.data as { status?: string }
      if (stepData.status === 'running') {
        setSession((prev) => prev ? { ...prev, status: 'running' } : null)
      }
      if (stepData.status === 'waiting') {
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

  const startEmptyStream = useCallback(() => {
    if (!sessionId) return
    if (emptyStreamCleanupRef.current) {
      emptyStreamCleanupRef.current()
      emptyStreamCleanupRef.current = null
    }
    emptyStreamCleanupRef.current = sessionApi.chat(
      sessionId,
      { event_id: lastEventIdRef.current || undefined },
      (ev) => appendEvent(ev),
      (err) => {
        if (err.name === 'AbortError') {
          return
        }
        // 流正常结束（服务端关闭连接），延迟重连
        if (err.message === 'SSE_STREAM_END') {
          emptyStreamCleanupRef.current = null
          emptyStreamTimerRef.current = setTimeout(() => {
            if (!emptyStreamCleanupRef.current && !isSendMessageRef.current) {
              startEmptyStream()
            }
          }, 500)
          return
        }
        emptyStreamCleanupRef.current = null
        setError(err)
        emptyStreamTimerRef.current = setTimeout(() => {
          if (!emptyStreamCleanupRef.current && !isSendMessageRef.current) {
            startEmptyStream()
          }
        }, 1000)
      }
    )
  }, [sessionId, appendEvent])

  const stopEmptyStream = useCallback(() => {
    if (emptyStreamTimerRef.current) {
      clearTimeout(emptyStreamTimerRef.current)
      emptyStreamTimerRef.current = null
    }
    if (emptyStreamCleanupRef.current) {
      emptyStreamCleanupRef.current()
      emptyStreamCleanupRef.current = null
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
      const [detail, fileListRaw] = await Promise.all([
        sessionApi.getSessionDetail(sessionId),
        sessionApi.getSessionFiles(sessionId),
      ])
      setSession(detail)
      setFiles(normalizeFileList(fileListRaw))
      const rawEvents = (detail as { events?: unknown }).events
      if (rawEvents && Array.isArray(rawEvents) && rawEvents.length > 0) {
        const normalized = normalizeEvents(rawEvents)
        setEvents(normalized)
        // DB 历史事件可能只有业务 UUID；只有 Redis ms-seq 才能作为 XREAD 游标。
        const lastEvId = [...normalized]
          .reverse()
          .find((event) => event.streamId && /^\d+-\d+$/.test(event.streamId))?.streamId
        if (lastEvId) lastEventIdRef.current = lastEvId
      }
    } catch (e) {
      setError(e instanceof Error ? e : new Error('加载失败'))
    } finally {
      setLoading(false)
    }
  }, [sessionId, normalizeFileList])

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
    if (messageStreamCleanupRef.current) {
      messageStreamCleanupRef.current()
      messageStreamCleanupRef.current = null
    }
    setEvents([])
    setLastSendError(null)
    lastEventIdRef.current = ''
    if (!sessionId) {
      setLoading(false)
      setSession(null)
      setFiles([])
      setEvents([])
      setError(null)
      stopEmptyStream()
      return
    }
    setLoading(true)
    refresh().then(() => {
      // 由下面的 effect 根据 session 状态决定是否开空流
    })
    return () => {
      stopEmptyStream()
    }
  }, [sessionId]) // eslint-disable-line react-hooks/exhaustive-deps

  useEffect(() => {
    if (!sessionId || !session) return
    const status = session.status
    const completed = status === 'completed'
    // 如果标记了跳过空流（比如有初始消息待发送），则不启动空流
    if (!completed && !isSendMessageRef.current && !skipEmptyStream) {
      startEmptyStream()
    }
    return () => {
      stopEmptyStream()
    }
  }, [sessionId, session?.status, skipEmptyStream, startEmptyStream, stopEmptyStream])

  // 组件卸载时清理消息流与重连定时器
  useEffect(() => {
    return () => {
      if (emptyStreamTimerRef.current) {
        clearTimeout(emptyStreamTimerRef.current)
        emptyStreamTimerRef.current = null
      }
      if (messageStreamCleanupRef.current) {
        messageStreamCleanupRef.current()
        messageStreamCleanupRef.current = null
      }
    }
  }, [])

  const sendMessage = useCallback(
    async (message: string, attachmentIds: string[], modelId?: string) => {
      if (!sessionId) return
      stopEmptyStream()
      // 清理已有的消息流连接（如 waiting 状态时用户再次发送）
      if (messageStreamCleanupRef.current) {
        messageStreamCleanupRef.current()
        messageStreamCleanupRef.current = null
      }
      // 发送消息时，清除跳过空流的标记
      setSkipEmptyStream(false)
      isSendMessageRef.current = true
      setStreaming(true)
      setLastSendError(null)

      // 立即更新状态为 running，不等待 SSE 事件
      setSession((prev) => prev ? { ...prev, status: 'running' } : null)

      const onEvent = (ev: SSEEventData) => {
        appendEvent(ev)
        // 错误事件：弹 toast 提示给用户，并把状态回滚到 completed
        if (ev.type === 'error') {
          const errorData = ev.data as { message?: string; error?: string }
          const errMsg = errorData.message || errorData.error || '服务异常，请稍后重试'
          toast.error(`AI 调用失败：${errMsg}`)
          setError(new Error(errMsg))
          setLastSendError(new Error(errMsg))
          setSession((prev) =>
            prev ? { ...prev, status: 'completed' } : null
          )
        }
        if (ev.type === 'done') {
          setStreaming(false)
          isSendMessageRef.current = false
          // 清理消息流的 cleanup
          if (messageStreamCleanupRef.current) {
            messageStreamCleanupRef.current()
            messageStreamCleanupRef.current = null
          }
          setSession((prev) => prev ? { ...prev, status: 'completed' } : null)
          startEmptyStream()
        }
      }
      const messageStreamCleanup = sessionApi.chat(
        sessionId,
        { message, attachments: attachmentIds, model_id: modelId },
        onEvent,
        (err) => {
          if (err.name === 'AbortError') {
            setStreaming(false)
            isSendMessageRef.current = false
            return
          }
          // 流正常结束（服务端关闭连接），重置状态并启动空流监听后续事件
          if (err.message === 'SSE_STREAM_END') {
            setStreaming(false)
            isSendMessageRef.current = false
            if (messageStreamCleanupRef.current) {
              messageStreamCleanupRef.current()
              messageStreamCleanupRef.current = null
            }
            startEmptyStream()
            return
          }
          // 实际错误
          const sendErr = err instanceof Error ? err : new Error('流式响应异常')
          setError(sendErr)
          setLastSendError(sendErr)
          setStreaming(false)
          isSendMessageRef.current = false
          // 网络断开不等于任务完成：保留当前状态，由空流重新连接并同步后端事件。
          if (messageStreamCleanupRef.current) {
            messageStreamCleanupRef.current()
            messageStreamCleanupRef.current = null
          }
          startEmptyStream()
        }
      )
      // 将消息流的 cleanup 存到独立的 ref，不与 emptyStream 混淆
      messageStreamCleanupRef.current = messageStreamCleanup
    },
    [sessionId, appendEvent, startEmptyStream, stopEmptyStream]
  )

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
  }
}
