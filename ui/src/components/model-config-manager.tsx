'use client'

import { useState } from 'react'
import { toast } from 'sonner'
import { useModels } from '@/providers/models-provider'
import { Loader2, Pencil, Plus, Star, Trash2 } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Badge } from '@/components/ui/badge'
import { Switch } from '@/components/ui/switch'
import {
  Dialog,
  DialogClose,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import {
  Field,
  FieldDescription,
  FieldGroup,
  FieldLabel,
  FieldSet,
} from '@/components/ui/field'
import { configApi } from '@/lib/api'
import type { LLMModel } from '@/lib/api'

type Mode = 'create' | 'edit' | null

const emptyDraft: Partial<LLMModel> = {
  name: '',
  provider: 'openai',
  base_url: '',
  api_key: '',
  model_name: '',
  temperature: 0.7,
  max_tokens: 8192,
  is_enabled: true,
}

export function ModelConfigManager() {
  // 模型列表来自全局 ModelsProvider：配置变更后 refresh()，
  // 聊天输入框的选择器同步生效，无需刷新页面。
  const {models, loading, refresh} = useModels()
  const [mode, setMode] = useState<Mode>(null)
  const [draft, setDraft] = useState<Partial<LLMModel>>(emptyDraft)
  const [saving, setSaving] = useState(false)

  const openCreate = () => {
    setDraft({ ...emptyDraft })
    setMode('create')
  }
  const openEdit = (m: LLMModel) => {
    setDraft({ ...m, api_key: '' }) // 不预填，保留旧 key
    setMode('edit')
  }
  const closeDialog = () => {
    if (!saving) setMode(null)
  }

  const handleSave = async () => {
    if (!draft.name || !draft.provider || !draft.base_url || !draft.model_name) {
      toast.error('名称 / 提供商 / 地址 / 模型名 必填')
      return
    }
    setSaving(true)
    try {
      if (mode === 'create') {
        await configApi.createLLMModel(draft)
        toast.success('模型已添加')
      } else if (mode === 'edit' && draft.id) {
        await configApi.updateLLMModel(draft.id, draft)
        toast.success('模型已更新')
      }
      setMode(null)
      await refresh()
    } catch (e) {
      const msg = e instanceof Error ? e.message : '保存失败'
      toast.error(msg)
    } finally {
      setSaving(false)
    }
  }

  const handleDelete = async (m: LLMModel) => {
    if (!confirm(`确定删除「${m.name}」？`)) return
    try {
      await configApi.deleteLLMModel(m.id)
      toast.success('已删除')
      await refresh()
    } catch (e) {
      const msg = e instanceof Error ? e.message : '删除失败'
      toast.error(msg)
    }
  }

  const handleSetDefault = async (m: LLMModel) => {
    try {
      await configApi.setDefaultLLMModel(m.id)
      toast.success(`已将「${m.name}」设为默认`)
      await refresh()
    } catch (e) {
      const msg = e instanceof Error ? e.message : '设置默认失败'
      toast.error(msg)
    }
  }

  const handleToggleEnabled = async (m: LLMModel, enabled: boolean) => {
    try {
      await configApi.updateLLMModel(m.id, { ...m, is_enabled: enabled })
      await refresh()
    } catch (e) {
      const msg = e instanceof Error ? e.message : '更新失败'
      toast.error(msg)
    }
  }

  return (
    <div className="w-full px-1 space-y-4">
      <div className="flex items-center justify-between">
        <h3 className="text-lg font-bold text-gray-700">模型列表</h3>
        <Button size="sm" onClick={openCreate} className="cursor-pointer">
          <Plus className="size-4" /> 添加模型
        </Button>
      </div>

      {loading ? (
        <div className="flex justify-center py-8">
          <Loader2 className="size-6 animate-spin text-muted-foreground" />
        </div>
      ) : models.length === 0 ? (
        <div className="py-8 text-center text-sm text-muted-foreground">
          暂无模型，请点击右上角添加
        </div>
      ) : (
        <div className="space-y-2">
          {models.map((m) => (
            <div
              key={m.id}
              className="border rounded-lg p-3 flex items-center gap-3"
            >
              <div className="flex-1 min-w-0">
                <div className="flex items-center gap-2">
                  <span className="font-medium text-gray-800 truncate">{m.name}</span>
                  {m.is_default && <Badge variant="default">默认</Badge>}
                  {!m.is_enabled && <Badge variant="secondary">已停用</Badge>}
                </div>
                <div className="text-xs text-muted-foreground mt-0.5 truncate">
                  {m.provider} · {m.model_name}
                </div>
                <div className="text-xs text-muted-foreground truncate">
                  {m.base_url}
                </div>
              </div>
              <Switch
                checked={m.is_enabled}
                onCheckedChange={(v) => handleToggleEnabled(m, v)}
              />
              {!m.is_default && (
                <Button
                  variant="ghost"
                  size="icon-sm"
                  className="cursor-pointer"
                  onClick={() => handleSetDefault(m)}
                  title="设为默认"
                >
                  <Star />
                </Button>
              )}
              <Button
                variant="ghost"
                size="icon-sm"
                className="cursor-pointer"
                onClick={() => openEdit(m)}
                title="编辑"
              >
                <Pencil />
              </Button>
              <Button
                variant="ghost"
                size="icon-sm"
                className="cursor-pointer"
                onClick={() => handleDelete(m)}
                title="删除"
                disabled={m.is_default}
              >
                <Trash2 />
              </Button>
            </div>
          ))}
        </div>
      )}

      <Dialog open={mode !== null} onOpenChange={(o) => { if (!o) closeDialog() }}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{mode === 'edit' ? '编辑模型' : '添加模型'}</DialogTitle>
            <DialogDescription>
              配置大语言模型的连接信息。带 <span className="text-red-500">*</span> 为必填。
            </DialogDescription>
          </DialogHeader>
          <form
            className="w-full"
            onSubmit={(e) => {
              e.preventDefault()
              handleSave()
            }}
          >
            <FieldGroup>
              <FieldSet>
                <Field>
                  <FieldLabel htmlFor="m-name">显示名 <span className="text-red-500">*</span></FieldLabel>
                  <Input
                    id="m-name"
                    value={draft.name ?? ''}
                    onChange={(e) => setDraft({ ...draft, name: e.target.value })}
                    placeholder="例如：我的 Claude"
                  />
                </Field>
                <Field>
                  <FieldLabel htmlFor="m-provider">提供商 <span className="text-red-500">*</span></FieldLabel>
                  <Input
                    id="m-provider"
                    value={draft.provider ?? ''}
                    onChange={(e) => setDraft({ ...draft, provider: e.target.value })}
                    placeholder="openai / anthropic / deepseek / custom"
                  />
                </Field>
                <Field>
                  <FieldLabel htmlFor="m-baseurl">基础 URL <span className="text-red-500">*</span></FieldLabel>
                  <Input
                    id="m-baseurl"
                    value={draft.base_url ?? ''}
                    onChange={(e) => setDraft({ ...draft, base_url: e.target.value })}
                    placeholder="https://api.openai.com/v1"
                  />
                </Field>
                <Field>
                  <FieldLabel htmlFor="m-apikey">API Key</FieldLabel>
                  <Input
                    id="m-apikey"
                    type="password"
                    value={draft.api_key ?? ''}
                    onChange={(e) => setDraft({ ...draft, api_key: e.target.value })}
                    placeholder={mode === 'edit' ? '留空则保留旧 Key' : '请输入 API Key'}
                  />
                </Field>
                <Field>
                  <FieldLabel htmlFor="m-modelname">模型名 <span className="text-red-500">*</span></FieldLabel>
                  <Input
                    id="m-modelname"
                    value={draft.model_name ?? ''}
                    onChange={(e) => setDraft({ ...draft, model_name: e.target.value })}
                    placeholder="例如 gpt-4o / claude-3-5-sonnet-20241022"
                  />
                </Field>
                <Field>
                  <FieldLabel htmlFor="m-temp">温度</FieldLabel>
                  <Input
                    id="m-temp"
                    type="number"
                    step={0.1}
                    min={0}
                    max={2}
                    value={draft.temperature ?? 0.7}
                    onChange={(e) => setDraft({ ...draft, temperature: Number(e.target.value) })}
                  />
                  <FieldDescription className="text-xs">0 = 稳定，2 = 随机。默认 0.7。</FieldDescription>
                </Field>
                <Field>
                  <FieldLabel htmlFor="m-maxtokens">最大输出 Token</FieldLabel>
                  <Input
                    id="m-maxtokens"
                    type="number"
                    min={1}
                    max={128000}
                    value={draft.max_tokens ?? 8192}
                    onChange={(e) => setDraft({ ...draft, max_tokens: Number(e.target.value) })}
                  />
                </Field>
              </FieldSet>
            </FieldGroup>
          </form>
          <DialogFooter>
            <DialogClose asChild>
              <Button variant="outline" className="cursor-pointer" disabled={saving}>取消</Button>
            </DialogClose>
            <Button className="cursor-pointer" onClick={handleSave} disabled={saving}>
              {saving && <Loader2 className="animate-spin" />}
              保存
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  )
}
