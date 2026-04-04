import { useEffect, useState } from 'react'
import { api, type PipelineGraphDTO, type PipelineSummary, type PluginGroup, type PluginGroupVersion } from '@/api'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'

const emptyGraph: PipelineGraphDTO = { name: 'group', nodes: [], edges: [] }

export default function PluginGroupsPage() {
  const [list, setList] = useState<PluginGroup[]>([])
  const [pipelines, setPipelines] = useState<PipelineSummary[]>([])
  const [selectedGroupId, setSelectedGroupId] = useState<number | null>(null)
  const [versions, setVersions] = useState<PluginGroupVersion[]>([])
  const [name, setName] = useState('基础清洗组')
  const [description, setDescription] = useState('供多个 pipeline 复用的插件组')
  const [graphJSON, setGraphJSON] = useState(JSON.stringify(emptyGraph, null, 2))
  const [extractPipelineId, setExtractPipelineId] = useState<number | null>(null)
  const [notice, setNotice] = useState<string | null>(null)

  const load = async () => {
    try {
      const [groups, pipelineList] = await Promise.all([api.listPluginGroups(), api.listPipelines()])
      const safeGroups = Array.isArray(groups) ? groups : []
      const safePipelines = Array.isArray(pipelineList) ? pipelineList : []
      setList(safeGroups)
      setPipelines(safePipelines)
      if (!selectedGroupId && safeGroups[0]) setSelectedGroupId(safeGroups[0].id)
    } catch (e) {
      setNotice(String(e))
    }
  }

  useEffect(() => {
    load()
  }, [])

  useEffect(() => {
    if (!selectedGroupId) {
      setVersions([])
      return
    }
    api
      .listPluginGroupVersions(selectedGroupId)
      .then((data) => setVersions(Array.isArray(data) ? data : []))
      .catch((e) => setNotice(String(e)))
  }, [selectedGroupId])

  useEffect(() => {
    const current = list.find((item) => item.id === selectedGroupId)
    if (!current) return
    setName(current.name)
    setDescription(current.description)
    setGraphJSON(JSON.stringify(current.graph ?? emptyGraph, null, 2))
  }, [list, selectedGroupId])

  const create = async () => {
    try {
      const group = await api.createPluginGroup(name, description, JSON.parse(graphJSON) as PipelineGraphDTO)
      setNotice(`插件组已保存，当前版本 ${group.current_version || 'v1'}`)
      await load()
      setSelectedGroupId(group.id)
    } catch (e) {
      setNotice(String(e))
    }
  }

  const extract = async () => {
    if (!extractPipelineId) return
    try {
      const group = await api.extractPluginGroupFromPipeline(extractPipelineId, name, description)
      setNotice(`已从 pipeline #${extractPipelineId} 提取插件组`)
      await load()
      setSelectedGroupId(group.id)
    } catch (e) {
      setNotice(String(e))
    }
  }

  const publishVersion = async () => {
    if (!selectedGroupId) return
    try {
      await api.updatePluginGroup(selectedGroupId, name, description, JSON.parse(graphJSON) as PipelineGraphDTO)
      setNotice('已发布插件组新版本')
      await load()
      const data = await api.listPluginGroupVersions(selectedGroupId)
      setVersions(Array.isArray(data) ? data : [])
    } catch (e) {
      setNotice(String(e))
    }
  }

  const remove = async (id: number) => {
    if (!window.confirm('确认删除该插件组？')) return
    try {
      await api.deletePluginGroup(id)
      setNotice('插件组已删除')
      if (selectedGroupId === id) setSelectedGroupId(null)
      await load()
    } catch (e) {
      setNotice(String(e))
    }
  }

  return (
    <div className="space-y-6">
      <Card>
        <CardHeader>
          <CardTitle>插件组 / 子图模板</CardTitle>
          <p className="text-sm text-muted-foreground">
            插件组用于把一段可复用子图保存为模板，多个 pipeline 可重复插入与复用。
          </p>
        </CardHeader>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle className="text-lg">新建插件组</CardTitle>
        </CardHeader>
        <CardContent className="space-y-3">
          <Input value={name} onChange={(e) => setName(e.target.value)} placeholder="插件组名称" />
          <Input value={description} onChange={(e) => setDescription(e.target.value)} placeholder="说明" />
          <div className="flex flex-wrap gap-2">
            <select
              className="h-10 rounded-md border bg-background px-3 text-sm"
              value={extractPipelineId ?? ''}
              onChange={(e) => setExtractPipelineId(e.target.value ? Number(e.target.value) : null)}
            >
              <option value="">从现有 pipeline 一键提取</option>
              {pipelines.map((p) => (
                <option key={p.id} value={p.id}>
                  #{p.id} {p.name}
                </option>
              ))}
            </select>
            <Button variant="outline" onClick={extract} disabled={!extractPipelineId}>
              提取为组
            </Button>
          </div>
          <textarea
            className="min-h-64 w-full rounded-md border bg-background p-3 font-mono text-xs"
            value={graphJSON}
            onChange={(e) => setGraphJSON(e.target.value)}
          />
          <div className="flex gap-2">
            <Button onClick={create}>创建插件组</Button>
            <Button variant="secondary" onClick={publishVersion} disabled={!selectedGroupId}>
              发布新版本
            </Button>
          </div>
          {notice ? <p className="text-sm text-muted-foreground">{notice}</p> : null}
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle className="text-lg">已保存的插件组</CardTitle>
        </CardHeader>
        <CardContent>
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>名称</TableHead>
                <TableHead>描述</TableHead>
                <TableHead>当前版本</TableHead>
                <TableHead>版本数</TableHead>
                <TableHead>节点数</TableHead>
                <TableHead>引用</TableHead>
                <TableHead className="text-right">操作</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {list.map((row) => (
                <TableRow key={row.id} className={selectedGroupId === row.id ? 'bg-muted/40' : ''}>
                  <TableCell className="font-medium">
                    <button type="button" className="text-left hover:underline" onClick={() => setSelectedGroupId(row.id)}>
                      {row.name}
                    </button>
                  </TableCell>
                  <TableCell className="text-muted-foreground">{row.description}</TableCell>
                  <TableCell>{row.current_version || 'v1'}</TableCell>
                  <TableCell>{row.version_count}</TableCell>
                  <TableCell>{row.graph?.nodes?.length ?? 0}</TableCell>
                  <TableCell>
                    {row.reference_count}
                    {row.outdated_ref_count > 0 ? (
                      <span className="ml-2 text-xs text-amber-600">有 {row.outdated_ref_count} 条引用待更新</span>
                    ) : null}
                  </TableCell>
                  <TableCell className="text-right">
                    <Button variant="outline" size="sm" onClick={() => remove(row.id)}>
                      删除
                    </Button>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle className="text-lg">版本历史</CardTitle>
        </CardHeader>
        <CardContent className="space-y-3">
          {selectedGroupId ? (
            versions.length > 0 ? (
              versions.map((ver) => (
                <div key={ver.id} className="rounded-md border p-3 text-sm">
                  <div className="font-medium">
                    {ver.version} {ver.source_pipeline_id ? `· 来源 pipeline #${ver.source_pipeline_id}` : ''}
                  </div>
                  <div className="text-muted-foreground">{ver.change_note || '常规发布'}</div>
                  <div className="mt-2 text-xs text-muted-foreground">创建时间：{ver.created_at}</div>
                </div>
              ))
            ) : (
              <p className="text-sm text-muted-foreground">该插件组还没有版本历史。</p>
            )
          ) : (
            <p className="text-sm text-muted-foreground">选择一个插件组后查看版本历史。</p>
          )}
        </CardContent>
      </Card>
    </div>
  )
}
