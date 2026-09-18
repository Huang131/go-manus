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
import {Input} from '@/components/ui/input'
import {Item, ItemContent, ItemGroup, ItemTitle} from '@/components/ui/item'
import {Switch} from '@/components/ui/switch'
import type {ListA2AServerItem} from '@/lib/api'

// ==================== A2A Agent 配置 ====================

type A2ASettingProps = {
  servers: ListA2AServerItem[]
  loading: boolean
  onToggleEnabled: (id: string, enabled: boolean) => void
  onDelete: (id: string) => void
  onAdd: (baseUrl: string) => Promise<boolean>
}

export function A2ASetting({servers, loading, onToggleEnabled, onDelete, onAdd}: A2ASettingProps) {
  const [addDialogOpen, setAddDialogOpen] = useState(false)
  const [addUrl, setAddUrl] = useState('')
  const [adding, setAdding] = useState(false)

  const handleAdd = async () => {
    if (!addUrl.trim()) {
      toast.error('请输入远程 Agent 地址')
      return
    }
    setAdding(true)
    try {
      const success = await onAdd(addUrl.trim())
      if (success) {
        setAddUrl('')
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
            A2A Agent 配置
            <Dialog open={addDialogOpen} onOpenChange={setAddDialogOpen}>
              <DialogTrigger asChild>
                <Button type="button" size="xs" className="cursor-pointer">添加远程Agent</Button>
              </DialogTrigger>
              <DialogContent>
                <DialogHeader>
                  <DialogTitle className="text-gray-700">添加远程Agent</DialogTitle>
                  <DialogDescription className="text-gray-500">
                    Manus 使用标准的 A2A 协议来连接远程 Agent。
                    <br/>
                    请将您的配置粘贴到下方，然后点击 &quot;添加&quot; 即可添加 Agent。
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
                        <Input
                          id="a2a_base_url"
                          type="url"
                          placeholder="Example: https://mooc-manus.com/weather-agent"
                          value={addUrl}
                          onChange={(e) => setAddUrl(e.target.value)}
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
            A2A（Agent-to-Agent）协议用于对接外部智能体服务，配置后可在对话中调用其他 Agent 的能力。
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
              暂无 A2A Agent，请点击上方按钮添加
            </div>
          )}

          {/* 列表 */}
          {!loading && servers.length > 0 && (
            <ItemGroup className="gap-3">
              {servers.map((server) => (
                <Item key={server.id} variant="outline">
                  <ItemContent>
                    <ItemTitle className="w-full flex justify-between items-center text-md font-bold text-gray-700">
                      <div className="flex gap-2 items-center">
                        {server.id}
                        {!server.enabled && <Badge>禁用</Badge>}
                      </div>
                      <div className="flex items-center justify-center gap-2">
                        <Button
                          type="button"
                          variant="ghost"
                          size="icon-xs"
                          className="cursor-pointer"
                          onClick={() => onDelete(server.id)}
                        >
                          <Trash/>
                        </Button>
                        <Switch
                          checked={server.enabled}
                          onCheckedChange={(checked) => onToggleEnabled(server.id, checked)}
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
