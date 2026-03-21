import type { ComponentType } from 'react'
import type { PluginConfigProps } from './types'

const registry = new Map<string, ComponentType<PluginConfigProps>>()

/** 插件在前端注册自定义配置表单（无注册则使用 Manifest 动态表单） */
export function registerPluginUI(pluginId: string, Comp: ComponentType<PluginConfigProps>) {
  registry.set(pluginId, Comp)
}

export function getPluginConfigComponent(pluginId: string): ComponentType<PluginConfigProps> | null {
  return registry.get(pluginId) ?? null
}
