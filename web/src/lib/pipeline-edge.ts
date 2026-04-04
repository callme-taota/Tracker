import type { Plugin } from '@/api'

/** Mirrors server EdgeFormatsCompatible + validatePipelineEdge. */
export function edgeFormatsCompatible(outF: string[], inF: string[]): boolean {
  if (inF.length === 0) return true
  if (inF.includes('*')) return true
  if (outF.length === 0) return false
  return outF.some((o) => inF.includes(o))
}

export type PluginEdgeMeta = {
  emits_items: boolean
  accepts_items: boolean
  allow_outbound_edges: boolean
  input_formats: string[]
  output_formats: string[]
  compatible_with: string[]
}

function inferFromKind(type_: string): Pick<PluginEdgeMeta, 'emits_items' | 'accepts_items' | 'allow_outbound_edges'> {
  if (type_ === 'source') return { emits_items: true, accepts_items: false, allow_outbound_edges: true }
  if (type_ === 'dispatch') return { emits_items: false, accepts_items: true, allow_outbound_edges: false }
  return { emits_items: true, accepts_items: true, allow_outbound_edges: true }
}

export function pluginsToEdgeMetaMap(plugins: Plugin[]): Map<string, PluginEdgeMeta> {
  const m = new Map<string, PluginEdgeMeta>()
  for (const p of plugins) {
    const fb = inferFromKind(p.type)
    m.set(p.name, {
      emits_items: p.emits_items ?? fb.emits_items,
      accepts_items: p.accepts_items ?? fb.accepts_items,
      allow_outbound_edges: p.allow_outbound_edges ?? fb.allow_outbound_edges,
      input_formats: p.input_formats ?? [],
      output_formats: p.output_formats ?? [],
      compatible_with: p.compatible_with ?? [],
    })
  }
  return m
}

export function validatePipelineConnection(
  sourcePluginId: string,
  targetPluginId: string,
  meta: Map<string, PluginEdgeMeta>,
): { ok: true } | { ok: false; reason: string } {
  const ms = meta.get(sourcePluginId)
  const mt = meta.get(targetPluginId)
  if (!ms || !mt) {
    return { ok: false, reason: '未找到插件连接规则，请刷新插件列表后再试' }
  }
  if (!mt.accepts_items) {
    return { ok: false, reason: `「${targetPluginId}」不接受上游数据流` }
  }
  if (!ms.emits_items) {
    return { ok: false, reason: `「${sourcePluginId}」不产生可传递的数据流` }
  }
  if (!ms.allow_outbound_edges) {
    return { ok: false, reason: `「${sourcePluginId}」不能再连接下游节点` }
  }
  if (!edgeFormatsCompatible(ms.output_formats, mt.input_formats)) {
    return {
      ok: false,
      reason: `数据格式不兼容：${sourcePluginId} 的输出与 ${targetPluginId} 的输入要求不匹配`,
    }
  }
  if (mt.compatible_with.length > 0 && !mt.compatible_with.includes(sourcePluginId)) {
    return {
      ok: false,
      reason: `「${targetPluginId}」仅允许来自：${mt.compatible_with.join('、')}`,
    }
  }
  return { ok: true }
}
