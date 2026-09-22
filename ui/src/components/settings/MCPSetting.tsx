'use client'

import {useState} from 'react'
import {toast} from 'sonner'
import {Loader2, Trash} from 'lucide-react'
import {
  Dialog,
  DialogClose,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from '@/components/ui/dialog'
import {Badge} from '@/components/ui/badge'
import {Button} from '@/components/ui/button'
import {Field, FieldDescription, FieldGroup, FieldLegend, FieldSet} from '@/components/ui/field'
import {Item, ItemContent, ItemGroup, ItemTitle} from '@/components/ui/item'
import {Switch} from '@/components/ui/switch'
import {Textarea} from '@/components/ui/textarea'
import type {ListMCPServerItem, MCPConfig, MCPServerConfig} from '@/lib/api'

export function normalizeMCPConfig(value: unknown): MCPConfig {
  if (!value || typeof value !== 'object') {
    throw new Error('MCP 配置必须是 JSON 对象')
  }
  const raw = value as {servers?: unknown; mcpServers?: unknown}
  if (Array.isArray(raw.servers)) {
    return {servers: raw.servers as MCPServerConfig[]}
  }
  if (!raw.mcpServers || typeof raw.mcpServers !== 'object') {
    throw new Error('MCP 配置必须包含 servers 或 mcpServers')
  }
  const servers = Object.entries(raw.mcpServers as Record<string, MCPServerConfig>).map(([name, config]) => ({
    ...config,
    name,
    enabled: config.enabled ?? true,
  }))
  return {servers}
}

// ==================== MCP 服务器 ====================

type MCPSettingProps = {
  servers: ListMCPServerItem[]
  loading: boolean
  onToggleEnabled: (serverName: string, enabled: boolean) => void
  onDelete: (serverName: string) => void
  onAdd: (config: string) => Promise<boolean>
}

export function MCPSetting({servers, loading, onToggleEnabled, onDelete, onAdd}: MCPSettingProps) {
  const [addDialogOpen, setAddDialogOpen] = useState(false)
  const [addConfig, setAddConfig] = useState('')
  const [adding, setAdding] = useState(false)

  const mcpConfigPlaceholder = `{
  "mcpServers": {
    "qiniu": {
      "command": "uvx",
      "args": [
        "qiniu-mcp-server"
      ],
      "env": {
        "QINIU_ACCESS_KEY": "YOUR_ACCESS_KEY",
        "QINIU_SECRET_KEY": "YOUR_SECRET_KEY"
      }
    }
  }
}`

  const handleAdd = async () => {
    if (!addConfig.trim()) {
      toast.error('请输入 MCP 服务器配置')
      return
    }
    setAdding(true)
    try {
      const success = await onAdd(addConfig.trim())
      if (success) {
        setAddConfig('')
        setAddDialogOpen(false)
      }
    } finally {
      setAdding(false)
    }
  }

  return (
    <div className="w-full px-1">
      <FieldGroup>
        <FieldSet>
          <FieldLegend className="w-full flex justify-between items-center text-lg font-bold text-gray-700">
            MCP 服务器
            <Dialog open={addDialogOpen} onOpenChange={setAddDialogOpen}>
              <DialogTrigger asChild>
                <Button type="button" size="xs" className="cursor-pointer">添加服务器</Button>
              </DialogTrigger>
              <DialogContent>
                <DialogHeader>
                  <DialogTitle className="text-gray-700">添加新的 MCP 服务器</DialogTitle>
                  <DialogDescription className="text-gray-500">
                    Manus 使用标准的 JSON MCP 配置来创建新服务器。
                    请将您的配置粘贴到下方，然后点击 &quot;添加&quot; 即可添加新服务器。
                  </DialogDescription>
                </DialogHeader>
                <form
                  className="w-full"
                  onSubmit={(e) => {
                    e.preventDefault()
                    handleAdd()
                  }}
                >
                  <FieldGroup>
                    <FieldSet>
                      <Field>
                        <Textarea
                          id="mcp_config"
                          placeholder={mcpConfigPlaceholder}
                          value={addConfig}
                          onChange={(e) => setAddConfig(e.target.value)}
                          className="min-h-[200px] font-mono text-xs"
                          disabled={adding}
                        />
                      </Field>
                    </FieldSet>
                  </FieldGroup>
                </form>
                <DialogFooter>
                  <DialogClose asChild>
                    <Button variant="outline" className="cursor-pointer" disabled={adding}>取消</Button>
                  </DialogClose>
                  <Button className="cursor-pointer" onClick={handleAdd} disabled={adding}>
                    {adding && <Loader2 className="animate-spin"/>}
                    添加
                  </Button>
                </DialogFooter>
              </DialogContent>
            </Dialog>
          </FieldLegend>
          <FieldDescription className="text-sm">
            模型上下文协议 (MCP) 通过集成外部工具来增强 Manus 的性能，例如私有域搜索、网页浏览、订餐、PPT 生成等任务。
          </FieldDescription>

          {/* 加载态 */}
          {loading && (
            <div className="flex justify-center py-8">
              <Loader2 className="size-6 animate-spin text-muted-foreground"/>
            </div>
          )}

          {/* 空态 */}
          {!loading && servers.length === 0 && (
            <div className="py-8 text-center text-sm text-muted-foreground">
              暂无 MCP 服务器，请点击上方按钮添加
            </div>
          )}

          {/* 列表 */}
          {!loading && servers.length > 0 && (
            <ItemGroup className="gap-3">
              {servers.map((server) => (
                <Item key={server.name} variant="outline">
                  <ItemContent>
                    <ItemTitle className="w-full flex justify-between items-center text-md font-bold text-gray-700">
                      <div className="flex gap-2 items-center">
                        {server.name}
                        <Badge>stdio</Badge>
                        {!server.enabled && <Badge>禁用</Badge>}
                      </div>
                      <div className="flex items-center justify-center gap-2">
                        <Button
                          type="button"
                          variant="ghost"
                          size="icon-xs"
                          className="cursor-pointer"
                          onClick={() => onDelete(server.name)}
                        >
                          <Trash/>
                        </Button>
                        <Switch
                          checked={server.enabled}
                          onCheckedChange={(checked) => onToggleEnabled(server.name, checked)}
                        />
                      </div>
                    </ItemTitle>
                  </ItemContent>
                </Item>
              ))}
            </ItemGroup>
          )}
        </FieldSet>
      </FieldGroup>
    </div>
  )
}
