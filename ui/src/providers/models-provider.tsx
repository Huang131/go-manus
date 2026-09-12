'use client'

import {createContext, useCallback, useContext, useEffect, useMemo, useState} from 'react'
import {configApi} from '@/lib/api'
import type {LLMModel} from '@/lib/api'

/**
 * Auto：跟随系统的特殊选项。
 * 选中时不携带 model_id，由后端按可用性路由（配置文件兜底模型作为最后防线）。
 * 模型选择相关的组件统一引用此常量，避免各处魔法字符串。
 */
export const AUTO_MODEL_ID = '__auto__'

type ModelsContextValue = {
  /** 全部模型（含禁用），消费者按需过滤 is_enabled */
  models: LLMModel[]
  loading: boolean
  /** 手动刷新（模型配置增删改后调用，保持选择器与配置页一致） */
  refresh: () => Promise<void>
}

const ModelsContext = createContext<ModelsContextValue | null>(null)

/**
 * 模型列表 Provider
 *
 * 统一管理模型列表数据：聊天输入框的选择器、模型配置管理页共用同一份，
 * 配置保存后调用 refresh() 即可全局生效，无需刷新页面。
 */
export function ModelsProvider({children}: {children: React.ReactNode}) {
  const [models, setModels] = useState<LLMModel[]>([])
  const [loading, setLoading] = useState(false)

  const refresh = useCallback(async () => {
    setLoading(true)
    try {
      const data = await configApi.listLLMModels()
      setModels(data?.models ?? [])
    } catch {
      // 静默失败：选择器降级为不可用，配置页自行 toast
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    refresh()
  }, [refresh])

  const value = useMemo(() => ({models, loading, refresh}), [models, loading, refresh])

  return <ModelsContext.Provider value={value}>{children}</ModelsContext.Provider>
}

export function useModels(): ModelsContextValue {
  const ctx = useContext(ModelsContext)
  if (!ctx) {
    throw new Error('useModels must be used within <ModelsProvider>')
  }
  return ctx
}
