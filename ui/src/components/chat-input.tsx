'use client'

import {useState, useRef, useEffect, forwardRef, useImperativeHandle} from 'react'
import {cn, formatFileSize} from '@/lib/utils'
import {ScrollArea, ScrollBar} from '@/components/ui/scroll-area'
import {Item, ItemActions, ItemContent, ItemDescription, ItemMedia, ItemTitle} from '@/components/ui/item'
import {Avatar, AvatarGroupCount} from '@/components/ui/avatar'
import {ArrowUp, FileText, Paperclip, XCircle, Loader2, Pause} from 'lucide-react'
import {Button} from '@/components/ui/button'
import {fileApi} from '@/lib/api/file'
import {configApi} from '@/lib/api/config'
import type {FileInfo} from '@/lib/api/types'
import {toast} from 'sonner'

interface ChatInputProps {
  className?: string
  onInputValueChange?: (value: string) => void
  onSend?: (message: string, files: FileInfo[], modelId?: string) => Promise<void>
  disabled?: boolean
  /** 当前会话 ID，上传附件时会关联到该会话 */
  sessionId?: string | null
  /** 任务是否正在运行中 */
  isRunning?: boolean
  /** 点击暂停按钮的回调 */
  onStop?: () => void
}

export interface ChatInputRef {
  setInputText: (text: string) => void
  getInputValue: () => string
  getFiles: () => FileInfo[]
}

export const ChatInput = forwardRef<ChatInputRef, ChatInputProps>(
  ({ className, onInputValueChange, onSend, disabled = false, sessionId, isRunning = false, onStop }, ref) => {
  // 当前会话选中的模型 ID（用户可在输入框上方切换；空字符串=走 default）
  const [currentModelId, setCurrentModelId] = useState<string>('')
  const [models, setModels] = useState<Array<{id: string; name: string; model_name: string; is_default?: boolean}>>([])
  const [modelsLoading, setModelsLoading] = useState(false)
  useEffect(() => {
    let alive = true
    setModelsLoading(true)
    configApi.listLLMModels()
      .then((data) => {
        if (!alive) return
        const enabled = (data.models || []).filter((m) => m.is_enabled !== false)
        setModels(enabled)
        // 默认选中 default
        if (!currentModelId) {
          const def = enabled.find((m) => m.is_default) || enabled[0]
          if (def) setCurrentModelId(def.id)
        }
      })
      .catch(() => {/* 静默失败，模型选择降级为不可用 */})
      .finally(() => alive && setModelsLoading(false))
    return () => { alive = false }
  }, [])

  // 切会话时，重置模型选择为 default（避免上一个会话的选模型"串"到新会话）
  useEffect(() => {
    // 直接选 default（models 已加载）
    const def = models.find((m) => m.is_default) || models[0]
    setCurrentModelId(def ? def.id : '')
  }, [sessionId])
    const [files, setFiles] = useState<FileInfo[]>([])
    const [uploading, setUploading] = useState(false)
    const [sending, setSending] = useState(false)
    const [inputValue, setInputValue] = useState('')
    const fileInputRef = useRef<HTMLInputElement>(null)
    const textareaRef = useRef<HTMLTextAreaElement>(null)

    const handleInputChange = (e: React.ChangeEvent<HTMLTextAreaElement>) => {
      const value = e.target.value
      setInputValue(value)
      onInputValueChange?.(value)
    }

    useImperativeHandle(ref, () => ({
      setInputText: (text: string) => {
        setInputValue(text)
        onInputValueChange?.(text)
        // 聚焦到输入框
        textareaRef.current?.focus()
      },
      getInputValue: () => inputValue,
      getFiles: () => files,
    }))

    const handleFileSelect = async (event: React.ChangeEvent<HTMLInputElement>) => {
      const selectedFiles = event.target.files
      if (!selectedFiles || selectedFiles.length === 0) {
        return
      }

      setUploading(true)

      try {
        const uploadPromises = Array.from(selectedFiles).map(async (file) => {
          try {
            const fileInfo = await fileApi.uploadFile({
              file,
              ...(sessionId && { session_id: sessionId }),
            })
            return fileInfo
          } catch (error) {
            const errorMessage = error instanceof Error ? error.message : '上传失败'
            toast.error(`文件「${file.name}」上传失败: ${errorMessage}`)
            return null
          }
        })

        const uploadedFiles = (await Promise.all(uploadPromises)).filter(
          (file): file is FileInfo => file !== null
        )

        if (uploadedFiles.length > 0) {
          setFiles((prev) => [...prev, ...uploadedFiles])
          toast.success(`成功上传 ${uploadedFiles.length} 个文件`)
        }
      } catch (error) {
        toast.error('文件上传过程中发生错误')
      } finally {
        setUploading(false)
        // 重置input，以便可以重复选择同一文件
        if (fileInputRef.current) {
          fileInputRef.current.value = ''
        }
      }
    }

    const handleUploadClick = () => {
      fileInputRef.current?.click()
    }

    const handleRemoveFile = (fileId: string) => {
      setFiles((prev) => prev.filter((file) => file.id !== fileId))
    }

    const handleSend = async () => {
      const trimmedMessage = inputValue.trim()
      
      // 验证消息不为空
      if (!trimmedMessage) {
        toast.error('请输入消息内容')
        textareaRef.current?.focus()
        return
      }

      // 如果提供了 onSend 回调，使用它
      if (onSend) {
        setSending(true)
        try {
          await onSend(trimmedMessage, files, currentModelId)
          // 发送成功后清空输入框和文件列表
          setInputValue('')
          setFiles([])
          onInputValueChange?.('')
        } catch (error) {
          // 错误处理由 onSend 内部处理
          console.error('发送消息失败:', error)
        } finally {
          setSending(false)
        }
      }
    }

    const handleKeyDown = (e: React.KeyboardEvent<HTMLTextAreaElement>) => {
      // 支持 Ctrl/Cmd + Enter 发送
      if (e.key === 'Enter' && (e.ctrlKey || e.metaKey)) {
        e.preventDefault()
        handleSend()
      }
    }

    return (
    <div className={cn('flex flex-col bg-white w-full rounded-2xl py-3 border shadow-sm', className)}>
      {/* 顶部的文件列表 */}
      {files.length > 0 && (
        <div className="w-full px-4 mb-1">
          <ScrollArea className="w-full whitespace-nowrap">
            <div className="flex w-max space-x-4 pb-4">
              {files.map((file) => (
                <Item
                  key={file.id}
                  variant="muted"
                  className="p-2 flex-shrink-0 gap-2"
                >
                  {/* 左侧文件图标 */}
                  <ItemMedia>
                    <Avatar className="size-8">
                      <AvatarGroupCount>
                        <FileText/>
                      </AvatarGroupCount>
                    </Avatar>
                  </ItemMedia>
                  {/* 文件信息 */}
                  <ItemContent className="gap-0">
                    <ItemTitle className="text-sm text-gray-700">{file.filename}</ItemTitle>
                    <ItemDescription className="text-xs">
                      {file.extension} · {formatFileSize(file.size)}
                    </ItemDescription>
                  </ItemContent>
                  <ItemActions>
                    <Button
                      variant="ghost"
                      size="icon-xs"
                      className="cursor-pointer"
                      onClick={() => handleRemoveFile(file.id)}
                      disabled={uploading}
                    >
                      <XCircle/>
                    </Button>
                  </ItemActions>
                </Item>
              ))}
            </div>
            <ScrollBar orientation="horizontal"/>
          </ScrollArea>
        </div>
      )}
      {/* 中间输入框 */}
      <div className="px-4 mb-3">
        <textarea
          ref={textareaRef}
          rows={2}
          value={inputValue}
          onChange={handleInputChange}
          onKeyDown={handleKeyDown}
          placeholder="分配一个任务或提问任何问题..."
          className="scrollbar-hide outline-none w-full text-sm resize-none h-[46px] min-h-[40px]"
          disabled={sending || disabled}
        />
      </div>
      {/* 底部上传&发送按钮 */}
      <footer className="flex flex-row justify-between items-center w-full px-3 gap-2">
        {/* 左侧：模型选择 + 上传 */}
        <div className="flex gap-2 items-center min-w-0 flex-1">
          <input
            ref={fileInputRef}
            type="file"
            multiple
            className="hidden"
            onChange={handleFileSelect}
            disabled={uploading}
          />
          <Button
            variant="outline"
            className="rounded-full w-8 h-8 cursor-pointer flex-shrink-0"
            onClick={handleUploadClick}
            disabled={uploading}
          >
            {uploading ? (
              <Loader2 className="size-4 animate-spin"/>
            ) : (
              <Paperclip/>
            )}
          </Button>
          {/* 模型选择下拉（原生 select，零依赖；用户切换即下次发送生效） */}
          {models.length > 0 && (
            <select
              value={currentModelId}
              onChange={(e) => setCurrentModelId(e.target.value)}
              disabled={modelsLoading}
              className="text-xs bg-transparent border rounded-full px-2 py-1 max-w-[180px] truncate cursor-pointer hover:bg-gray-50 focus:outline-none focus:ring-1 focus:ring-primary"
              title={models.find((m) => m.id === currentModelId)?.model_name || '选择模型'}
            >
              {models.map((m) => (
                <option key={m.id} value={m.id}>
                  {m.is_default ? '★ ' : ''}{m.name} · {m.model_name}
                </option>
              ))}
            </select>
          )}
        </div>
        {/* 发送/暂停按钮 */}
        <div className="flex gap-2">
          {isRunning ? (
            // 任务运行中时显示暂停按钮
            <Button
              variant="outline"
              className="rounded-full w-8 h-8 cursor-pointer"
              onClick={onStop}
              disabled={!onStop}
            >
              <Pause className="size-4" />
            </Button>
          ) : (
            // 任务未运行时显示发送按钮
            <Button
              variant="outline"
              className="rounded-full w-8 h-8 cursor-pointer"
              onClick={handleSend}
              disabled={sending || disabled || !inputValue.trim()}
            >
              {sending ? (
                <Loader2 className="size-4 animate-spin"/>
              ) : (
                <ArrowUp/>
              )}
            </Button>
          )}
        </div>
      </footer>
    </div>
    )
  }
)

ChatInput.displayName = 'ChatInput'