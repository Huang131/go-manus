'use client'

import {
  Field,
  FieldDescription,
  FieldGroup,
  FieldLabel,
  FieldLegend,
  FieldSet,
} from '@/components/ui/field'
import {Input} from '@/components/ui/input'
import type {AgentConfig} from '@/lib/api'

// ==================== 通用配置 ====================

type CommonSettingProps = {
  config: AgentConfig | null
  onChange: (config: AgentConfig) => void
}

export function CommonSetting({config, onChange}: CommonSettingProps) {
  // 后端在 app_configs 表为空时返回 null，避免访问 null 字段崩溃
  const safeConfig = config ?? {}
  const handleChange = (field: keyof AgentConfig, value: string) => {
    const numValue = value === '' ? undefined : Number(value)
    onChange({...safeConfig, [field]: numValue})
  }

  return (
    <form className="w-full px-1" onSubmit={(e) => e.preventDefault()}>
      <FieldGroup>
        <FieldSet>
          <FieldLegend className="text-lg font-bold text-gray-700">通用配置</FieldLegend>
          <FieldGroup>
            <Field>
              <FieldLabel htmlFor="max_iterations">最大计划迭代次数</FieldLabel>
              <Input
                id="max_iterations"
                type="number"
                placeholder="Agent最大迭代次数"
                value={safeConfig.max_iterations ?? 100}
                onChange={(e) => handleChange('max_iterations', e.target.value)}
                min={0}
                max={200}
              />
              <FieldDescription className="text-xs">
                执行Agent最大能迭代循环调用工具的次数，默认为100
              </FieldDescription>
            </Field>
            <Field>
              <FieldLabel htmlFor="max_retries">最大重试次数</FieldLabel>
              <Input
                id="max_retries"
                type="number"
                placeholder="LLM/Tool最大重试次数"
                value={safeConfig.max_retries ?? 3}
                onChange={(e) => handleChange('max_retries', e.target.value)}
                min={0}
                max={10}
              />
              <FieldDescription className="text-xs">
                默认情况下，最大重试次数为3
              </FieldDescription>
            </Field>
            <Field>
              <FieldLabel htmlFor="max_search_results">最大搜索结果</FieldLabel>
              <Input
                id="max_search_results"
                type="number"
                placeholder="搜索工具返回的最大结果数"
                value={safeConfig.max_search_results ?? 10}
                onChange={(e) => handleChange('max_search_results', e.target.value)}
                min={0}
                max={30}
              />
              <FieldDescription className="text-xs">
                默认情况下，每个搜索步骤包含 10 个结果。
              </FieldDescription>
            </Field>
          </FieldGroup>
        </FieldSet>
      </FieldGroup>
    </form>
  )
}

// 将常见的 mcpServers 配置格式转换为后端统一的 servers 数组。
