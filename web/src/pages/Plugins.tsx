import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { Settings2 } from 'lucide-react'
import { api, type Plugin } from '@/api'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'

export default function PluginsPage() {
  const [list, setList] = useState<Plugin[]>([])
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    setLoading(true)
    api
      .getPlugins()
      .then((data) => setList(Array.isArray(data) ? data : []))
      .catch(() => setList([]))
      .finally(() => setLoading(false))
  }, [])

  return (
    <Card>
      <CardHeader>
        <CardTitle>插件列表</CardTitle>
        <p className="text-sm text-muted-foreground">
          点击「配置」进入通用配置页（支持 manifest 表单或 JSON）；预设保存在本浏览器，拖入管道节点时会合并。
        </p>
      </CardHeader>
      <CardContent>
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>名称</TableHead>
              <TableHead className="w-28">版本</TableHead>
              <TableHead className="w-32">类型</TableHead>
              <TableHead className="w-28 text-right">操作</TableHead>
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
              list.map((p) => (
                <TableRow key={p.name}>
                  <TableCell className="font-medium">{p.name}</TableCell>
                  <TableCell className="text-muted-foreground">{p.version}</TableCell>
                  <TableCell>
                    <Badge variant="secondary">{p.type}</Badge>
                  </TableCell>
                  <TableCell className="text-right">
                    <Button size="sm" variant="secondary" asChild>
                      <Link to={`/plugins/${encodeURIComponent(p.name)}`} className="gap-1">
                        <Settings2 className="h-4 w-4" />
                        配置
                      </Link>
                    </Button>
                  </TableCell>
                </TableRow>
              ))
            )}
          </TableBody>
        </Table>
      </CardContent>
    </Card>
  )
}
