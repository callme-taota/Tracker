import { getPluginConfigComponent } from './registry'
import { DynamicManifestForm } from './DynamicManifestForm'
import { GenericJSONConfig } from './GenericJSONConfig'
import type { PluginConfigProps } from './types'

export type PluginConfigPanelProps = PluginConfigProps & {
  pluginId: string
}

/**
 * 通用插件配置区：优先自定义注册组件 → Manifest 动态表单 → 无 schema 时用 JSON 编辑器。
 */
export function PluginConfigPanel({ pluginId, value, onChange, disabled }: PluginConfigPanelProps) {
  const Custom = getPluginConfigComponent(pluginId)
  if (Custom) {
    return <Custom value={value} onChange={onChange} disabled={disabled} />
  }
  return (
    <DynamicManifestForm
      pluginId={pluginId}
      value={value}
      onChange={onChange}
      disabled={disabled}
      fallback={<GenericJSONConfig value={value} onChange={onChange} disabled={disabled} />}
    />
  )
}
