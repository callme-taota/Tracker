import type { ReactNode } from 'react'
import { useEffect, useMemo, useState } from 'react'
import { api } from '@/api'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Checkbox } from '@/components/ui/checkbox'
import { cn } from '@/lib/utils'
import type { PluginConfigProps } from './types'

type JsonSchemaProperty = {
  type?: string
  description?: string
  /** JSON Schema enum */
  enum?: unknown
}

type ConfigSchema = {
  type?: string
  properties?: Record<string, JsonSchemaProperty>
  required?: string[]
}

function parseSchema(raw: unknown): ConfigSchema | null {
  if (!raw) return null
  if (typeof raw === 'string') {
    try {
      return JSON.parse(raw) as ConfigSchema
    } catch {
      return null
    }
  }
  if (typeof raw === 'object') return raw as ConfigSchema
  return null
}

function stringifyVal(key: string, v: unknown, prop: JsonSchemaProperty): string {
  if (v === undefined || v === null) return ''
  if (prop.type === 'array') {
    if (Array.isArray(v)) return v.map(String).join('\n')
    return String(v)
  }
  if (typeof v === 'object') return JSON.stringify(v, null, 2)
  return String(v)
}

function enumOptions(prop: JsonSchemaProperty): string[] {
  if (!Array.isArray(prop.enum)) return []
  return prop.enum.filter((x): x is string => typeof x === 'string')
}

function parseVal(_key: string, text: string, prop: JsonSchemaProperty): unknown {
  const t = text.trim()
  if (prop.type === 'array') {
    if (!t) return []
    return t
      .split(/\r?\n/)
      .map((s) => s.trim())
      .filter(Boolean)
  }
  if (prop.type === 'number') {
    if (!t) return undefined
    const n = Number(t)
    return Number.isFinite(n) ? n : t
  }
  if (prop.type === 'boolean') {
    if (!t) return false
    return t === 'true' || t === '1'
  }
  return t
}

export function DynamicManifestForm({
  pluginId,
  value,
  onChange,
  disabled,
  fallback,
}: PluginConfigProps & { pluginId: string; fallback?: ReactNode }) {
  const [manifest, setManifest] = useState<Record<string, unknown> | null>(null)
  const [err, setErr] = useState<string | null>(null)

  useEffect(() => {
    if (!pluginId) {
      setManifest(null)
      return
    }
    let cancelled = false
    api
      .getPluginManifest(pluginId)
      .then((m) => {
        if (!cancelled) setManifest(m)
      })
      .catch((e) => {
        if (!cancelled) setErr(String(e))
      })
    return () => {
      cancelled = true
    }
  }, [pluginId])

  const schema = useMemo(() => parseSchema(manifest?.config_schema), [manifest])

  const entries = useMemo(() => {
    const props = schema?.properties
    if (!props || !Object.keys(props).length) return [] as { key: string; prop: JsonSchemaProperty }[]
    return Object.entries(props).map(([key, prop]) => ({ key, prop }))
  }, [schema])

  if (!pluginId) {
    return <p className="text-sm text-muted-foreground">缺少插件 ID</p>
  }
  if (err) {
    if (fallback != null) {
      return (
        <div className="space-y-3">
          <p className="text-sm text-amber-700 dark:text-amber-400">无法加载 manifest：{err}</p>
          {fallback}
        </div>
      )
    }
    return <p className="text-sm text-destructive">{err}</p>
  }
  if (!manifest) {
    return <p className="text-sm text-muted-foreground">加载 manifest…</p>
  }
  if (!entries.length) {
    if (fallback != null) return <>{fallback}</>
    return (
      <p className="text-sm text-muted-foreground">
        该插件未提供 config_schema，请使用通用 JSON 配置（由外层提供）。
      </p>
    )
  }

  const patch = (key: string, text: string, prop: JsonSchemaProperty) => {
    const next = { ...value }
    const parsed = parseVal(key, text, prop)
    if (parsed === undefined || parsed === '') {
      delete next[key]
    } else {
      next[key] = parsed as unknown
    }
    onChange(next)
  }

  const patchDirect = (key: string, v: unknown) => {
    const next = { ...value }
    if (v === undefined || v === '') {
      delete next[key]
    } else {
      next[key] = v as unknown
    }
    onChange(next)
  }

  const selectClass = cn(
    'flex h-9 w-full rounded-md border border-input bg-transparent px-3 py-1 text-sm shadow-sm',
    'focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring disabled:cursor-not-allowed disabled:opacity-50',
  )

  return (
    <div className="flex flex-col gap-4">
      {entries.map(({ key, prop }) => {
        const options = enumOptions(prop)
        return (
          <div key={key} className="space-y-2">
            <Label htmlFor={`cfg-${key}`}>
              {key}
              {schema?.required?.includes(key) ? <span className="text-destructive"> *</span> : null}
            </Label>
            {prop.description ? <p className="text-xs text-muted-foreground">{prop.description}</p> : null}
            {options.length > 0 ? (
              <select
                id={`cfg-${key}`}
                disabled={disabled}
                className={selectClass}
                value={String(value[key] ?? '')}
                onChange={(e) => {
                  const t = e.target.value
                  patchDirect(key, t === '' ? undefined : t)
                }}
              >
                <option value="">（未设置）</option>
                {options.map((opt) => (
                  <option key={opt} value={opt}>
                    {opt}
                  </option>
                ))}
              </select>
            ) : prop.type === 'boolean' ? (
              <div className="flex items-center gap-2">
                <Checkbox
                  id={`cfg-${key}`}
                  disabled={disabled}
                  checked={value[key] === true}
                  onCheckedChange={(c) => patchDirect(key, c === true)}
                />
                <Label htmlFor={`cfg-${key}`} className="cursor-pointer font-normal">
                  启用
                </Label>
              </div>
            ) : prop.type === 'number' || prop.type === 'integer' ? (
              <Input
                id={`cfg-${key}`}
                type="number"
                disabled={disabled}
                value={value[key] === undefined || value[key] === null ? '' : String(value[key])}
                onChange={(e) => patch(key, e.target.value, prop)}
              />
            ) : prop.type === 'array' ? (
              <textarea
                id={`cfg-${key}`}
                disabled={disabled}
                className="flex min-h-[120px] w-full rounded-md border border-input bg-transparent px-3 py-2 text-sm shadow-sm placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring disabled:cursor-not-allowed disabled:opacity-50"
                placeholder="每行一项"
                value={stringifyVal(key, value[key], prop)}
                onChange={(e) => patch(key, e.target.value, prop)}
              />
            ) : (
              <Input
                id={`cfg-${key}`}
                disabled={disabled}
                value={stringifyVal(key, value[key], prop)}
                onChange={(e) => patch(key, e.target.value, prop)}
              />
            )}
          </div>
        )
      })}
    </div>
  )
}
