'use client'

import Link from 'next/link'
import {Home} from 'lucide-react'
import {SidebarTrigger, useSidebar} from '@/components/ui/sidebar'
import {Button} from '@/components/ui/button'
import {ManusSettings} from '@/components/manus-settings'

/**
 * 全局 Header（仅在左侧面板关闭/移动端时显示 logo 和齿轮）
 * - 任意页面右上角都有「Manus 设置」入口
 * - 与 ChatHeader/SessionHeader 职责解耦：仅做"全局"那一部分
 */
export function GlobalHeader() {
  const {open, isMobile} = useSidebar()

  return (
    <header className="flex justify-between items-center w-full py-2 px-4 z-50">
      <div className="flex items-center gap-2">
        {(!open || isMobile) && <SidebarTrigger className="cursor-pointer"/>}
        <Button
          asChild
          variant="ghost"
          size="icon-sm"
          className="cursor-pointer"
          title="返回首页"
        >
          <Link href="/" aria-label="返回首页">
            <Home/>
          </Link>
        </Button>
      </div>
      <ManusSettings/>
    </header>
  )
}
