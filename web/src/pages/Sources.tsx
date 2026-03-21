import { useEffect, useState } from 'react'
import { Trash2, Plus } from 'lucide-react'
import { api, type Source } from '@/api'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { Dialog, DialogContent, DialogHeader, DialogTitle } from '@/components/ui/dialog'

export default function SourcesPage() {
  const [list, setList] = useState<Source[]>([])
  const [loading, setLoading] = useState(true)
  const [open, setOpen] = useState(false)
  const [url, setUrl] = useState('')
  const [type, setType] = useState('rss')
  const [config, setConfig] = useState('{}')
  const [notice, setNotice] = useState<string | null>(null)

  const load = async () => {
    setLoading(true)
    setNotice(null)
    try {
      const data = await api.getSources()
      setList(Array.isArray(data) ? data : [])
    } catch (e) {
      setNotice(String(e))
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    load()
  }, [])

  const add = async () => {
    if (!url.trim()) {
      setNotice('请填写 URL')
      return
    }
    try {
      await api.addSource(url.trim(), type || 'rss', config || '{}')
      setOpen(false)
      setUrl('')
      setType('rss')
      setConfig('{}')
      load()
    } catch (e) {
      setNotice(String(e))
    }
  }

  const remove = async (id: number) => {
    try {
      await api.deleteSource(id)
      load()
    } catch (e) {
      setNotice(String(e))
    }
  }

  return (
    <Card>
      <CardHeader className="flex flex-row items-center justify-between space-y-0">
        <CardTitle>数据源</CardTitle>
        <Button size="sm" onClick={() => setOpen(true)}>
          <Plus className="h-4 w-4" />
          添加
        </Button>
      </CardHeader>
      <CardContent>
        {notice ? <p className="mb-4 text-sm text-destructive">{notice}</p> : null}
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead className="w-16">ID</TableHead>
              <TableHead>URL</TableHead>
              <TableHead className="w-24">类型</TableHead>
              <TableHead className="w-24 text-right">操作</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {loading ? (
              <TableRow>
                <TableCell colSpan={4} className="text-center text-muted-foreground">
                  加载中…
                </TableCell>
              </TableRow>
            ) : list.length === 0 ? (
              <TableRow>
                <TableCell colSpan={4} className="text-center text-muted-foreground">
                  暂无数据
                </TableCell>
              </TableRow>
            ) : (
              list.map((r) => (
                <TableRow key={r.id}>
                  <TableCell>{r.id}</TableCell>
                  <TableCell className="max-w-md truncate font-mono text-xs">{r.url}</TableCell>
                  <TableCell>{r.type}</TableCell>
                  <TableCell className="text-right">
                    <Button variant="ghost" size="icon" onClick={() => remove(r.id)} aria-label="删除">
                      <Trash2 className="h-4 w-4 text-destructive" />
                    </Button>
                  </TableCell>
                </TableRow>
              ))
            )}
          </TableBody>
        </Table>
      </CardContent>

      <Dialog open={open} onOpenChange={setOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>添加数据源</DialogTitle>
          </DialogHeader>
          <div className="grid gap-4 py-2">
            <div className="space-y-2">
              <Label htmlFor="src-url">URL</Label>
              <Input
                id="src-url"
                placeholder="https://example.com/feed.xml"
                value={url}
                onChange={(e) => setUrl(e.target.value)}
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="src-type">类型</Label>
              <Input id="src-type" placeholder="rss" value={type} onChange={(e) => setType(e.target.value)} />
            </div>
            <div className="space-y-2">
              <Label htmlFor="src-cfg">Config (JSON)</Label>
              <Input id="src-cfg" placeholder="{}" value={config} onChange={(e) => setConfig(e.target.value)} />
            </div>
            <div className="flex justify-end gap-2">
              <Button variant="secondary" onClick={() => setOpen(false)}>
                取消
              </Button>
              <Button onClick={add}>添加</Button>
            </div>
          </div>
        </DialogContent>
      </Dialog>
    </Card>
  )
}
