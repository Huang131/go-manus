'use client'

import {useCallback, useEffect, useRef, useState} from 'react'
import {toast} from 'sonner'
import {Languages, LayoutGrid, Loader2, Settings, Wrench} from 'lucide-react'
import {
  Dialog,
  DialogClose,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from '@/components/ui/dialog';
import {Button} from '@/components/ui/button'
import {Separator} from '@/components/ui/separator'
import {ModelConfigManager} from '@/components/model-config-manager'
import {CommonSetting} from '@/components/settings/CommonSetting'
import {A2ASetting} from '@/components/settings/A2ASetting'
import {MCPSetting, normalizeMCPConfig} from '@/components/settings/MCPSetting'
import {configApi} from '@/lib/api'
import type {
  AgentConfig,
  ListMCPServerItem,
  ListA2AServerItem,
  CreateA2AServerParams,
} from '@/lib/api'

// ==================== 设置弹窗主组件 ====================

type SettingTab = 'common-setting' | 'llm-setting' | 'a2a-setting' | 'mcp-setting'

const SETTING_MENUS: Array<{
  key: SettingTab
  icon: typeof Settings
  title: string
}> = [
  {key: 'common-setting', icon: Settings, title: '通用配置'},
  {key: 'llm-setting', icon: Languages, title: '模型提供商'},
  {key: 'a2a-setting', icon: LayoutGrid, title: 'A2A Agent 配置'},
  {key: 'mcp-setting', icon: Wrench, title: 'MCP 服务器'},
]

export function ManusSettings() {
  // ---- 防止 SSR hydration 不匹配（Radix Dialog 在服务端/客户端生成不同的 aria-controls ID）----
  const [mounted, setMounted] = useState(false)
  useEffect(() => { setMounted(true) }, [])

  // ---- 弹窗 & 导航 ----
  const [open, setOpen] = useState(false)
  const [activeSetting, setActiveSetting] = useState<SettingTab>('common-setting')

  // ---- 数据 ----
  const [agentConfig, setAgentConfig] = useState<AgentConfig>({})
  const [mcpServers, setMcpServers] = useState<ListMCPServerItem[]>([])
  const [a2aServers, setA2aServers] = useState<ListA2AServerItem[]>([])

  // ---- 状态 ----
  const [loadingConfig, setLoadingConfig] = useState(false)
  const [loadingMCP, setLoadingMCP] = useState(false)
  const [loadingA2A, setLoadingA2A] = useState(false)
  const [saving, setSaving] = useState(false)

  // 防止 Strict Mode 重复获取
  const fetchingRef = useRef(false)

  // ---- 数据拉取（各接口独立请求、独立更新，互不阻塞） ----
  const fetchAllConfigs = useCallback(() => {
    if (fetchingRef.current) return
    fetchingRef.current = true

    // 1. Agent + LLM 配置（通常很快）
    setLoadingConfig(true)
    setLoadingConfig(true)
    configApi.getAgentConfig()
      .then((agent) => {
        setAgentConfig(agent)
      })
      .catch((err) => {
        console.error('[Settings] 获取基础配置失败:', err)
      })
      .finally(() => {
        setLoadingConfig(false)
      })

    // 2. MCP 服务器列表（可能较慢）
    setLoadingMCP(true)
    configApi
      .getMCPServers()
      .then((data) => {
        setMcpServers(data?.servers ?? [])
      })
      .catch((err) => {
        console.error('[Settings] 获取 MCP 服务器列表失败:', err)
      })
      .finally(() => {
        setLoadingMCP(false)
      })

    // 3. A2A 服务器列表
    setLoadingA2A(true)
    configApi
      .getA2AServers()
      .then((data) => {
        setA2aServers(data?.servers ?? [])
      })
      .catch((err) => {
        console.error('[Settings] 获取 A2A 服务器列表失败:', err)
      })
      .finally(() => {
        setLoadingA2A(false)
      })
  }, [])

  // 弹窗打开时拉取数据
  useEffect(() => {
    if (open) {
      fetchAllConfigs()
    } else {
      // 弹窗关闭时重置 ref，下次打开可以重新获取
      fetchingRef.current = false
    }
  }, [open, fetchAllConfigs])

  // ---- 保存 (通用配置 / LLM) ----
  const handleSave = async () => {
    setSaving(true)
    try {
      if (activeSetting === 'common-setting') {
        await configApi.updateAgentConfig(agentConfig)
        toast.success('通用配置保存成功')
      }
    } catch (err) {
      const msg = err instanceof Error ? err.message : '保存失败'
      toast.error(msg)
    } finally {
      setSaving(false)
    }
  }

  // ---- MCP 操作 ----
  const handleMCPToggle = useCallback(async (serverName: string, enabled: boolean) => {
    // 乐观更新
    setMcpServers((prev) =>
      prev.map((s) => (s.name === serverName ? {...s, enabled} : s)),
    )
    try {
      await configApi.updateMCPServerEnabled(serverName, enabled)
      toast.success(`${serverName} 已${enabled ? '启用' : '禁用'}`)
    } catch {
      // 回滚
      setMcpServers((prev) =>
        prev.map((s) => (s.name === serverName ? {...s, enabled: !enabled} : s)),
      )
      toast.error(`操作失败，请重试`)
    }
  }, [])

  const handleMCPDelete = useCallback(async (serverName: string) => {
    const prev = mcpServers
    // 乐观更新
    setMcpServers((list) => list.filter((s) => s.name !== serverName))
    try {
      await configApi.deleteMCPServer(serverName)
      toast.success(`已删除 MCP 服务器「${serverName}」`)
    } catch {
      setMcpServers(prev)
      toast.error(`删除失败，请重试`)
    }
  }, [mcpServers])

  const handleMCPAdd = useCallback(async (configText: string): Promise<boolean> => {
    try {
      const config = normalizeMCPConfig(JSON.parse(configText))
      await configApi.addMCPServer(config)
      toast.success('MCP 服务器添加成功')
      // 重新拉取列表
      try {
        const data = await configApi.getMCPServers()
        setMcpServers(data?.servers ?? [])
      } catch { /* 忽略刷新失败 */ }
      return true
    } catch (err) {
      if (err instanceof SyntaxError) {
        toast.error('JSON 格式错误，请检查配置')
      } else {
        toast.error(err instanceof Error ? err.message : '添加失败')
      }
      return false
    }
  }, [])

  // ---- A2A 操作 ----
  const handleA2AToggle = useCallback(async (id: string, enabled: boolean) => {
    setA2aServers((prev) =>
      prev.map((s) => (s.id === id ? {...s, enabled} : s)),
    )
    try {
      await configApi.updateA2AServerEnabled(id, enabled)
      const server = a2aServers.find((s) => s.id === id)
      toast.success(`${server?.id ?? 'Agent'} 已${enabled ? '启用' : '禁用'}`)
    } catch {
      setA2aServers((prev) =>
        prev.map((s) => (s.id === id ? {...s, enabled: !enabled} : s)),
      )
      toast.error(`操作失败，请重试`)
    }
  }, [a2aServers])

  const handleA2ADelete = useCallback(async (id: string) => {
    const prev = a2aServers
    const target = a2aServers.find((s) => s.id === id)
    setA2aServers((list) => list.filter((s) => s.id !== id))
    try {
      await configApi.deleteA2AServer(id)
      toast.success(`已删除 A2A Agent「${target?.id ?? id}」`)
    } catch {
      setA2aServers(prev)
      toast.error(`删除失败，请重试`)
    }
  }, [a2aServers])

  const handleA2AAdd = useCallback(async (baseUrl: string): Promise<boolean> => {
    try {
      const url = new URL(baseUrl)
      const config: CreateA2AServerParams = {
        servers: [{
          id: `a2a-${crypto.randomUUID()}`,
          url: url.toString(),
          enabled: true,
        }],
      }
      await configApi.addA2AServer(config)
      toast.success('远程 Agent 添加成功')
      // 重新拉取列表
      try {
        const data = await configApi.getA2AServers()
        setA2aServers(data?.servers ?? [])
      } catch { /* 忽略刷新失败 */ }
      return true
    } catch (err) {
      toast.error(err instanceof Error ? err.message : '添加失败')
      return false
    }
  }, [])

  // 客户端挂载前，仅渲染普通按钮占位，避免 Radix Dialog SSR hydration 不匹配
  if (!mounted) {
    return (
      <Button variant="outline" size="icon-sm" className="cursor-pointer">
        <Settings/>
      </Button>
    )
  }

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      {/* 触发按钮 */}
      <DialogTrigger asChild>
        <Button variant="outline" size="icon-sm" className="cursor-pointer">
          <Settings/>
        </Button>
      </DialogTrigger>

      {/* 弹窗内容 */}
      <DialogContent className="!max-w-[850px]">
        {/* 头部 */}
        <DialogHeader className="border-b pb-4">
          <DialogTitle className="text-gray-700">Manus 设置</DialogTitle>
          <DialogDescription className="text-gray-500">在此管理您的 Manus 设置。</DialogDescription>
        </DialogHeader>

        {/* 中间主体 */}
        <div className="flex flex-row gap-4">
          {/* 左侧导航菜单 */}
          <div className="max-w-[180px]">
            <div className="flex flex-col gap-0">
              {SETTING_MENUS.map((menu) => (
                <Button
                  key={menu.key}
                  variant={activeSetting === menu.key ? 'default' : 'ghost'}
                  className="cursor-pointer justify-start"
                  onClick={() => setActiveSetting(menu.key)}
                >
                  <menu.icon/>
                  {menu.title}
                </Button>
              ))}
            </div>
          </div>

          {/* 分隔符 */}
          <Separator orientation="vertical"/>

          {/* 右侧内容 */}
          <div className="flex-1 h-[500px] scrollbar-hide overflow-y-auto">
            {loadingConfig && activeSetting === 'common-setting' ? (
              <div className="flex justify-center items-center h-full">
                <Loader2 className="size-6 animate-spin text-muted-foreground"/>
              </div>
            ) : (
              <>
                {activeSetting === 'common-setting' && (
                  <CommonSetting config={agentConfig} onChange={setAgentConfig}/>
                )}
                {activeSetting === 'llm-setting' && (
                  <ModelConfigManager/>
                )}
              </>
            )}
            {activeSetting === 'a2a-setting' && (
              <A2ASetting
                servers={a2aServers}
                loading={loadingA2A}
                onToggleEnabled={handleA2AToggle}
                onDelete={handleA2ADelete}
                onAdd={handleA2AAdd}
              />
            )}
            {activeSetting === 'mcp-setting' && (
              <MCPSetting
                servers={mcpServers}
                loading={loadingMCP}
                onToggleEnabled={handleMCPToggle}
                onDelete={handleMCPDelete}
                onAdd={handleMCPAdd}
              />
            )}
          </div>
        </div>

        {/* 底部按钮（llm-setting 内部已自管保存，禁用底部） */}
        <DialogFooter className="border-t pt-4">
          <DialogClose asChild>
            <Button variant="outline" className="cursor-pointer">关闭</Button>
          </DialogClose>
          <Button
            className="cursor-pointer"
            disabled={saving || activeSetting === 'llm-setting'}
            onClick={handleSave}
          >
            {saving && <Loader2 className="animate-spin"/>}
            保存
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
