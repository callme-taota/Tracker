import { Fragment, useEffect, useState } from 'react'
import { api, type SummaryRow } from '@/api'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'

function parseKeyPoints(s: string): string[] {
  if (!s) return []
  try {
    const a = JSON.parse(s) as unknown
    return Array.isArray(a) ? (a as string[]) : [s]
  } catch {
    return [s]
  }
}

export default function SummariesPage() {
  const [list, setList] = useState<SummaryRow[]>([])
  const [total, setTotal] = useState(0)
  const [loading, setLoading] = useState(true)
  const [page, setPage] = useState(1)
  const [expanded, setExpanded] = useState<number | null>(null)
  const pageSize = 20

  const load = async (p = 1) => {
    setLoading(true)
    try {
      const r = await api.getSummaries(pageSize, (p - 1) * pageSize)
      setList(Array.isArray(r?.summaries) ? r.summaries : [])
      setTotal(r?.total ?? 0)
    } catch {
      setList([])
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    load(page)
  }, [page])

  const totalPages = Math.max(1, Math.ceil(total / pageSize))

  return (
    <Card>
      <CardHeader>
        <CardTitle>摘要与关键点</CardTitle>
      </CardHeader>
      <CardContent>
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead className="w-14">ID</TableHead>
              <TableHead>原文标题</TableHead>
              <TableHead>摘要</TableHead>
              <TableHead className="w-40">时间</TableHead>
              <TableHead className="w-24">详情</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {loading ? (
              <TableRow>
                <TableCell colSpan={5} className="text-center text-muted-foreground">
                  加载中…
                </TableCell>
              </TableRow>
            ) : (
              list.map((r) => (
                <Fragment key={r.id}>
                  <TableRow>
                    <TableCell>{r.id}</TableCell>
                    <TableCell className="max-w-xs truncate">
                      {r.item_url ? (
                        <a
                          href={r.item_url}
                          target="_blank"
                          rel="noopener noreferrer"
                          className="text-primary underline"
                        >
                          {r.item_title || '-'}
                        </a>
                      ) : (
                        r.item_title || '-'
                      )}
                    </TableCell>
                    <TableCell className="max-w-md truncate text-muted-foreground">{r.summary || '—'}</TableCell>
                    <TableCell className="whitespace-nowrap text-xs text-muted-foreground">
                      {r.created_at ? new Date(r.created_at).toLocaleString('zh-CN') : '-'}
                    </TableCell>
                    <TableCell>
                      <Button variant="ghost" size="sm" onClick={() => setExpanded(expanded === r.id ? null : r.id)}>
                        {expanded === r.id ? '收起' : '展开'}
                      </Button>
                    </TableCell>
                  </TableRow>
                  {expanded === r.id ? (
                    <TableRow>
                      <TableCell colSpan={5}>
                        <p className="mb-2 font-medium">摘要</p>
                        <p className="mb-4 text-sm text-muted-foreground">{r.summary || '-'}</p>
                        <p className="mb-2 font-medium">关键点</p>
                        <ul className="list-inside list-disc text-sm">
                          {parseKeyPoints(r.key_points).map((k, i) => (
                            <li key={i}>{k}</li>
                          ))}
                        </ul>
                      </TableCell>
                    </TableRow>
                  ) : null}
                </Fragment>
              ))
            )}
          </TableBody>
        </Table>
        <div className="mt-4 flex items-center justify-between text-sm text-muted-foreground">
          <span>
            共 {total} 条 · 第 {page}/{totalPages} 页
          </span>
          <div className="flex gap-2">
            <Button variant="outline" size="sm" disabled={page <= 1} onClick={() => setPage((p) => p - 1)}>
              上一页
            </Button>
            <Button variant="outline" size="sm" disabled={page >= totalPages} onClick={() => setPage((p) => p + 1)}>
              下一页
            </Button>
          </div>
        </div>
      </CardContent>
    </Card>
  )
}
