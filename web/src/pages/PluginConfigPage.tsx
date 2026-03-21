import { useCallback, useEffect, useState } from 'react'
import { Link, useParams } from 'react-router-dom'
import { ArrowLeft, Check, ClipboardCopy, Trash2, Zap } from 'lucide-react'
import { api } from '@/api'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Separator } from '@/components/ui/separator'
import { clearPluginPreset, loadPluginPreset, savePluginPreset } from '@/lib/plugin-presets'
import { PluginConfigPanel } from '@/plugin-ui/PluginConfigPanel'

type ManifestInfo = {
  id?: string
  display_name?: string
  kind?: string
  version?: string
}

export default function PluginConfigPage() {
  const { pluginId: rawId } = useParams()
  const pluginId = rawId ? decodeURIComponent(rawId) : ''

  const [config, setConfig] = useState<Record<string, unknown>>({})
  const [manifest, setManifest] = useState<ManifestInfo | null>(null)
  const [manifestErr, setManifestErr] = useState<string | null>(null)
  const [savedHint, setSavedHint] = useState<string | null>(null)
  const [testLoading, setTestLoading] = useState(false)
  const [testResult, setTestResult] = useState<{ ok: boolean; text: string } | null>(null)

  useEffect(() => {
    if (!pluginId) return
    setConfig(loadPluginPreset(pluginId))
    setManifest(null)
    setManifestErr(null)
    let cancelled = false
    api
      .getPluginManifest(pluginId)
      .then((m) => {
        if (!cancelled) setManifest(m as ManifestInfo)
      })
      .catch((e) => {
        if (!cancelled) setManifestErr(String(e))
      })
    return () => {
      cancelled = true
    }
  }, [pluginId])

  const persist = useCallback(() => {
    if (!pluginId) return
    savePluginPreset(pluginId, config)
    setSavedHint('已保存到本浏览器（管道拖入节点时会自动带上此预设）')
    window.setTimeout(() => setSavedHint(null), 4000)
  }, [pluginId, config])

  const copyJSON = async () => {
    try {
      await navigator.clipboard.writeText(JSON.stringify(config, null, 2))
      setSavedHint('已复制 JSON 到剪贴板')
      window.setTimeout(() => setSavedHint(null), 3000)
    } catch {
      setSavedHint('复制失败')
    }
  }

  const runTest = async () => {
    if (!pluginId) return
    setTestLoading(true)
    setTestResult(null)
    try {
      const r = await api.testPluginConfig(pluginId, config)
      if (!r.supported) {
        setTestResult({ ok: false, text: '该插件未实现服务端连接测试。' })
        return
      }
      setTestResult({
        ok: r.ok,
        text: r.ok ? '测试通过。' : (r.error ?? '测试失败'),
      })
    } catch (e) {
      setTestResult({ ok: false, text: String(e) })
    } finally {
      setTestLoading(false)
    }
  }

  const resetLocal = () => {
    if (!pluginId) return
    if (!window.confirm('清除本插件在本机的保存配置？')) return
    clearPluginPreset(pluginId)
    setConfig({})
    setSavedHint('已清除本地预设')
    window.setTimeout(() => setSavedHint(null), 3000)
  }

  if (!pluginId) {
    return (
      <Card>
        <CardContent className="pt-6">
          <p className="text-muted-foreground">无效的插件 ID</p>
          <Button className="mt-4" asChild>
            <Link to="/plugins">返回列表</Link>
          </Button>
        </CardContent>
      </Card>
    )
  }

  const title = manifest?.display_name || pluginId

  return (
    <div className="mx-auto max-w-2xl space-y-6">
      <div className="flex flex-wrap items-center gap-2">
        <Button variant="ghost" size="sm" asChild>
          <Link to="/plugins" className="gap-1">
            <ArrowLeft className="h-4 w-4" />
            插件列表
          </Link>
        </Button>
      </div>

      <Card>
        <CardHeader>
          <div className="flex flex-wrap items-start justify-between gap-2">
            <div>
              <CardTitle className="text-2xl">{title}</CardTitle>
              <CardDescription className="mt-1 font-mono text-xs">{pluginId}</CardDescription>
            </div>
            <div className="flex flex-wrap gap-2">
              {manifest?.kind ? <Badge variant="secondary">{manifest.kind}</Badge> : null}
              {manifest?.version ? (
                <Badge variant="outline" className="font-mono">
                  v{manifest.version}
                </Badge>
              ) : null}
            </div>
          </div>
          {manifestErr ? (
            <p className="text-sm text-destructive">Manifest 加载失败：{manifestErr}（仍可编辑本地 JSON）</p>
          ) : null}
        </CardHeader>
        <Separator />
        <CardContent className="space-y-6 pt-6">
          <p className="text-sm text-muted-foreground">
            此处修改的是<strong>本浏览器保存的默认配置</strong>，保存后：在管道编辑器中<strong>从左侧拖入该插件</strong>时会自动合并到节点{' '}
            <code className="rounded bg-muted px-1">config</code>。生产环境密钥仍建议用环境变量或 YAML。
          </p>

          <PluginConfigPanel pluginId={pluginId} value={config} onChange={setConfig} />

          {savedHint ? (
            <p className="text-sm text-emerald-600 dark:text-emerald-400">{savedHint}</p>
          ) : null}

          {testResult ? (
            <p
              className={
                testResult.ok
                  ? 'text-sm text-emerald-600 dark:text-emerald-400'
                  : 'text-sm text-destructive'
              }
            >
              {testResult.text}
            </p>
          ) : null}

          <div className="flex flex-wrap gap-2">
            <Button type="button" variant="secondary" onClick={runTest} disabled={testLoading}>
              <Zap className="h-4 w-4" />
              {testLoading ? '测试中…' : '测试连接 / 配置'}
            </Button>
            <Button onClick={persist}>
              <Check className="h-4 w-4" />
              保存预设
            </Button>
            <Button type="button" variant="secondary" onClick={copyJSON}>
              <ClipboardCopy className="h-4 w-4" />
              复制 JSON
            </Button>
            <Button type="button" variant="outline" onClick={resetLocal}>
              <Trash2 className="h-4 w-4" />
              清除本地
            </Button>
          </div>
        </CardContent>
      </Card>
    </div>
  )
}
