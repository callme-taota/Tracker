import { useEffect, useMemo, useState } from 'react'
import { Link, useParams } from 'react-router-dom'
import { ArrowLeft, FilePlus2, Play, Save, ShieldAlert, Square, Trash2, Upload, Wrench } from 'lucide-react'
import { api, type PluginPackageDetail, type ReviewReport } from '@/api'
import { CodeEditor } from '@/components/editor/code-editor'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Badge } from '@/components/ui/badge'

type TreeNode = {
  name: string
  path: string
  kind: 'dir' | 'file'
  children?: TreeNode[]
}

function buildFileTree(paths: string[]): TreeNode[] {
  const roots: TreeNode[] = []
  for (const fullPath of [...paths].sort()) {
    const parts = fullPath.split('/')
    let level = roots
    let current = ''
    parts.forEach((part, index) => {
      current = current ? `${current}/${part}` : part
      const isFile = index === parts.length - 1
      let node = level.find((item) => item.path === current)
      if (!node) {
        node = { name: part, path: current, kind: isFile ? 'file' : 'dir', children: isFile ? undefined : [] }
        level.push(node)
      }
      if (!isFile) level = node.children ?? []
    })
  }
  return roots
}

function FileTree({
  nodes,
  selectedPath,
  onSelect,
}: {
  nodes: TreeNode[]
  selectedPath: string
  onSelect: (path: string) => void
}) {
  return (
    <div className="space-y-1">
      {nodes.map((node) =>
        node.kind === 'dir' ? (
          <div key={node.path} className="space-y-1">
            <div className="px-2 text-xs font-medium text-muted-foreground">{node.name}</div>
            <div className="ml-3 border-l pl-2">
              <FileTree nodes={node.children ?? []} selectedPath={selectedPath} onSelect={onSelect} />
            </div>
          </div>
        ) : (
          <button
            key={node.path}
            type="button"
            className={`w-full rounded px-2 py-1 text-left text-sm ${selectedPath === node.path ? 'bg-secondary' : 'hover:bg-accent'}`}
            onClick={() => onSelect(node.path)}
          >
            {node.name}
          </button>
        ),
      )}
    </div>
  )
}

function editorLanguageForPath(path: string): string {
  const lower = path.toLowerCase()
  if (lower.endsWith('.ts') || lower.endsWith('.tsx')) return 'typescript'
  if (lower.endsWith('.js') || lower.endsWith('.jsx')) return 'javascript'
  if (lower.endsWith('.go')) return 'go'
  if (lower.endsWith('.rs')) return 'rust'
  if (lower.endsWith('.json')) return 'json'
  if (lower.endsWith('.md')) return 'markdown'
  if (lower.endsWith('.yaml') || lower.endsWith('.yml')) return 'yaml'
  if (lower.endsWith('.sh')) return 'shell'
  return 'plaintext'
}

export default function PluginWorkbenchPage() {
  const { id } = useParams()
  const packageId = Number(id)
  const [detail, setDetail] = useState<PluginPackageDetail | null>(null)
  const [files, setFiles] = useState<Record<string, string>>({})
  const [selectedPath, setSelectedPath] = useState('')
  const [code, setCode] = useState('')
  const [manifestJSON, setManifestJSON] = useState('')
  const [buildCommand, setBuildCommand] = useState('')
  const [runCommand, setRunCommand] = useState('')
  const [notice, setNotice] = useState<string | null>(null)
  const [review, setReview] = useState<ReviewReport | null>(null)
  const [newFilePath, setNewFilePath] = useState('')
  const [busy, setBusy] = useState<'review' | 'build' | 'run' | 'stop' | null>(null)

  const load = async () => {
    if (!packageId) return
    try {
      const [pkg, fs] = await Promise.all([api.getPluginPackage(packageId), api.getPluginPackageFiles(packageId)])
      setDetail(pkg)
      const safeFiles = fs && typeof fs === 'object' ? fs : {}
      setFiles(safeFiles)
      const first = Object.keys(safeFiles)[0] ?? pkg.current_version?.entry_file ?? ''
      setSelectedPath(first)
      setCode(first ? safeFiles[first] ?? '' : '')
      setManifestJSON(pkg.current_version?.manifest_json ?? '')
      setBuildCommand(pkg.current_version?.build_command ?? '')
      setRunCommand(pkg.current_version?.run_command ?? '')
      if (pkg.current_version?.review_report_json) {
        try {
          const parsed = JSON.parse(pkg.current_version.review_report_json) as ReviewReport
          setReview({
            ...parsed,
            findings: Array.isArray(parsed.findings) ? parsed.findings : [],
          })
        } catch {
          setReview(null)
        }
      } else {
        setReview(null)
      }
    } catch (e) {
      setNotice(String(e))
    }
  }

  useEffect(() => {
    load()
  }, [packageId])

  useEffect(() => {
    if (selectedPath) setCode(files[selectedPath] ?? '')
  }, [files, selectedPath])

  const reviewBadges = useMemo(() => {
    if (!detail?.package) return null
    return (
      <div className="flex flex-wrap gap-2">
        <Badge variant="secondary">{detail.package.runtime}</Badge>
        <Badge variant="outline">{detail.package.source_kind}</Badge>
        <Badge variant={detail.package.review_status === 'approved' ? 'secondary' : 'outline'}>
          {detail.package.review_status}/{detail.package.risk_level}
        </Badge>
      </div>
    )
  }, [detail])

  const fileTree = useMemo(() => buildFileTree(Object.keys(files)), [files])

  if (!packageId) {
    return <p className="text-muted-foreground">无效插件包 ID</p>
  }

  const saveCurrentFile = async () => {
    if (!selectedPath) return
    try {
      await api.putPluginPackageFile(packageId, selectedPath, code)
      setFiles((prev) => ({ ...prev, [selectedPath]: code }))
      setNotice(`已保存 ${selectedPath}`)
    } catch (e) {
      setNotice(String(e))
    }
  }

  const createFile = async () => {
    const path = newFilePath.trim()
    if (!path) return
    try {
      await api.putPluginPackageFile(packageId, path, files[path] ?? '')
      setFiles((prev) => ({ ...prev, [path]: prev[path] ?? '' }))
      setSelectedPath(path)
      setCode(files[path] ?? '')
      setNewFilePath('')
      setNotice(`已创建 ${path}`)
    } catch (e) {
      setNotice(String(e))
    }
  }

  const deleteCurrentFile = async () => {
    if (!selectedPath) return
    if (!window.confirm(`确认删除文件「${selectedPath}」？`)) return
    try {
      await api.deletePluginPackageFile(packageId, selectedPath)
      const next = { ...files }
      delete next[selectedPath]
      const first = Object.keys(next)[0] ?? ''
      setFiles(next)
      setSelectedPath(first)
      setCode(first ? next[first] ?? '' : '')
      setNotice(`已删除 ${selectedPath}`)
    } catch (e) {
      setNotice(String(e))
    }
  }

  const saveMetadata = async () => {
    try {
      await api.updatePluginPackage(packageId, {
        manifest_json: manifestJSON,
        build_command: buildCommand,
        run_command: runCommand,
        enabled: detail?.package.enabled ?? false,
      })
      setNotice('已保存插件元数据')
      await load()
    } catch (e) {
      setNotice(String(e))
    }
  }

  const doReview = async () => {
    try {
      setBusy('review')
      const r = await api.reviewPluginPackage(packageId)
      setReview(r)
      setNotice(`审查完成：${r.status}/${r.risk_level}`)
      await load()
    } catch (e) {
      setNotice(String(e))
    } finally {
      setBusy(null)
    }
  }

  const doBuild = async () => {
    try {
      setBusy('build')
      const r = await api.buildPluginPackage(packageId)
      setNotice(r.ok ? `构建成功\n${r.log}` : `构建失败\n${r.log}\n${r.error ?? ''}`)
      await load()
    } catch (e) {
      setNotice(String(e))
    } finally {
      setBusy(null)
    }
  }

  const doRun = async () => {
    try {
      setBusy('run')
      const r = await api.runPluginPackage(packageId)
      setNotice(r.ok ? `已构建并同步运行时\n${r.log}` : `运行失败\n${r.log}\n${r.error ?? ''}`)
      await load()
    } catch (e) {
      setNotice(String(e))
    } finally {
      setBusy(null)
    }
  }

  const doStop = async () => {
    try {
      setBusy('stop')
      await api.stopPluginPackage(packageId)
      setNotice('已停用该插件包')
      await load()
    } catch (e) {
      setNotice(String(e))
    } finally {
      setBusy(null)
    }
  }

  const doExport = async () => {
    try {
      const bundle = await api.exportPluginPackage(packageId)
      await navigator.clipboard.writeText(JSON.stringify(bundle, null, 2))
      setNotice('已复制导出 bundle 到剪贴板')
    } catch (e) {
      setNotice(String(e))
    }
  }

  return (
    <div className="space-y-6">
      <div className="flex flex-wrap items-center gap-2">
        <Button variant="ghost" size="sm" asChild>
          <Link to="/plugin-packages" className="gap-1">
            <ArrowLeft className="h-4 w-4" />
            返回工作区
          </Link>
        </Button>
        {reviewBadges}
      </div>

      <Card>
        <CardHeader>
          <CardTitle>高级插件工作台</CardTitle>
        </CardHeader>
        <CardContent className="grid gap-6 lg:grid-cols-[260px_minmax(0,1fr)]">
          <div className="space-y-3">
            <div className="text-sm font-medium">文件树</div>
            <div className="flex gap-2">
              <Input value={newFilePath} onChange={(e) => setNewFilePath(e.target.value)} placeholder="src/index.ts" />
              <Button variant="outline" size="icon" onClick={createFile}>
                <FilePlus2 className="h-4 w-4" />
              </Button>
            </div>
            <div className="max-h-[60vh] overflow-y-auto rounded-md border p-2">
              <FileTree nodes={fileTree} selectedPath={selectedPath} onSelect={setSelectedPath} />
            </div>
            <Button variant="outline" size="sm" onClick={doExport} className="w-full gap-1">
              <Upload className="h-4 w-4" />
              导出 Bundle
            </Button>
          </div>

          <div className="space-y-4">
            <div className="grid gap-3 xl:grid-cols-3">
              <Card>
                <CardHeader className="pb-2">
                  <CardTitle className="text-base">运行状态</CardTitle>
                </CardHeader>
                <CardContent className="space-y-2 text-sm">
                  <div>启用：{detail?.package.enabled ? '是' : '否'}</div>
                  <div>加载：{detail?.runtime_status?.loaded ? '已加载到运行时' : '未加载'}</div>
                  <div>健康：{detail?.runtime_status?.health_status ?? detail?.runtime_status?.health_error ?? '未知'}</div>
                  <div>上次运行：{detail?.package.last_run_status || '未运行'}</div>
                </CardContent>
              </Card>
              <Card>
                <CardHeader className="pb-2">
                  <CardTitle className="text-base">构建面板</CardTitle>
                </CardHeader>
                <CardContent className="space-y-2 text-sm">
                  <div>状态：{detail?.package.last_build_status || '未构建'}</div>
                  <div>时间：{detail?.package.last_build_at || '—'}</div>
                  <pre className="max-h-28 overflow-auto rounded border bg-muted/40 p-2 text-xs">
                    {detail?.package.last_build_log || '暂无构建日志'}
                  </pre>
                </CardContent>
              </Card>
              <Card>
                <CardHeader className="pb-2">
                  <CardTitle className="text-base">运行日志</CardTitle>
                </CardHeader>
                <CardContent className="space-y-2 text-sm">
                  <div>状态：{detail?.package.last_run_status || '未运行'}</div>
                  <div>时间：{detail?.package.last_run_at || '—'}</div>
                  <pre className="max-h-28 overflow-auto rounded border bg-muted/40 p-2 text-xs">
                    {detail?.package.last_run_log || '暂无运行日志'}
                  </pre>
                </CardContent>
              </Card>
            </div>

            <div className="grid gap-3 md:grid-cols-2">
              <div>
                <div className="mb-1 text-sm text-muted-foreground">manifest_json</div>
                <CodeEditor value={manifestJSON} onChange={setManifestJSON} language="json" height={320} path="manifest.json" />
              </div>
              <div className="space-y-3">
                <div>
                  <div className="mb-1 text-sm text-muted-foreground">build command</div>
                  <Input value={buildCommand} onChange={(e) => setBuildCommand(e.target.value)} />
                </div>
                <div>
                  <div className="mb-1 text-sm text-muted-foreground">run command</div>
                  <Input value={runCommand} onChange={(e) => setRunCommand(e.target.value)} />
                </div>
                <div className="flex flex-wrap gap-2">
                  <Button onClick={saveMetadata} className="gap-1">
                    <Save className="h-4 w-4" />
                    保存元数据
                  </Button>
                  <Button variant="secondary" onClick={doReview} className="gap-1" disabled={busy !== null}>
                    <ShieldAlert className="h-4 w-4" />
                    {busy === 'review' ? '审查中…' : '审查风险'}
                  </Button>
                  <Button variant="outline" onClick={doBuild} className="gap-1" disabled={busy !== null}>
                    <Wrench className="h-4 w-4" />
                    {busy === 'build' ? '构建中…' : '构建'}
                  </Button>
                  <Button variant="secondary" onClick={doRun} className="gap-1" disabled={busy !== null}>
                    <Play className="h-4 w-4" />
                    {busy === 'run' ? '运行中…' : '构建并运行'}
                  </Button>
                  <Button variant="outline" onClick={doStop} className="gap-1" disabled={busy !== null}>
                    <Square className="h-4 w-4" />
                    {busy === 'stop' ? '停用中…' : '停用'}
                  </Button>
                </div>
              </div>
            </div>

            <div>
              <div className="mb-1 text-sm text-muted-foreground">源码：{selectedPath || '未选择文件'}</div>
              <CodeEditor
                value={code}
                onChange={setCode}
                language={editorLanguageForPath(selectedPath)}
                height={520}
                path={selectedPath || 'untitled.txt'}
              />
              <div className="mt-2 flex gap-2">
                <Button onClick={saveCurrentFile} disabled={!selectedPath}>
                  保存当前文件
                </Button>
                <Button variant="outline" onClick={deleteCurrentFile} disabled={!selectedPath}>
                  <Trash2 className="mr-1 h-4 w-4" />
                  删除文件
                </Button>
              </div>
            </div>

            {review ? (
              <Card>
                <CardHeader>
                  <CardTitle className="text-lg">风险审查</CardTitle>
                </CardHeader>
                <CardContent className="space-y-2">
                  <p className="text-sm text-muted-foreground">
                    {review.status}/{review.risk_level}：{review.summary}
                  </p>
                  {(review.findings ?? []).map((f, idx) => (
                    <div key={`${f.code}-${idx}`} className="rounded border p-2 text-sm">
                      <div className="font-medium">
                        [{f.severity}] {f.code}
                      </div>
                      <div>{f.message}</div>
                      {f.path ? <div className="font-mono text-xs text-muted-foreground">{f.path}</div> : null}
                    </div>
                  ))}
                </CardContent>
              </Card>
            ) : null}

            {notice ? (
              <pre className="whitespace-pre-wrap rounded-md border bg-muted/40 p-3 text-xs">{notice}</pre>
            ) : null}
          </div>
        </CardContent>
      </Card>
    </div>
  )
}
