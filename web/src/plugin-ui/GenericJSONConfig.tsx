import { useEffect, useState } from 'react'
import { Label } from '@/components/ui/label'
import type { PluginConfigProps } from './types'

/** 无 config_schema 或 schema 为空时的通用 JSON 配置编辑器 */
export function GenericJSONConfig({ value, onChange, disabled }: PluginConfigProps) {
  const [text, setText] = useState('')
  const [parseErr, setParseErr] = useState<string | null>(null)

  useEffect(() => {
    try {
      setText(JSON.stringify(value ?? {}, null, 2))
      setParseErr(null)
    } catch {
      setText('{}')
      setParseErr('无法序列化当前配置')
    }
  }, [value])

  const apply = (raw: string) => {
    setText(raw)
    const t = raw.trim()
    if (!t) {
      setParseErr(null)
      onChange({})
      return
    }
    try {
      const p = JSON.parse(t) as unknown
      if (p === null || typeof p !== 'object' || Array.isArray(p)) {
        setParseErr('配置必须是 JSON 对象 {}')
        return
      }
      setParseErr(null)
      onChange(p as Record<string, unknown>)
    } catch (e) {
      setParseErr(String(e))
    }
  }

  return (
    <div className="space-y-2">
      <Label htmlFor="plugin-json-cfg">配置（JSON 对象）</Label>
      <p className="text-xs text-muted-foreground">
        该插件未提供结构化 config_schema，请直接编辑 JSON。保存语法正确后才会写入。
      </p>
      <textarea
        id="plugin-json-cfg"
        disabled={disabled}
        spellCheck={false}
        className="flex min-h-[220px] w-full rounded-md border border-input bg-transparent px-3 py-2 font-mono text-sm shadow-sm focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring disabled:cursor-not-allowed disabled:opacity-50"
        value={text}
        onChange={(e) => apply(e.target.value)}
      />
      {parseErr ? <p className="text-sm text-destructive">{parseErr}</p> : null}
    </div>
  )
}
