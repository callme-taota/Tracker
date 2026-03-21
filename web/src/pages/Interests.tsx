import { useEffect, useState } from 'react'
import { Trash2, Plus } from 'lucide-react'
import { api, type Interest } from '@/api'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { Dialog, DialogContent, DialogHeader, DialogTitle } from '@/components/ui/dialog'

export default function InterestsPage() {
  const [list, setList] = useState<Interest[]>([])
  const [loading, setLoading] = useState(true)
  const [open, setOpen] = useState(false)
  const [name, setName] = useState('')
  const [keywords, setKeywords] = useState('')
  const [notice, setNotice] = useState<string | null>(null)

  const load = async () => {
    setLoading(true)
    setNotice(null)
    try {
      const data = await api.getInterests()
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
    if (!name.trim()) {
      setNotice('请填写名称')
      return
    }
    try {
      await api.addInterest(name.trim(), keywords || '')
      setOpen(false)
      setName('')
      setKeywords('')
      load()
    } catch (e) {
      setNotice(String(e))
    }
  }

  const remove = async (id: number) => {
    try {
      await api.deleteInterest(id)
      load()
    } catch (e) {
      setNotice(String(e))
    }
  }

  return (
    <Card>
      <CardHeader className="flex flex-row items-center justify-between space-y-0">
        <CardTitle>兴趣</CardTitle>
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
              <TableHead>名称</TableHead>
              <TableHead>关键词</TableHead>
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
            ) : (
              list.map((r) => (
                <TableRow key={r.id}>
                  <TableCell>{r.id}</TableCell>
                  <TableCell>{r.name}</TableCell>
                  <TableCell className="max-w-lg truncate text-muted-foreground">{r.keywords}</TableCell>
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
            <DialogTitle>添加兴趣</DialogTitle>
          </DialogHeader>
          <div className="grid gap-4 py-2">
            <div className="space-y-2">
              <Label htmlFor="int-name">名称</Label>
              <Input id="int-name" placeholder="tech" value={name} onChange={(e) => setName(e.target.value)} />
            </div>
            <div className="space-y-2">
              <Label htmlFor="int-kw">关键词（逗号分隔）</Label>
              <Input
                id="int-kw"
                placeholder="AI, open source"
                value={keywords}
                onChange={(e) => setKeywords(e.target.value)}
              />
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
