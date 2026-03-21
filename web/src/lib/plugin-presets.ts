/**
 * 浏览器本地保存的「插件默认配置」草稿，供配置页编辑、管道编辑器拖入节点时合并。
 * （服务端全局插件配置若后续有 API，可再同步到后端。）
 */
const STORAGE_KEY = 'tracker_plugin_presets_v1'

type Store = Record<string, Record<string, unknown>>

function readAll(): Store {
  try {
    const raw = localStorage.getItem(STORAGE_KEY)
    if (!raw) return {}
    const p = JSON.parse(raw) as unknown
    return p && typeof p === 'object' && !Array.isArray(p) ? (p as Store) : {}
  } catch {
    return {}
  }
}

function writeAll(all: Store) {
  localStorage.setItem(STORAGE_KEY, JSON.stringify(all))
}

export function loadPluginPreset(pluginId: string): Record<string, unknown> {
  const all = readAll()
  const v = all[pluginId]
  return v && typeof v === 'object' ? { ...v } : {}
}

export function savePluginPreset(pluginId: string, config: Record<string, unknown>) {
  const all = readAll()
  all[pluginId] = { ...config }
  writeAll(all)
}

export function clearPluginPreset(pluginId: string) {
  const all = readAll()
  delete all[pluginId]
  writeAll(all)
}
