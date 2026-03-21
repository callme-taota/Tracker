import { useEffect, useState } from 'react'
import { RefreshCw, Play } from 'lucide-react'
import { api, type Stats, type PipelineStatus } from '@/api'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'

export default function Dashboard() {
  const [stats, setStats] = useState<Stats | null>(null)
  const [pipeline, setPipeline] = useState<PipelineStatus | null>(null)
  const [loading, setLoading] = useState(true)
  const [running, setRunning] = useState(false)
  const [notice, setNotice] = useState<string | null>(null)

  const load = async () => {
    setLoading(true)
    setNotice(null)
    try {
      const [s, p] = await Promise.all([api.getStats(), api.getPipelineStatus()])
      setStats(s ?? { sources: 0, items: 0, interests: 0, summaries: 0 })
      setPipeline(p ?? null)
    } catch (e) {
      setNotice(String(e))
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    load()
  }, [])

  const runPipeline = async () => {
    setRunning(true)
    try {
      const r = await api.runPipeline()
      setNotice(`已处理 ${r?.items_processed ?? 0} 条${r?.saved ? '，已写入存储' : ''}`)
      load()
    } catch (e) {
      setNotice(String(e))
    } finally {
      setRunning(false)
    }
  }

  const maxStat = Math.max(stats?.sources ?? 0, stats?.items ?? 0, stats?.interests ?? 0, stats?.summaries ?? 0, 1)
  const chartItems = [
    { label: '数据源', value: stats?.sources ?? 0 },
    { label: '抓取条目', value: stats?.items ?? 0 },
    { label: '兴趣', value: stats?.interests ?? 0 },
    { label: '摘要', value: stats?.summaries ?? 0 },
  ]

  if (loading && !stats) {
    return <div className="py-16 text-center text-muted-foreground">加载中…</div>
  }

  return (
    <div className="space-y-6">
      {notice ? (
        <div className="rounded-md border bg-muted/50 px-4 py-2 text-sm text-muted-foreground">{notice}</div>
      ) : null}
      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
        {chartItems.map(({ label, value }) => (
          <Card key={label}>
            <CardHeader className="pb-2">
              <CardDescription>{label}</CardDescription>
              <CardTitle className="text-3xl tabular-nums">{value}</CardTitle>
            </CardHeader>
          </Card>
        ))}
      </div>

      <div className="grid gap-4 lg:grid-cols-3">
        <Card className="lg:col-span-2">
          <CardHeader className="flex flex-row items-center justify-between space-y-0">
            <CardTitle>存储概览</CardTitle>
            <Button variant="ghost" size="icon" onClick={load}>
              <RefreshCw className="h-4 w-4" />
            </Button>
          </CardHeader>
          <CardContent className="space-y-3">
            {chartItems.map(({ label, value }) => (
              <div key={label}>
                <div className="mb-1 flex justify-between text-sm">
                  <span>{label}</span>
                  <span className="tabular-nums text-muted-foreground">{value}</span>
                </div>
                <div className="h-2 overflow-hidden rounded-full bg-muted">
                  <div
                    className="h-full rounded-full bg-primary transition-all"
                    style={{ width: `${maxStat ? Math.round((value / maxStat) * 100) : 0}%` }}
                  />
                </div>
              </div>
            ))}
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0">
            <CardTitle>管道</CardTitle>
            <Button size="sm" onClick={runPipeline} disabled={running}>
              <Play className="h-4 w-4" />
              {running ? '运行中' : '运行'}
            </Button>
          </CardHeader>
          <CardContent className="text-sm">
            {pipeline && 'error' in pipeline && pipeline.error ? (
              <p className="text-destructive">{pipeline.error}</p>
            ) : pipeline ? (
              <div className="space-y-2">
                <p className="font-medium">{pipeline.name}</p>
                <p className="text-muted-foreground">状态: {pipeline.status}</p>
                <ul className="list-inside list-disc text-muted-foreground">
                  {(pipeline.stages ?? []).map((s, i) => (
                    <li key={i}>
                      {s.name} ({s.plugin_id})
                    </li>
                  ))}
                </ul>
              </div>
            ) : (
              <p className="text-muted-foreground">未加载</p>
            )}
          </CardContent>
        </Card>
      </div>
    </div>
  )
}
