import { useEffect, useState } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { Plus, RefreshCw, Trash2 } from 'lucide-react'
import { api, type PipelineGraphDTO, type PipelineSummary } from '@/api'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Badge } from '@/components/ui/badge'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'

/** 最小可校验 DAG：rss(source) → clean(processor) */
const SAMPLE_GRAPH: PipelineGraphDTO = {
  name: 'default',
  nodes: [
    { id: 'n1', plugin_type: 'source', plugin_id: 'rss', config: {}, position: { x: 80, y: 80 } },
    { id: 'n2', plugin_type: 'processor', plugin_id: 'clean', config: {}, position: { x: 320, y: 80 } },
  ],
  edges: [{ id: 'e1', source: 'n1', target: 'n2' }],
}

export default function PipelinesPage() {
  const nav = useNavigate()
  const [list, setList] = useState<PipelineSummary[]>([])
  const [loading, setLoading] = useState(true)
  const [storageOn, setStorageOn] = useState<boolean | null>(null)
  const [creating, setCreating] = useState(false)
  const [deletingId, setDeletingId] = useState<number | null>(null)
  const [jobId, setJobId] = useState('')
  const [jobStatus, setJobStatus] = useState<string>('')
  const [notice, setNotice] = useState<string | null>(null)

  const load = async () => {
    setLoading(true)
    setNotice(null)
    try {
      const [l, st] = await Promise.all([api.listPipelines(), api.getStats()])
      setList(l ?? [])
      const se = st.storage_enabled
      setStorageOn(se === true ? true : se === false ? false : null)
    } catch (e) {
      setNotice(String(e))
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    load()
  }, [])

  const runAsync = async (id: number) => {
    try {
      const r = await api.enqueuePipelineRun(id)
      setNotice(`已入队 job #${r.job_id}`)
    } catch (e) {
      setNotice(String(e))
    }
  }

  const probe = async (id: number) => {
    try {
      const r = await api.probePipeline(id)
      setNotice(r.ok ? '探针通过' : `探针失败: ${r.error ?? ''}`)
    } catch (e) {
      setNotice(String(e))
    }
  }

  const fetchJob = async () => {
    const id = Number(jobId)
    if (!id) return
    try {
      const j = await api.getJob(id)
      setJobStatus(JSON.stringify(j, null, 2))
    } catch (e) {
      setNotice(String(e))
    }
  }

  const createSamplePipeline = async () => {
    setCreating(true)
    setNotice(null)
    try {
      const r = await api.createPipeline('默认管道', SAMPLE_GRAPH, true)
      await load()
      nav(`/pipelines/${r.id}`)
    } catch (e) {
      const msg = String(e)
      if (/storage not configured|503|service unavailable/i.test(msg)) {
        setNotice('未连接存储：请配置 db_path 或 TRACKER_DB_PATH 或 tracker serve --db tracker.db 后重启。')
      } else {
        setNotice(msg)
      }
    } finally {
      setCreating(false)
    }
  }

  const removePipeline = async (id: number, name: string, isDefault: boolean) => {
    const tip = isDefault
      ? `将删除默认管道「${name}」，确认继续？`
      : `确认删除管道「${name}」？`
    if (!window.confirm(tip)) return
    setDeletingId(id)
    setNotice(null)
    try {
      await api.deletePipeline(id)
      await load()
      setNotice(`已删除管道 #${id}`)
    } catch (e) {
      setNotice(String(e))
    } finally {
      setDeletingId(null)
    }
  }

  return (
    <div className="space-y-6">
      <Card>
        <CardHeader className="flex flex-row flex-wrap items-center justify-between gap-2 space-y-0">
          <div>
            <CardTitle>管道</CardTitle>
            <CardDescription>打开「编辑」进入可拖拽画布；异步任务由服务端 worker 消费。</CardDescription>
          </div>
          <div className="flex flex-wrap gap-2">
            {storageOn !== false && !loading ? (
              <Button size="sm" onClick={createSamplePipeline} disabled={creating}>
                <Plus className="h-4 w-4" />
                {creating ? '创建中…' : '新建管道(示例)'}
              </Button>
            ) : null}
            <Button variant="outline" size="sm" onClick={load} disabled={loading}>
              <RefreshCw className={`h-4 w-4 ${loading ? 'animate-spin' : ''}`} />
              刷新
            </Button>
          </div>
        </CardHeader>
        <CardContent>
          {notice ? <p className="mb-4 text-sm text-destructive">{notice}</p> : null}
          {storageOn === false && !loading ? (
            <div className="mb-4 rounded-lg border border-amber-500/40 bg-amber-500/10 px-4 py-3 text-sm text-foreground">
              <p className="font-medium text-amber-900 dark:text-amber-100">当前未启用 SQLite 存储</p>
              <p className="mt-2 text-muted-foreground">
                管道列表来自表 <code className="rounded bg-muted px-1">pipeline_definitions</code>，需服务端打开数据库文件。请任选其一后
                <strong>重启</strong> <code className="rounded bg-muted px-1">tracker serve</code>：
              </p>
              <ul className="mt-2 list-inside list-disc space-y-1 text-muted-foreground">
                <li>
                  在 <code className="rounded bg-muted px-1">tracker.yaml</code> / <code className="rounded bg-muted px-1">config.yaml</code>{' '}
                  的 <code className="rounded bg-muted px-1">app</code> 下设置{' '}
                  <code className="rounded bg-muted px-1">db_path: tracker.db</code>（勿留空覆盖默认值）
                </li>
                <li>
                  或启动：<code className="rounded bg-muted px-1">tracker serve --db tracker.db</code>
                </li>
                <li>
                  或环境变量：<code className="rounded bg-muted px-1">export TRACKER_DB_PATH=tracker.db</code>
                </li>
              </ul>
              <p className="mt-2 text-xs text-muted-foreground">参考仓库 <code className="rounded bg-muted px-1">configs/tracker.example.yaml</code>。</p>
            </div>
          ) : null}
          {storageOn === true && list.length === 0 && !loading ? (
            <p className="mb-4 text-sm text-muted-foreground">
              存储已连接，但还没有管道。可点击上方「新建示例管道」生成一条 <strong>rss → clean</strong> 的 DAG，再进编辑器调整。
            </p>
          ) : null}
          {storageOn === null && list.length === 0 && !loading ? (
            <p className="mb-4 text-sm text-muted-foreground">
              当前后端未报告存储状态。若「新建示例管道」失败，请按上方「未启用存储」说明配置 <code className="rounded bg-muted px-1">db_path</code> 并重启服务。
            </p>
          ) : null}
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead className="w-16">ID</TableHead>
                <TableHead>名称</TableHead>
                <TableHead className="w-24">默认</TableHead>
                <TableHead>更新</TableHead>
                <TableHead className="text-right">操作</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {list.length === 0 && !loading ? (
                <TableRow>
                  <TableCell colSpan={5} className="text-center text-muted-foreground">
                    {storageOn === false
                      ? '未连接存储 — 见上方说明'
                      : storageOn === true
                        ? '暂无管道 — 可使用「新建示例管道」'
                        : '暂无管道 — 可尝试「新建示例管道」或检查存储配置'}
                  </TableCell>
                </TableRow>
              ) : (
                list.map((row) => (
                  <TableRow key={row.id}>
                    <TableCell>{row.id}</TableCell>
                    <TableCell className="font-medium">{row.name}</TableCell>
                    <TableCell>
                      {row.is_default ? (
                        <Badge>默认</Badge>
                      ) : (
                        <span className="text-muted-foreground">—</span>
                      )}
                    </TableCell>
                    <TableCell className="text-muted-foreground">{row.updated_at}</TableCell>
                    <TableCell className="text-right">
                      <div className="flex justify-end gap-2">
                        <Button size="sm" asChild>
                          <Link to={`/pipelines/${row.id}`}>编辑</Link>
                        </Button>
                        <Button size="sm" variant="secondary" onClick={() => runAsync(row.id)}>
                          异步运行
                        </Button>
                        <Button size="sm" variant="outline" onClick={() => probe(row.id)}>
                          探针
                        </Button>
                        <Button
                          size="sm"
                          variant="destructive"
                          disabled={deletingId === row.id}
                          onClick={() => removePipeline(row.id, row.name, row.is_default)}
                        >
                          <Trash2 className="h-4 w-4" />
                          {deletingId === row.id ? '删除中…' : '删除'}
                        </Button>
                      </div>
                    </TableCell>
                  </TableRow>
                ))
              )}
            </TableBody>
          </Table>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle className="text-lg">任务查询</CardTitle>
          <CardDescription>根据 job id 查看队列任务状态</CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="flex gap-2">
            <Input placeholder="Job ID" value={jobId} onChange={(e) => setJobId(e.target.value)} className="max-w-xs" />
            <Button variant="secondary" onClick={fetchJob}>
              查询
            </Button>
          </div>
          {jobStatus ? (
            <pre className="max-h-64 overflow-auto rounded-md border bg-muted/50 p-4 text-xs">{jobStatus}</pre>
          ) : null}
        </CardContent>
      </Card>
    </div>
  )
}
