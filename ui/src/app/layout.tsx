import React from 'react'
import type {Metadata} from 'next'
import {SidebarProvider} from '@/components/ui/sidebar'
import {SessionsProvider} from '@/providers/sessions-provider'
import {ModelsProvider} from '@/providers/models-provider'
import {Toaster} from '@/components/ui/sonner'
import './globals.css'
import {LeftPanel} from '@/components/left-panel'
import {GlobalHeader} from '@/components/global-header'

export const metadata: Metadata = {
  title: 'Manus',
  description: 'Manus 是一个行动引擎，它超越了答案的范畴，可以执行任务、自动化工作流程，并扩展您的能力。',
  icons: {
    icon: '/icon.png',
  },
}

export default function RootLayout(
  {
    children,
  }: Readonly<{
    children: React.ReactNode;
  }>,
) {
  return (
    <html lang="zh-CN" suppressHydrationWarning>
    <body className="h-dvh overflow-hidden">
    <ModelsProvider>
    <SessionsProvider>
      <SidebarProvider
        className="h-full !min-h-0"
        style={{
          // eslint-disable-next-line @typescript-eslint/ban-ts-comment
          // @ts-expect-error
          '--sidebar-width': '300px',
          '--sidebar-width-icon': '300px',
        }}
      >
        {/* 左侧的面板 */}
        <LeftPanel/>
        {/* 右侧的内容 */}
        <div className="flex-1 bg-[#f8f8f7] overflow-hidden flex flex-col min-h-0 h-full">
          <GlobalHeader/>
          <div className="flex-1 overflow-hidden min-h-0 h-full">
            {children}
          </div>
        </div>
      </SidebarProvider>
    </SessionsProvider>
    </ModelsProvider>
    <Toaster position="top-center" richColors/>
    </body>
    </html>
  )
}
