import { useCallback, useEffect, useMemo, useState, type Dispatch, type DragEvent, type MouseEvent, type SetStateAction } from 'react'
import {
  ReactFlow,
  ReactFlowProvider,
  Background,
  Controls,
  MiniMap,
  useNodesState,
  useEdgesState,
  addEdge,
  Handle,
  Position,
  useReactFlow,
  type Connection,
  type Edge,
  type Node,
  type NodeProps,
} from '@xyflow/react'
import '@xyflow/react/dist/style.css'
import { Link, useParams } from 'react-router-dom'
import { Play, Save, FlaskConical, ArrowLeft, ExternalLink, Zap, Trash2, Boxes } from 'lucide-react'
import { api, type GraphEdgeDTO, type GraphGroupRefDTO, type GraphNodeDTO, type PipelineGraphDTO, type Plugin, type PluginGroup } from '@/api'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Checkbox } from '@/components/ui/checkbox'
import { Separator } from '@/components/ui/separator'
import { Badge } from '@/components/ui/badge'
import { Sheet, SheetContent, SheetHeader, SheetTitle } from '@/components/ui/sheet'
import { cn } from '@/lib/utils'
import { loadPluginPreset } from '@/lib/plugin-presets'
import { pluginsToEdgeMetaMap, validatePipelineConnection } from '@/lib/pipeline-edge'
import { PluginConfigPanel } from '@/plugin-ui/PluginConfigPanel'

const DND_MIME = 'application/tracker-plugin'
const DND_GROUP_MIME = 'application/tracker-plugin-group'

type PalettePayload = { plugin_id: string; plugin_type: string }
type GroupPalettePayload = { group_id: number }

function PluginNode({ data, selected }: NodeProps) {
  const t = String(data.plugin_type ?? '')
  const border =
    t === 'source'
      ? 'border-emerald-500'
      : t === 'dispatch'
        ? 'border-orange-500'
        : t === 'summary'
          ? 'border-violet-500'
          : t === 'interest'
            ? 'border-amber-600'
            : 'border-sky-600'
  return (
    <div
      className={cn(
        'min-w-[150px] rounded-lg border-2 bg-card px-3 py-2 shadow-sm',
        border,
        selected && 'ring-2 ring-ring ring-offset-2 ring-offset-background',
      )}
    >
      <Handle type="target" position={Position.Left} className="!h-2.5 !w-2.5 !bg-muted-foreground" />
      <Badge variant="outline" className="mb-1 text-[10px] uppercase">
        {t}
      </Badge>
      <div className="font-semibold leading-tight">{String(data.plugin_id ?? '')}</div>
      <div className="truncate text-xs text-muted-foreground">{String(data.label ?? data.id ?? '')}</div>
      <Handle type="source" position={Position.Right} className="!h-2.5 !w-2.5 !bg-muted-foreground" />
    </div>
  )
}

const nodeTypes = { plugin: PluginNode }

function graphToFlow(g: PipelineGraphDTO): { nodes: Node[]; edges: Edge[]; groupRefs: GraphGroupRefDTO[] } {
  const nodes: Node[] = (g.nodes ?? []).map((n) => ({
    id: n.id,
    type: 'plugin',
    position: n.position ?? { x: 0, y: 0 },
    data: {
      plugin_type: n.plugin_type,
      plugin_id: n.plugin_id,
      config: n.config ?? {},
      label: n.id,
    },
  }))
  const edges: Edge[] = (g.edges ?? []).map((e) => ({
    id: e.id,
    source: e.source,
    target: e.target,
    sourceHandle: e.sourceHandle,
    targetHandle: e.targetHandle,
    data: e.on_condition ? { on_condition: e.on_condition } : undefined,
  }))
  return { nodes, edges, groupRefs: g.group_refs ?? [] }
}

function flowToGraph(name: string, nodes: Node[], edges: Edge[], groupRefs: GraphGroupRefDTO[]): PipelineGraphDTO {
  const gn: GraphNodeDTO[] = nodes.map((n) => ({
    id: n.id,
    plugin_type: String(n.data?.plugin_type ?? 'processor'),
    plugin_id: String(n.data?.plugin_id ?? 'rss'),
    config: (n.data?.config as Record<string, unknown>) ?? {},
    position: { x: n.position.x, y: n.position.y },
  }))
  const ge: GraphEdgeDTO[] = edges.map((e) => ({
    id: e.id,
    source: e.source,
    target: e.target,
    sourceHandle: e.sourceHandle ?? undefined,
    targetHandle: e.targetHandle ?? undefined,
    on_condition: typeof e.data?.on_condition === 'string' ? e.data.on_condition : undefined,
  }))
  const validNodeIDs = new Set(gn.map((n) => n.id))
  return {
    name,
    nodes: gn,
    edges: ge,
    group_refs: groupRefs
      .map((ref) => ({ ...ref, node_ids: (ref.node_ids ?? []).filter((id) => validNodeIDs.has(id)) }))
      .filter((ref) => (ref.node_ids?.length ?? 0) > 0),
  }
}

let nodeSeq = 0
function nextNodeId() {
  nodeSeq += 1
  return `n_${Date.now()}_${nodeSeq}`
}

function expandPluginGroup(group: PluginGroup, position: { x: number; y: number }): { nodes: Node[]; edges: Edge[]; groupRef?: GraphGroupRefDTO } {
  const src = group.graph
  const minX = Math.min(...(src.nodes ?? []).map((n) => n.position?.x ?? 0), 0)
  const minY = Math.min(...(src.nodes ?? []).map((n) => n.position?.y ?? 0), 0)
  const idMap = new Map<string, string>()
  const nodes: Node[] = (src.nodes ?? []).map((n) => {
    const id = nextNodeId()
    idMap.set(n.id, id)
    return {
      id,
      type: 'plugin',
      position: {
        x: position.x + ((n.position?.x ?? 0) - minX),
        y: position.y + ((n.position?.y ?? 0) - minY),
      },
      data: {
        plugin_id: n.plugin_id,
        plugin_type: n.plugin_type,
        config: n.config ?? {},
        label: `${group.name}:${n.id}`,
      },
    }
  })
  const edges: Edge[] = (src.edges ?? []).map((e, idx) => ({
    id: `group_${group.id}_${idx}_${Date.now()}`,
    source: idMap.get(e.source) ?? e.source,
    target: idMap.get(e.target) ?? e.target,
    sourceHandle: e.sourceHandle,
    targetHandle: e.targetHandle,
    data: e.on_condition ? { on_condition: e.on_condition } : undefined,
  }))
  return {
    nodes,
    edges,
    groupRef:
      group.current_version_id != null
        ? {
            group_id: group.id,
            group_name: group.name,
            group_version_id: group.current_version_id,
            group_version: group.current_version,
            node_ids: nodes.map((n) => n.id),
          }
        : undefined,
  }
}

function FlowCanvas({
  nodes,
  edges,
  onNodesChange,
  onEdgesChange,
  onConnect,
  isValidConnection,
  onNodeClick,
  onPaneClick,
  setNodes,
  setEdges,
  setGroupRefs,
  groups,
}: {
  nodes: Node[]
  edges: Edge[]
  onNodesChange: ReturnType<typeof useNodesState>[2]
  onEdgesChange: ReturnType<typeof useEdgesState>[2]
  onConnect: (c: Connection) => void
  isValidConnection?: (c: Connection | Edge) => boolean
  onNodeClick: (_: MouseEvent, node: Node) => void
  onPaneClick: () => void
  setNodes: ReturnType<typeof useNodesState>[1]
  setEdges: ReturnType<typeof useEdgesState>[1]
  setGroupRefs: Dispatch<SetStateAction<GraphGroupRefDTO[]>>
  groups: PluginGroup[]
}) {
  const { screenToFlowPosition } = useReactFlow()

  const onDragOver = useCallback((e: DragEvent) => {
    e.preventDefault()
    e.dataTransfer.dropEffect = 'copy'
  }, [])

  const onDrop = useCallback(
    (e: DragEvent) => {
      e.preventDefault()
      const position = screenToFlowPosition({ x: e.clientX, y: e.clientY })
      const groupRaw = e.dataTransfer.getData(DND_GROUP_MIME)
      if (groupRaw) {
        let payload: GroupPalettePayload
        try {
          payload = JSON.parse(groupRaw) as GroupPalettePayload
        } catch {
          return
        }
        const group = groups.find((g) => g.id === payload.group_id)
        if (!group) return
        const expanded = expandPluginGroup(group, position)
        setNodes((nds) => [...nds, ...expanded.nodes])
        setEdges((eds) => [...eds, ...expanded.edges])
        if (expanded.groupRef) setGroupRefs((refs) => [...refs, expanded.groupRef!])
        return
      }
      const raw = e.dataTransfer.getData(DND_MIME)
      if (!raw) return
      let p: PalettePayload
      try {
        p = JSON.parse(raw) as PalettePayload
      } catch {
        return
      }
      if (!p.plugin_id) return
      const id = nextNodeId()
      const preset = loadPluginPreset(p.plugin_id)
      setNodes((nds) => [
        ...nds,
        {
          id,
          type: 'plugin',
          position,
          data: {
            plugin_id: p.plugin_id,
            plugin_type: p.plugin_type || 'processor',
            config: { ...preset },
            label: id,
          },
        },
      ])
    },
    [groups, screenToFlowPosition, setEdges, setGroupRefs, setNodes],
  )

  return (
    <ReactFlow
      nodes={nodes}
      edges={edges}
      onNodesChange={onNodesChange}
      onEdgesChange={onEdgesChange}
      onConnect={onConnect}
      isValidConnection={isValidConnection}
      onNodeClick={onNodeClick}
      onPaneClick={onPaneClick}
      onDrop={onDrop}
      onDragOver={onDragOver}
      nodeTypes={nodeTypes}
      nodesDraggable
      nodesConnectable
      elementsSelectable
      panOnScroll
      fitView
      deleteKeyCode={['Backspace', 'Delete']}
      className="h-full w-full touch-none bg-muted/40"
      proOptions={{ hideAttribution: true }}
    >
      <Background gap={16} />
      <Controls />
      <MiniMap className="!bg-card" zoomable pannable />
    </ReactFlow>
  )
}

function PipelineEditorInner({ pipelineId }: { pipelineId: number }) {
  const [pipeName, setPipeName] = useState('default')
  const [isDefault, setIsDefault] = useState(false)
  const [plugins, setPlugins] = useState<Plugin[]>([])
  const [pluginGroups, setPluginGroups] = useState<PluginGroup[]>([])
  const [loading, setLoading] = useState(true)
  const [notice, setNotice] = useState<{ kind: 'ok' | 'err'; text: string } | null>(null)
  const [selectedNodeId, setSelectedNodeId] = useState<string | null>(null)
  const [nodeTestLoading, setNodeTestLoading] = useState(false)
  const [nodeTestMsg, setNodeTestMsg] = useState<string | null>(null)
  const [groupRefs, setGroupRefs] = useState<GraphGroupRefDTO[]>([])

  const [nodes, setNodes, onNodesChange] = useNodesState<Node>([])
  const [edges, setEdges, onEdgesChange] = useEdgesState<Edge>([])

  const pluginEdgeMeta = useMemo(() => pluginsToEdgeMetaMap(plugins), [plugins])

  const isValidConnection = useCallback(
    (c: Connection | Edge) => {
      if (pluginEdgeMeta.size === 0) return true
      const srcNode = nodes.find((n) => n.id === c.source)
      const tgtNode = nodes.find((n) => n.id === c.target)
      if (!srcNode || !tgtNode) return false
      const sp = String(srcNode.data?.plugin_id ?? '')
      const tp = String(tgtNode.data?.plugin_id ?? '')
      const r = validatePipelineConnection(sp, tp, pluginEdgeMeta)
      if (!r.ok) {
        setNotice({ kind: 'err', text: r.reason })
        return false
      }
      return true
    },
    [nodes, pluginEdgeMeta],
  )

  const onConnect = useCallback(
    (c: Connection) =>
      setEdges((eds) => addEdge({ ...c, id: `e_${c.source}_${c.target}_${eds.length}` }, eds)),
    [setEdges],
  )

  useEffect(() => {
    api.getPlugins().then(setPlugins).catch(() => {})
    api.listPluginGroups().then(setPluginGroups).catch(() => {})
  }, [])

  useEffect(() => {
    let cancelled = false
    setLoading(true)
    ;(async () => {
      try {
        const d = await api.getPipeline(pipelineId)
        if (cancelled) return
        setPipeName(d.name)
        setIsDefault(d.is_default)
        const { nodes: n, edges: e, groupRefs: refs } = graphToFlow(d.graph)
        setNodes(n)
        setEdges(e)
        setGroupRefs(refs)
      } catch (err) {
        if (!cancelled) {
          setNotice({ kind: 'err', text: String(err) })
          setNodes([])
          setEdges([])
          setGroupRefs([])
        }
      } finally {
        if (!cancelled) setLoading(false)
      }
    })()
    return () => {
      cancelled = true
    }
  }, [pipelineId, setNodes, setEdges])

  const selectedNode = useMemo(
    () => (selectedNodeId ? nodes.find((n) => n.id === selectedNodeId) ?? null : null),
    [nodes, selectedNodeId],
  )

  const pluginId = selectedNode ? String(selectedNode.data?.plugin_id ?? '') : ''
  const nodeConfig = (selectedNode?.data?.config as Record<string, unknown>) ?? {}

  const testNodePlugin = async () => {
    if (!pluginId) return
    setNodeTestLoading(true)
    setNodeTestMsg(null)
    try {
      const r = await api.testPluginConfig(pluginId, nodeConfig)
      if (!r.supported) {
        setNodeTestMsg('该插件不支持在线测试')
        return
      }
      setNodeTestMsg(r.ok ? '测试通过' : (r.error ?? '测试失败'))
    } catch (e) {
      setNodeTestMsg(String(e))
    } finally {
      setNodeTestLoading(false)
    }
  }

  const removeSelectedNode = useCallback(() => {
    if (!selectedNodeId) return
    const removingId = selectedNodeId
    setNodes((nds) => nds.filter((n) => n.id !== removingId))
    setEdges((eds) => eds.filter((e) => e.source !== removingId && e.target !== removingId))
    setGroupRefs((refs) =>
      refs
        .map((ref) => ({ ...ref, node_ids: (ref.node_ids ?? []).filter((id) => id !== removingId) }))
        .filter((ref) => (ref.node_ids?.length ?? 0) > 0),
    )
    setSelectedNodeId(null)
    setNodeTestMsg(null)
  }, [selectedNodeId, setEdges, setNodes])

  const updateSelectedConfig = useCallback(
    (cfg: Record<string, unknown>) => {
      if (!selectedNodeId) return
      setNodes((nds) =>
        nds.map((n) => (n.id === selectedNodeId ? { ...n, data: { ...n.data, config: cfg } } : n)),
      )
    },
    [selectedNodeId, setNodes],
  )

  const grouped = useMemo(() => {
    const m = new Map<string, Plugin[]>()
    for (const p of plugins) {
      const k = p.type || 'other'
      if (!m.has(k)) m.set(k, [])
      m.get(k)!.push(p)
    }
    return Array.from(m.entries()).sort(([a], [b]) => a.localeCompare(b))
  }, [plugins])

  useEffect(() => {
    if (selectedNodeId && !nodes.some((n) => n.id === selectedNodeId)) {
      setSelectedNodeId(null)
      setNodeTestMsg(null)
    }
  }, [nodes, selectedNodeId])

  useEffect(() => {
    const valid = new Set(nodes.map((n) => n.id))
    setGroupRefs((refs) =>
      refs
        .map((ref) => ({ ...ref, node_ids: (ref.node_ids ?? []).filter((id) => valid.has(id)) }))
        .filter((ref) => (ref.node_ids?.length ?? 0) > 0),
    )
  }, [nodes])

  const outdatedGroupRefs = useMemo(() => {
    return groupRefs.filter((ref) => {
      const current = pluginGroups.find((g) => g.id === ref.group_id)?.current_version_id
      return current != null && current !== ref.group_version_id
    })
  }, [groupRefs, pluginGroups])

  const save = async () => {
    const graph = flowToGraph(pipeName, nodes, edges, groupRefs)
    try {
      await api.updatePipeline(pipelineId, pipeName, graph, isDefault)
      setNotice({ kind: 'ok', text: '已保存' })
    } catch (e) {
      setNotice({ kind: 'err', text: String(e) })
    }
  }

  const extractCurrentPipelineAsGroup = async () => {
    const raw = window.prompt('输入要提取的插件组名称', `${pipeName}-group`)
    const groupName = raw?.trim()
    if (!groupName) return
    try {
      const group = await api.createPluginGroup(groupName, `从 pipeline「${pipeName}」提取`, flowToGraph(pipeName, nodes, edges, groupRefs))
      setNotice({ kind: 'ok', text: `已提取为插件组「${group.name}」` })
      setPluginGroups((prev) => [group, ...prev.filter((item) => item.id !== group.id)])
    } catch (e) {
      setNotice({ kind: 'err', text: String(e) })
    }
  }

  const run = async () => {
    try {
      const r = await api.runPipelineById(pipelineId)
      setNotice({ kind: 'ok', text: `已处理 ${r.items_processed} 条` })
    } catch (e) {
      setNotice({ kind: 'err', text: String(e) })
    }
  }

  const probe = async () => {
    try {
      const r = await api.probePipeline(pipelineId)
      setNotice({ kind: r.ok ? 'ok' : 'err', text: r.ok ? '探针通过' : `探针失败: ${r.error ?? ''}` })
    } catch (e) {
      setNotice({ kind: 'err', text: String(e) })
    }
  }

  if (loading) {
    return (
      <div className="flex h-full items-center justify-center text-muted-foreground">加载管道…</div>
    )
  }

  return (
    <div className="flex h-full min-h-0 flex-1 flex-col gap-2 overflow-hidden p-2">
      <div className="flex flex-wrap items-center gap-2 border-b pb-2">
        <Button variant="ghost" size="sm" asChild>
          <Link to="/pipelines" className="gap-1">
            <ArrowLeft className="h-4 w-4" />
            列表
          </Link>
        </Button>
        <Separator orientation="vertical" className="h-6" />
        <div className="flex items-center gap-2">
          <Label htmlFor="pipe-name" className="whitespace-nowrap text-muted-foreground">
            名称
          </Label>
          <Input id="pipe-name" className="h-8 w-40" value={pipeName} onChange={(e) => setPipeName(e.target.value)} />
        </div>
        <div className="flex items-center gap-2">
          <Checkbox
            id="pipe-default"
            checked={isDefault}
            onCheckedChange={(v) => setIsDefault(v === true)}
          />
          <Label htmlFor="pipe-default" className="cursor-pointer font-normal">
            默认管道
          </Label>
        </div>
        <div className="ml-auto flex flex-wrap gap-2">
          <Button size="sm" variant="outline" onClick={extractCurrentPipelineAsGroup}>
            <Boxes className="h-4 w-4" />
            提取为组
          </Button>
          <Button size="sm" variant="destructive" onClick={removeSelectedNode} disabled={!selectedNodeId}>
            <Trash2 className="h-4 w-4" />
            删除节点
          </Button>
          <Button size="sm" variant="outline" onClick={probe}>
            <FlaskConical className="h-4 w-4" />
            探针
          </Button>
          <Button size="sm" variant="secondary" onClick={run}>
            <Play className="h-4 w-4" />
            运行
          </Button>
          <Button size="sm" onClick={save}>
            <Save className="h-4 w-4" />
            保存
          </Button>
        </div>
      </div>
      {notice ? (
        <div
          className={cn(
            'rounded-md px-3 py-2 text-sm',
            notice.kind === 'ok' ? 'bg-secondary text-secondary-foreground' : 'bg-destructive/10 text-destructive',
          )}
        >
          {notice.text}
        </div>
      ) : null}
      {outdatedGroupRefs.length > 0 ? (
        <div className="rounded-md border border-amber-500/40 bg-amber-500/10 px-3 py-2 text-sm text-amber-800 dark:text-amber-200">
          检测到 {outdatedGroupRefs.length} 个插件组引用不是最新版本，保存前可考虑重新拖入最新插件组或手动更新节点。
        </div>
      ) : null}

      {/* 横向 flex 子项默认 min-height:auto，会按内容撑高侧栏导致无法内部滚动；须 min-h-0 + self-stretch */}
      <div className="flex min-h-0 min-w-0 flex-1 items-stretch gap-2">
        <aside className="flex min-h-0 w-56 shrink-0 grow-0 flex-col self-stretch overflow-hidden rounded-lg border bg-card">
          <div className="shrink-0 border-b px-3 py-2 text-sm font-medium">插件库</div>
          <p className="shrink-0 px-3 py-1 text-xs text-muted-foreground">拖到画布添加节点；节点可在画布上拖动；连线连接输出→输入。</p>
          <div className="min-h-0 flex-1 overflow-y-auto overflow-x-hidden overscroll-y-contain px-2 [max-height:min(100%,calc(100vh-12rem))]">
            <div className="flex flex-col gap-3 pb-2 pr-2 pt-1">
              {pluginGroups.length > 0 ? (
                <div>
                  <div className="mb-1 text-xs font-semibold uppercase text-muted-foreground">插件组</div>
                  <div className="flex flex-col gap-1">
                    {pluginGroups.map((group) => (
                      <div
                        key={group.id}
                        draggable
                        onDragStart={(e) => {
                          e.dataTransfer.setData(DND_GROUP_MIME, JSON.stringify({ group_id: group.id } satisfies GroupPalettePayload))
                          e.dataTransfer.effectAllowed = 'copy'
                        }}
                        className="cursor-grab rounded-md border bg-secondary/20 px-2 py-1.5 text-sm shadow-sm active:cursor-grabbing hover:bg-accent/50"
                      >
                        <div className="font-medium">{group.name}</div>
                        <div className="text-[10px] text-muted-foreground">
                          {group.current_version || 'v1'} · {group.graph.nodes.length} 个节点
                          {group.outdated_ref_count > 0 ? ` · ${group.outdated_ref_count} 条引用待更新` : ''}
                        </div>
                      </div>
                    ))}
                  </div>
                </div>
              ) : null}
              {grouped.map(([type, list]) => (
                <div key={type}>
                  <div className="mb-1 text-xs font-semibold uppercase text-muted-foreground">{type}</div>
                  <div className="flex flex-col gap-1">
                    {list.map((p) => (
                      <div
                        key={p.name}
                        draggable
                        onDragStart={(e) => {
                          e.dataTransfer.setData(
                            DND_MIME,
                            JSON.stringify({ plugin_id: p.name, plugin_type: p.type } satisfies PalettePayload),
                          )
                          e.dataTransfer.effectAllowed = 'copy'
                        }}
                        className="cursor-grab rounded-md border bg-background px-2 py-1.5 text-sm shadow-sm active:cursor-grabbing hover:bg-accent/50"
                      >
                        <div className="font-medium">{p.name}</div>
                        <div className="text-[10px] text-muted-foreground">{p.type}</div>
                      </div>
                    ))}
                  </div>
                </div>
              ))}
            </div>
          </div>
        </aside>

        <div className="relative min-h-[min(420px,50vh)] min-w-0 flex-1 self-stretch rounded-lg border bg-background">
          <div className="absolute inset-0 min-h-0">
            <FlowCanvas
              nodes={nodes}
              edges={edges}
              onNodesChange={onNodesChange}
              onEdgesChange={onEdgesChange}
              onConnect={onConnect}
              isValidConnection={isValidConnection}
              onNodeClick={(_, n) => setSelectedNodeId(n.id)}
              onPaneClick={() => setSelectedNodeId(null)}
              setNodes={setNodes}
              setEdges={setEdges}
              setGroupRefs={setGroupRefs}
              groups={pluginGroups}
            />
          </div>
        </div>
      </div>

      <Sheet
        open={selectedNodeId != null}
        onOpenChange={(o) => {
          if (!o) {
            setSelectedNodeId(null)
            setNodeTestMsg(null)
          }
        }}
      >
        <SheetContent side="right" className="flex w-full flex-col sm:max-w-md">
          <SheetHeader>
            <SheetTitle>节点配置</SheetTitle>
            <p className="text-sm text-muted-foreground">
              {pluginId ? (
                <>
                  插件 <span className="font-mono">{pluginId}</span>
                  <Button variant="link" className="ml-1 h-auto p-0 text-xs" size="sm" asChild>
                    <Link to={`/plugins/${encodeURIComponent(pluginId)}`} className="gap-1">
                      <ExternalLink className="h-3 w-3" />
                      完整配置页
                    </Link>
                  </Button>
                </>
              ) : (
                '未选择'
              )}
            </p>
            {pluginId ? (
              <div className="mt-2 flex flex-wrap gap-2">
                <Button
                  type="button"
                  size="sm"
                  variant="secondary"
                  disabled={nodeTestLoading}
                  onClick={testNodePlugin}
                  className="gap-1"
                >
                  <Zap className="h-3 w-3" />
                  {nodeTestLoading ? '测试中…' : '测试配置'}
                </Button>
                <Button type="button" size="sm" variant="destructive" onClick={removeSelectedNode}>
                  <Trash2 className="h-3 w-3" />
                  删除节点
                </Button>
              </div>
            ) : null}
            {nodeTestMsg ? (
              <p
                className={cn(
                  'mt-2 text-xs',
                  nodeTestMsg === '测试通过' ? 'text-emerald-600 dark:text-emerald-400' : 'text-destructive',
                )}
              >
                {nodeTestMsg}
              </p>
            ) : null}
          </SheetHeader>
          <div className="mt-4 flex-1 overflow-y-auto pr-2">
            {pluginId ? (
              <PluginConfigPanel pluginId={pluginId} value={nodeConfig} onChange={updateSelectedConfig} />
            ) : null}
          </div>
        </SheetContent>
      </Sheet>
    </div>
  )
}

export default function PipelineEditorPage() {
  const { id } = useParams()
  const pipelineId = Number(id)

  if (!Number.isFinite(pipelineId) || pipelineId <= 0) {
    return (
      <div className="space-y-4 p-6">
        <p className="text-muted-foreground">无效的管道 ID</p>
        <Button asChild>
          <Link to="/pipelines">返回列表</Link>
        </Button>
      </div>
    )
  }

  return (
    <ReactFlowProvider>
      <div className="flex h-full min-h-0 flex-1 flex-col">
        <PipelineEditorInner pipelineId={pipelineId} />
      </div>
    </ReactFlowProvider>
  )
}
