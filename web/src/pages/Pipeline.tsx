import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { Network, Play } from 'lucide-react'
import { api, type PipelineStatus } from '@/api'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Separator } from '@/components/ui/separator'

export default function PipelinePage() {
  const [data, setData] = useState<PipelineStatus | null>(null)
  const [loading, setLoading] = useState(true)
  const [running, setRunning] = useState(false)
  const [notice, setNotice] = useState<string | null>(null)

  const load = async () => {
    setLoading(true)
    setNotice(null)
    try {
      const p = await api.getPipelineStatus()
      setData(p ?? null)
    } catch (e) {
      setNotice(String(e))
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    load()
  }, [])

  const run = async () => {
    setRunning(true)
    try {
      const r = await api.runPipeline()
      setNotice(
        `已处理 ${r?.items_processed ?? 0} 条${r?.saved ? '，已写入存储' : ''}${r?.source ? `（${r.source}）` : ''}`,
      )
      load()
    } catch (e) {
      setNotice(String(e))
    } finally {
      setRunning(false)
    }
  }

  if (loading && !data) {
    return <div className="py-16 text-center text-muted-foreground">加载中…</div>
  }

  const fileStages =
    data && data.source === 'file'
      ? ((data.stages ?? data.linear_stages ?? []) as { name: string; plugin_id: string; type: string }[])
      : []
  const dbSteps =
    data && data.source === 'db' && data.graph?.nodes
      ? data.graph.nodes.map((n) => ({
          title: n.id,
          description: `${n.plugin_id} (${n.plugin_type})`,
        }))
      : []

  return (
    <Card>
      <CardHeader className="flex flex-row flex-wrap items-center justify-between gap-2 space-y-0">
        <div>
          <CardTitle>管道状态</CardTitle>
          <CardDescription>数据库管道可在画布中编辑 DAG</CardDescription>
        </div>
        <div className="flex gap-2">
          {data && data.source === 'db' && (
            <Button variant="secondary" size="sm" asChild>
              <Link to={`/pipelines/${data.id}`} className="gap-2">
                <Network className="h-4 w-4" />
                打开编辑器
              </Link>
            </Button>
          )}
          <Button size="sm" onClick={run} disabled={running}>
            <Play className="h-4 w-4" />
            运行管道
          </Button>
        </div>
      </CardHeader>
      <CardContent className="space-y-4 text-sm">
        {notice ? <p className="text-muted-foreground">{notice}</p> : null}
        {data && 'error' in data && data.error ? (
          <p className="text-destructive">{data.error}</p>
        ) : data?.source === 'db' ? (
          <>
            <p>
              来源: <strong>数据库</strong> · 管道 #{data.id} · {data.name}
              {data.is_default ? ' · 默认' : ''}
            </p>
            <p className="text-muted-foreground">
              当前 DAG：{data.graph?.nodes?.length ?? 0} 节点 / {data.graph?.edges?.length ?? 0} 边
            </p>
            <Separator />
            <ol className="list-decimal space-y-2 pl-5">
              {dbSteps.map((s) => (
                <li key={s.title}>
                  <span className="font-medium">{s.title}</span>
                  <span className="text-muted-foreground"> — {s.description}</span>
                </li>
              ))}
            </ol>
          </>
        ) : (
          <>
            <p>
              来源: <strong>YAML 文件</strong>
              {'config_path' in (data ?? {}) && (data as { config_path?: string }).config_path
                ? ` · ${(data as { config_path?: string }).config_path}`
                : ''}
            </p>
            <p>
              管道名: <strong>{data?.name ?? '-'}</strong> · 状态: {data?.status ?? '-'}
            </p>
            <Separator />
            <ol className="list-decimal space-y-2 pl-5">
              {fileStages.map((s) => (
                <li key={s.name}>
                  <span className="font-medium">{s.name}</span>
                  <span className="text-muted-foreground">
                    {' '}
                    — {s.plugin_id} ({s.type})
                  </span>
                </li>
              ))}
            </ol>
          </>
        )}
      </CardContent>
    </Card>
  )
}
