import { useEffect, useMemo, useState } from 'react'
import { Link } from 'react-router-dom'
import { Code2, Download, Hammer } from 'lucide-react'
import { api, type PluginImportRequest, type PluginPackage } from '@/api'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Badge } from '@/components/ui/badge'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'

const defaultBundle = JSON.stringify(
  {
    plugin_id: 'example_ts_plugin',
    name: 'Example TS Plugin',
    runtime: 'ts',
    source_kind: 'imported',
    version: '0.1.0',
    manifest_json: JSON.stringify(
      {
        id: 'example_ts_plugin',
        version: '0.1.0',
        kind: 'processor',
        input_formats: ['tracker.item.v1'],
        output_formats: ['tracker.item.v1'],
      },
      null,
      2,
    ),
    entry_file: 'main.ts',
    files: {
      'main.ts': 'console.log(JSON.stringify({ listen_addr: "127.0.0.1:0" }))\n',
    },
  },
  null,
  2,
)

export default function PluginPackagesPage() {
  const [list, setList] = useState<PluginPackage[]>([])
  const [notice, setNotice] = useState<string | null>(null)
  const [pluginId, setPluginId] = useState('workspace_ts_plugin')
  const [name, setName] = useState('Workspace TS Plugin')
  const [runtime, setRuntime] = useState('ts')
  const [bundle, setBundle] = useState(defaultBundle)

  const load = async () => {
    try {
      const data = await api.listPluginPackages()
      setList(Array.isArray(data) ? data : [])
    } catch (e) {
      setNotice(String(e))
    }
  }

  useEffect(() => {
    load()
  }, [])

  const createScaffold = async () => {
    const body: PluginImportRequest = {
      plugin_id: pluginId,
      name,
      runtime,
      source_kind: 'workspace',
      manifest_json: JSON.stringify(
        {
          id: pluginId,
          version: '0.1.0',
          kind: 'processor',
          input_formats: ['tracker.item.v1'],
          output_formats: ['tracker.item.v1'],
        },
        null,
        2,
      ),
      entry_file: runtime === 'rust' ? 'src/main.rs' : runtime === 'go' ? 'main.go' : 'main.ts',
      files: {},
    }
    try {
      const res = await api.createPluginPackage(body)
      setNotice(`已创建插件工作区：${res.package.plugin_id}`)
      await load()
    } catch (e) {
      setNotice(String(e))
    }
  }

  const importBundle = async () => {
    try {
      const parsed = JSON.parse(bundle) as PluginImportRequest
      const res = await api.importPluginPackage(parsed)
      setNotice(`导入完成：${res.package.plugin_id}，审查状态：${res.review.status}`)
      await load()
    } catch (e) {
      setNotice(String(e))
    }
  }

  const stats = useMemo(
    () => ({
      total: list.length,
      approved: list.filter((p) => p.review_status === 'approved').length,
      enabled: list.filter((p) => p.enabled).length,
    }),
    [list],
  )

  return (
    <div className="space-y-6">
      <Card>
        <CardHeader>
          <CardTitle>插件工作区</CardTitle>
          <p className="text-sm text-muted-foreground">
            管理平台托管插件、导入第三方插件包，并进入高级工作台编辑代码、审查风险、构建与运行。
          </p>
        </CardHeader>
        <CardContent className="flex flex-wrap gap-4 text-sm">
          <Badge variant="secondary">总数 {stats.total}</Badge>
          <Badge variant="outline">已审查通过 {stats.approved}</Badge>
          <Badge variant="outline">已启用 {stats.enabled}</Badge>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle className="text-lg">新建工作区插件</CardTitle>
        </CardHeader>
        <CardContent className="grid gap-3 md:grid-cols-3">
          <Input value={pluginId} onChange={(e) => setPluginId(e.target.value)} placeholder="plugin_id" />
          <Input value={name} onChange={(e) => setName(e.target.value)} placeholder="显示名称" />
          <Input value={runtime} onChange={(e) => setRuntime(e.target.value)} placeholder="runtime: go / ts / rust" />
          <div className="md:col-span-3">
            <Button onClick={createScaffold} className="gap-2">
              <Hammer className="h-4 w-4" />
              创建脚手架
            </Button>
          </div>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle className="text-lg">导入插件包</CardTitle>
        </CardHeader>
        <CardContent className="space-y-3">
          <textarea
            className="min-h-56 w-full rounded-md border bg-background p-3 font-mono text-xs"
            value={bundle}
            onChange={(e) => setBundle(e.target.value)}
          />
          <Button onClick={importBundle} className="gap-2">
            <Download className="h-4 w-4" />
            导入 JSON Bundle
          </Button>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle className="text-lg">插件工作区列表</CardTitle>
        </CardHeader>
        <CardContent>
          {notice ? <p className="mb-3 text-sm text-muted-foreground">{notice}</p> : null}
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>插件 ID</TableHead>
                <TableHead>运行时</TableHead>
                <TableHead>来源</TableHead>
                <TableHead>审查</TableHead>
                <TableHead>启用</TableHead>
                <TableHead className="text-right">操作</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {list.map((row) => (
                <TableRow key={row.id}>
                  <TableCell>
                    <div className="font-medium">{row.name}</div>
                    <div className="font-mono text-xs text-muted-foreground">{row.plugin_id}</div>
                  </TableCell>
                  <TableCell>{row.runtime}</TableCell>
                  <TableCell>{row.source_kind}</TableCell>
                  <TableCell>
                    <Badge variant={row.review_status === 'approved' ? 'secondary' : 'outline'}>
                      {row.review_status}/{row.risk_level}
                    </Badge>
                  </TableCell>
                  <TableCell>{row.enabled ? '是' : '否'}</TableCell>
                  <TableCell className="text-right">
                    <Button size="sm" variant="secondary" asChild>
                      <Link to={`/plugin-packages/${row.id}`} className="gap-1">
                        <Code2 className="h-4 w-4" />
                        工作台
                      </Link>
                    </Button>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </CardContent>
      </Card>
    </div>
  )
}
