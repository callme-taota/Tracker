import { Fragment, useEffect, useState } from 'react'
import { api, type Item } from '@/api'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'

export default function ItemsPage() {
  const [items, setItems] = useState<Item[]>([])
  const [total, setTotal] = useState(0)
  const [loading, setLoading] = useState(true)
  const [page, setPage] = useState(1)
  const [expanded, setExpanded] = useState<number | null>(null)
  const pageSize = 20

  const load = async (p = 1) => {
    setLoading(true)
    try {
      const r = await api.getItems(pageSize, (p - 1) * pageSize)
      setItems(Array.isArray(r?.items) ? r.items : [])
      setTotal(r?.total ?? 0)
    } catch {
      setItems([])
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
        <CardTitle>抓取条目</CardTitle>
      </CardHeader>
      <CardContent>
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead className="w-14">ID</TableHead>
              <TableHead>标题</TableHead>
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
              items.map((r) => (
                <Fragment key={r.id}>
                  <TableRow>
                    <TableCell>{r.id}</TableCell>
                    <TableCell className="max-w-xs truncate">
                      {r.url ? (
                        <a href={r.url} target="_blank" rel="noopener noreferrer" className="text-primary underline">
                          {r.title || '-'}
                        </a>
                      ) : (
                        r.title || '-'
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
                        <pre className="max-h-48 overflow-auto whitespace-pre-wrap rounded-md border bg-muted/40 p-3 text-xs">
                          {r.content || r.summary || '-'}
                        </pre>
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
