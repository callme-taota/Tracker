import { Label } from '@/components/ui/label'
import type { PluginConfigProps } from '../types'

/** RSS 插件专用配置：每行一个 feed URL */
export function RssPluginConfig({ value, onChange, disabled }: PluginConfigProps) {
  const feeds = Array.isArray(value.feeds) ? (value.feeds as unknown[]).map(String) : []
  const text = feeds.join('\n')

  return (
    <div className="space-y-2">
      <Label htmlFor="rss-feeds">RSS 地址（每行一个）</Label>
      <p className="text-xs text-muted-foreground">
        每行一个 RSS URL。与管道节点 <code className="rounded bg-muted px-0.5">config</code> 及全局配置互补；留空则使用服务端默认
        feeds。
      </p>
      <textarea
        id="rss-feeds"
        disabled={disabled}
        className="flex min-h-[140px] w-full rounded-md border border-input bg-transparent px-3 py-2 text-sm shadow-sm placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring disabled:cursor-not-allowed disabled:opacity-50"
        value={text}
        onChange={(e) => {
          const lines = e.target.value
            .split(/\r?\n/)
            .map((s) => s.trim())
            .filter(Boolean)
          const next = { ...value, feeds: lines }
          onChange(next)
        }}
      />
    </div>
  )
}
