import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { Boxes, Info, Settings2, ShieldCheck } from 'lucide-react'
import { api, type Plugin, type PluginQualityRow } from '@/api'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from '@/components/ui/tooltip'

function QualityIssuesInfo({ row }: { row: PluginQualityRow }) {
  const issues = Array.isArray(row.issues) ? row.issues : []
  if (issues.length === 0) {
    return null
  }

  return (
    <Tooltip>
      <TooltipTrigger asChild>
        <button
          type="button"
          className="inline-flex h-4 w-4 items-center justify-center rounded-full text-muted-foreground transition-colors hover:text-foreground"
          aria-label={`查看 ${row.name} 的治理问题`}
        >
          <Info className="h-3.5 w-3.5" />
        </button>
      </TooltipTrigger>
      <TooltipContent className="space-y-2">
        <div className="font-medium">{row.name} 的治理问题</div>
        {issues.map((issue) => (
          <div key={`${issue.code}-${issue.message}`} className="space-y-0.5">
            <div className="font-mono text-[11px] text-muted-foreground">{issue.code}</div>
            <div>{issue.message}</div>
          </div>
        ))}
      </TooltipContent>
    </Tooltip>
  )
}

export default function PluginsPage() {
  const [list, setList] = useState<Plugin[]>([])
  const [quality, setQuality] = useState<PluginQualityRow[]>([])
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    setLoading(true)
    api
      .getPlugins()
      .then((data) => setList(Array.isArray(data) ? data : []))
      .catch(() => setList([]))
      .finally(() => setLoading(false))
    api
      .getPluginQualityReport()
      .then((data) => setQuality(Array.isArray(data) ? data : []))
      .catch(() => setQuality([]))
  }, [])

  return (
    <div className="space-y-6">
      <Card>
        <CardHeader>
          <CardTitle>插件平台</CardTitle>
          <p className="text-sm text-muted-foreground">
            内置插件配置、工作区插件代码编辑/构建/运行、导入导出，以及插件组与质量治理入口。
          </p>
        </CardHeader>
        <CardContent className="flex flex-wrap gap-2">
          <Button asChild>
            <Link to="/plugin-packages" className="gap-1">
              <Boxes className="h-4 w-4" />
              插件工作区
            </Link>
          </Button>
          <Button variant="secondary" asChild>
            <Link to="/plugin-groups" className="gap-1">
              <Boxes className="h-4 w-4" />
              插件组
            </Link>
          </Button>
        </CardContent>
      </Card>

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

      <TooltipProvider>
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2 text-lg">
              <ShieldCheck className="h-5 w-5" />
              插件质量治理报告
            </CardTitle>
          </CardHeader>
          <CardContent>
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>插件</TableHead>
                  <TableHead>分数</TableHead>
                  <TableHead>分类</TableHead>
                  <TableHead>问题数</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {quality.map((row) => (
                  <TableRow key={row.name}>
                    <TableCell>
                      <div className="font-medium">{row.name}</div>
                      <div className="text-xs text-muted-foreground">
                        {row.type} / {row.runtime}
                      </div>
                    </TableCell>
                    <TableCell>{row.score}</TableCell>
                    <TableCell>
                      <Badge variant={row.category === 'governance-ready' ? 'secondary' : 'outline'}>{row.category}</Badge>
                    </TableCell>
                    <TableCell>
                      <div className="flex items-center gap-1">
                        <span>{row.issues?.length ?? 0}</span>
                        <QualityIssuesInfo row={row} />
                      </div>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </CardContent>
        </Card>
      </TooltipProvider>
    </div>
  )
}
