'use client'

import { useCallback, useEffect, useLayoutEffect, useMemo, useRef, useState } from 'react'
import { useRouter } from 'next/navigation'
import { SessionHeader } from '@/components/session-header'
import { ChatInput, type ChatInputRef } from '@/components/chat-input'
import { AUTO_MODEL_ID } from '@/providers/models-provider'
import { PlanPanel } from '@/components/plan-panel'
import { ChatMessage } from '@/components/chat-message'
import { FilePreviewPanel } from '@/components/file-preview-panel'
import { ToolPreviewPanel } from '@/components/tool-preview-panel'
import { VNCOverlay } from '@/components/vnc-overlay'
import { useSessionDetail } from '@/hooks/use-session-detail'
import { getToolKind } from '@/components/tool-use/utils'
import {
  eventsToTimeline,
  getLatestPlanFromEvents,
} from '@/lib/session-events'
import type { ToolEvent, FileInfo } from '@/lib/api/types'
import type { AttachmentFile, TimelineItem } from '@/lib/session-events'
import { sessionApi } from '@/lib/api/session'
import { toast } from 'sonner'
import { ArrowDown, Loader2 } from 'lucide-react'
import { Button } from '@/components/ui/button'

export interface SessionDetailViewProps {
  sessionId: string
  initialMessage?: string
  initialAttachments?: string[]
  hasInitialMessage?: boolean
}

/**
 * 从 timeline 中找到最后一个非 message 类型的工具事件
 */
function findLatestTool(timeline: TimelineItem[]): ToolEvent | null {
  for (let i = timeline.length - 1; i >= 0; i--) {
    const item = timeline[i]
    if (item.kind === 'tool' && getToolKind(item.data) !== 'message') {
      return item.data
    }
    if (item.kind === 'step' && item.tools.length > 0) {
      for (let j = item.tools.length - 1; j >= 0; j--) {
        if (getToolKind(item.tools[j]) !== 'message') {
          return item.tools[j]
        }
      }
    }
  }
  return null
}

export function SessionDetailView({ sessionId, initialMessage, initialAttachments, hasInitialMessage }: SessionDetailViewProps) {
  const router = useRouter()
  const chatInputRef = useRef<ChatInputRef>(null)
  const {
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
  } = useSessionDetail(sessionId, hasInitialMessage)

  // 模型选择状态（提升到此处持有：chat-input 只做受控展示，
  // 支持失败后从外部把选择切换为 Auto 并重发）
  const [selectedModelId, setSelectedModelId] = useState<string>(AUTO_MODEL_ID)
  // 上一次发送记录：失败时用于"切换 Auto 重试"
  const lastSendRef = useRef<{message: string; files: FileInfo[]; modelId?: string} | null>(null)
  const [autoRetry, setAutoRetry] = useState<{message: string; files: FileInfo[]} | null>(null)

  // 按会话持久化模型选择：刷新页面后保持该会话上次的选择（无记录则 Auto）。
  // localStorage 仅在客户端 effect 中访问，避免 SSR 水合不一致。
  const modelSelectionKey = sessionId ? `manus:model-selection:${sessionId}` : null

  // 切会话时恢复该会话上次的选择（而不是无条件重置为 Auto），
  // 避免上一个会话的选模型"串"到新会话
  useEffect(() => {
    setAutoRetry(null)
    if (!sessionId) {
      setSelectedModelId(AUTO_MODEL_ID)
      return
    }
    let saved = AUTO_MODEL_ID
    try {
      saved = localStorage.getItem(modelSelectionKey ?? '') || AUTO_MODEL_ID
    } catch {
      // localStorage 不可用（隐私模式等）：退回 Auto
    }
    setSelectedModelId(saved)
  }, [sessionId, modelSelectionKey])

  // 选择变更时写回持久化存储
  const handleModelSelect = useCallback((id: string) => {
    setSelectedModelId(id)
    if (modelSelectionKey) {
      try {
        localStorage.setItem(modelSelectionKey, id)
      } catch {
        // 存储不可用时静默：仅影响刷新后的记忆
      }
    }
  }, [modelSelectionKey])

  // 发送链路出错（404 预检/SSE error）且当时选定了具体模型 → 给出切 Auto 重试入口
  useEffect(() => {
    if (lastSendError && lastSendRef.current?.modelId) {
      setAutoRetry({message: lastSendRef.current.message, files: lastSendRef.current.files})
    }
  }, [lastSendError])

  const timeline = useMemo(() => eventsToTimeline(events), [events])
  const planSteps = useMemo(() => getLatestPlanFromEvents(events), [events])

  const [fileListOpen, setFileListOpen] = useState(false)
  const [previewFile, setPreviewFile] = useState<AttachmentFile | null>(null)
  const [previewTool, setPreviewTool] = useState<ToolEvent | null>(null)
  const [vncOpen, setVncOpen] = useState(false)
  const [showJumpToBottom, setShowJumpToBottom] = useState(false)
  const initialMessageSentRef = useRef(false)
  const scrollContainerRef = useRef<HTMLDivElement>(null)
  const prevToolCountRef = useRef(0)

  const hasPreview = previewFile !== null || previewTool !== null

  /**
   * 将 previewTool 解析为 timeline 中最新版本的工具对象。
   * 自动跟踪设置 previewTool 时工具事件可能尚无 content（如截图），
   * 后续 SSE 更新后 timeline 中对象已刷新但 state 仍为旧引用。
   * 通过 tool_call_id 匹配获取最新版本。
   */
  const resolvedPreviewTool = useMemo(() => {
    if (!previewTool) return null
    const id = (previewTool as { tool_call_id?: string }).tool_call_id
    if (!id) return previewTool

    for (let i = timeline.length - 1; i >= 0; i--) {
      const item = timeline[i]
      if (item.kind === 'tool' && (item.data as { tool_call_id?: string }).tool_call_id === id) {
        return item.data
      }
      if (item.kind === 'step') {
        for (const t of item.tools) {
          if ((t as { tool_call_id?: string }).tool_call_id === id) return t
        }
      }
    }
    return previewTool
  }, [previewTool, timeline])

  // 任务运行中自动追踪最新工具预览（VNC 打开时暂停）
  useEffect(() => {
    if (session?.status !== 'running' || vncOpen) return

    const latestTool = findLatestTool(timeline)
    const toolCount = timeline.reduce((n, item) => {
      if (item.kind === 'tool') return n + 1
      if (item.kind === 'step') return n + item.tools.length
      return n
    }, 0)

    if (toolCount > prevToolCountRef.current && latestTool) {
      setPreviewTool(latestTool)
      setPreviewFile(null)
      scrollContainerRef.current?.scrollTo({ top: scrollContainerRef.current.scrollHeight, behavior: 'smooth' })
    }
    prevToolCountRef.current = toolCount
  }, [timeline, session?.status, vncOpen])

  // 进入会话（sessionId 变化）时重置"已就位"标记
  const scrolledForSessionRef = useRef<string | null>(null)
  useLayoutEffect(() => {
    scrolledForSessionRef.current = null
  }, [sessionId])

  // 滚到底：仅当当前会话还没滚过、且内容已加载
  useLayoutEffect(() => {
    const el = scrollContainerRef.current
    if (!el) return
    if (scrolledForSessionRef.current === sessionId) return
    if (timeline.length > 0 || streaming) {
      el.scrollTop = el.scrollHeight
      scrolledForSessionRef.current = sessionId
    }
  })

  useEffect(() => {
    const el = scrollContainerRef.current
    if (!el) return
    const onScroll = () => {
      const distanceToBottom = el.scrollHeight - el.scrollTop - el.clientHeight
      setShowJumpToBottom(distanceToBottom > 200)
    }
    el.addEventListener('scroll', onScroll, {passive: true})
    return () => el.removeEventListener('scroll', onScroll)
  }, [])

  useEffect(() => {
    if (
      initialMessage &&
      !initialMessageSentRef.current &&
      session &&
      !loading &&
      !streaming
    ) {
      initialMessageSentRef.current = true
      sendMessage(initialMessage, initialAttachments || [])
        .then(() => {
          setTimeout(() => {
            router.replace(`/sessions/${sessionId}`)
          }, 100)
        })
        .catch((e) => {
          toast.error(e instanceof Error ? e.message : '发送消息失败')
        })
    }
  }, [initialMessage, initialAttachments, session, loading, streaming, sendMessage, sessionId, router])

  const handleSend = useCallback(
    async (message: string, uploadedFiles: FileInfo[], modelId?: string) => {
      lastSendRef.current = {message, files: uploadedFiles, modelId}
      try {
        const attachmentIds = uploadedFiles.map((f) => f.id)
        await sendMessage(message, attachmentIds, modelId)
        setAutoRetry(null)
      } catch (e) {
        toast.error(e instanceof Error ? e.message : '发送失败，请重试')
        // 选定了具体模型且发送失败（如模型不存在/上游不可用）：
        // 给出"切换 Auto 重试"的快速恢复路径（对齐 Cursor 的错误恢复交互）
        if (modelId) {
          setAutoRetry({message, files: uploadedFiles})
        }
        throw e
      }
    },
    [sendMessage]
  )

  const handleAutoRetry = useCallback(() => {
    if (!autoRetry) return
    handleModelSelect(AUTO_MODEL_ID)
    const {message, files} = autoRetry
    setAutoRetry(null)
    chatInputRef.current?.clear()
    handleSend(message, files, undefined).catch(() => {
      // 重试失败已在 handleSend 内 toast，不再叠加
    })
  }, [autoRetry, handleSend])

  const handleViewAllFiles = useCallback(() => {
    refreshFiles()
    setFileListOpen(true)
  }, [refreshFiles])

  const handleFileClick = useCallback((file: AttachmentFile) => {
    setPreviewFile(file)
    setPreviewTool(null)
  }, [])

  const handleToolClick = useCallback((tool: ToolEvent) => {
    const kind = getToolKind(tool)
    if (kind === 'message') return
    setPreviewTool(tool)
    setPreviewFile(null)
  }, [])

  const handleClosePreview = useCallback(() => {
    setPreviewFile(null)
    setPreviewTool(null)
  }, [])

  const handleJumpToLatest = useCallback(() => {
    const latest = findLatestTool(timeline)
    if (latest) {
      setPreviewTool(latest)
      setPreviewFile(null)
    }
    scrollContainerRef.current?.scrollTo({ top: scrollContainerRef.current.scrollHeight, behavior: 'smooth' })
  }, [timeline])

  const handleOpenVNC = useCallback(() => {
    setVncOpen(true)
  }, [])

  const handleCloseVNC = useCallback(() => {
    setVncOpen(false)
    // 关闭 VNC 后跳转到最新工具
    const latest = findLatestTool(timeline)
    if (latest && session?.status === 'running') {
      setPreviewTool(latest)
      setPreviewFile(null)
      setTimeout(() => {
        scrollContainerRef.current?.scrollTo({ top: scrollContainerRef.current.scrollHeight, behavior: 'smooth' })
      }, 100)
    }
  }, [timeline, session?.status])

  const handleStop = useCallback(async () => {
    if (!session) return
    try {
      await sessionApi.stopSession(sessionId)
      toast.success('任务已停止')
      refresh()
    } catch (error) {
      toast.error(error instanceof Error ? error.message : '停止任务失败')
    }
  }, [session, sessionId, refresh])

  if (loading && !session) {
    return (
      <div className="relative flex flex-col h-full flex-1 min-w-0 px-4 items-center justify-center">
        {hasInitialMessage ? (
          <div className="flex items-center gap-2 text-sm text-gray-500">
            <Loader2 className="size-4 animate-spin" />
            <span>正在思考中...</span>
          </div>
        ) : (
          <p className="text-sm text-gray-500">加载中...</p>
        )}
      </div>
    )
  }

  if (error && !session) {
    return (
      <div className="relative flex flex-col h-full flex-1 min-w-0 px-4 items-center justify-center gap-2">
        <p className="text-sm text-red-600">{error.message}</p>
        <button
          type="button"
          onClick={() => refresh()}
          className="text-sm text-primary underline"
        >
          重试
        </button>
      </div>
    )
  }

  if (!session) {
    return (
      <div className="relative flex flex-col h-full flex-1 min-w-0 px-4 items-center justify-center">
        <p className="text-sm text-gray-500">未找到该任务</p>
      </div>
    )
  }

  return (
    <>
      <div className="relative flex flex-row h-full w-full overflow-hidden">
        {/* 主内容区 - flex column：header / scroll / input */}
        <div className="relative flex flex-col flex-1 min-w-0 min-h-0 h-full overflow-hidden">
          <div className={`flex flex-col h-full w-full mx-auto min-w-0 px-4 ${hasPreview ? '' : 'max-w-[768px]'}`}>
            <div className="flex-shrink-0 z-10 bg-[#f8f8f7]">
              <SessionHeader
                title={session.title}
                files={files}
                fileListOpen={fileListOpen}
                onFileListOpenChange={setFileListOpen}
                onFetchFiles={refreshFiles}
                onFileClick={handleFileClick}
              />
            </div>

            <div
              ref={scrollContainerRef}
              className="relative flex-1 overflow-y-auto pb-2"
              style={{overflowAnchor: 'none', minHeight: 0}}
            >
              {showJumpToBottom && (
                <Button
                  size="icon-sm"
                  variant="secondary"
                  className="sticky bottom-4 z-20 ml-auto mr-2 rounded-full shadow-md cursor-pointer block"
                  onClick={() => {
                    const el = scrollContainerRef.current
                    if (!el) return
                    el.scrollTo({top: el.scrollHeight, behavior: 'smooth'})
                  }}
                  title="跳到最新"
                >
                  <ArrowDown/>
                </Button>
              )}
              <div className="flex flex-col w-full gap-3 pt-3">
                {timeline.length === 0 && !streaming && !hasInitialMessage && (
                  <div className="flex items-center justify-center py-8 text-sm text-gray-500">
                    暂无对话记录，在下方输入任务或提问
                  </div>
                )}
                {timeline.map((item) => (
                  <ChatMessage
                    key={item.id}
                    item={item}
                    onViewAllFiles={handleViewAllFiles}
                    onFileClick={handleFileClick}
                    onToolClick={handleToolClick}
                  />
                ))}

                {(session?.status === 'running' || (hasInitialMessage && !initialMessageSentRef.current)) && (
                  <div className="flex items-center gap-2 text-sm text-gray-500 py-3">
                    <Loader2 className="size-4 animate-spin" />
                    <span>正在思考中...</span>
                  </div>
                )}
              </div>
            </div>

            <div className="flex-shrink-0 bg-[#f8f8f7] py-3">
              <PlanPanel className="mb-2" steps={planSteps} />
              {/* 发送失败（选定了具体模型）时的快速恢复：切 Auto 重试 */}
              {autoRetry && (
                <div className="flex items-center justify-between gap-2 mx-3 mb-1 px-3 py-2 text-xs rounded-lg bg-red-50 border border-red-200">
                  <span className="text-red-600 truncate">所选模型调用失败，可切换为 Auto 后自动重试</span>
                  <div className="flex gap-1 flex-shrink-0">
                    <Button size="xs" variant="ghost" className="cursor-pointer" onClick={() => setAutoRetry(null)}>
                      忽略
                    </Button>
                    <Button size="xs" className="cursor-pointer" onClick={handleAutoRetry}>
                      切换 Auto 重试
                    </Button>
                  </div>
                </div>
              )}
              <ChatInput
                ref={chatInputRef}
                onSend={handleSend}
                modelId={selectedModelId}
                onModelIdChange={handleModelSelect}
                sessionId={sessionId}
                isRunning={session?.status === 'running'}
                onStop={handleStop}
              />
            </div>
          </div>
        </div>

        {/* 文件预览面板 */}
        {previewFile && (
          <div className="flex-shrink-0 w-[600px] h-full animate-in slide-in-from-right duration-300">
            <FilePreviewPanel file={previewFile} onClose={handleClosePreview} />
          </div>
        )}

        {/* 工具预览面板 */}
        {resolvedPreviewTool && (
          <div className="flex-shrink-0 w-[600px] h-full py-2 pr-2 animate-in slide-in-from-right duration-300">
            <ToolPreviewPanel
              tool={resolvedPreviewTool}
              onClose={handleClosePreview}
              onJumpToLatest={handleJumpToLatest}
              onOpenVNC={getToolKind(resolvedPreviewTool) === 'browser' ? handleOpenVNC : undefined}
            />
          </div>
        )}
      </div>

      {/* noVNC 全屏远程桌面覆盖层 */}
      {vncOpen && (
        <VNCOverlay sessionId={sessionId} onClose={handleCloseVNC} />
      )}
    </>
  )
}
