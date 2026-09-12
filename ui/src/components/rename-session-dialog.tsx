'use client'

import {useEffect, useState} from 'react'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import {Button} from '@/components/ui/button'
import {Input} from '@/components/ui/input'

type RenameSessionDialogProps = {
  open: boolean
  initialTitle: string
  onOpenChange: (open: boolean) => void
  onConfirm: (title: string) => Promise<void>
}

/**
 * 会话重命名弹窗
 * 打开时回填当前标题，确认后发起 API 请求
 */
export function RenameSessionDialog({open, initialTitle, onOpenChange, onConfirm}: RenameSessionDialogProps) {
  const [title, setTitle] = useState(initialTitle)
  const [renaming, setRenaming] = useState(false)

  // 每次打开时回填当前标题
  useEffect(() => {
    if (open) {
      setTitle(initialTitle)
    }
  }, [open, initialTitle])

  const trimmed = title.trim()
  const canConfirm = trimmed.length > 0 && trimmed.length <= 100 && trimmed !== initialTitle

  const handleConfirm = async () => {
    if (!canConfirm) return
    setRenaming(true)
    try {
      await onConfirm(trimmed)
    } finally {
      setRenaming(false)
    }
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-[440px]">
        <DialogHeader>
          <DialogTitle className="text-lg font-semibold">重命名任务</DialogTitle>
          <DialogDescription className="text-sm text-muted-foreground">
            为该任务设置一个更容易识别的名称。
          </DialogDescription>
        </DialogHeader>
        <Input
          autoFocus
          value={title}
          maxLength={100}
          placeholder="输入任务名称"
          onChange={(e) => setTitle(e.target.value)}
          onKeyDown={(e) => {
            if (e.key === 'Enter' && canConfirm) {
              handleConfirm()
            }
          }}
          onClick={(e) => e.stopPropagation()}
        />
        <DialogFooter>
          <Button
            variant="outline"
            className="cursor-pointer"
            onClick={() => onOpenChange(false)}
            disabled={renaming}
          >
            取消
          </Button>
          <Button className="cursor-pointer" onClick={handleConfirm} disabled={renaming || !canConfirm}>
            {renaming ? '保存中...' : '确认'}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
