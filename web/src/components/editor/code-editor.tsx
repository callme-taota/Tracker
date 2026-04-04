import Editor from '@monaco-editor/react'
import { cn } from '@/lib/utils'

type CodeEditorProps = {
  value: string
  onChange: (value: string) => void
  language?: string
  height?: number | string
  path?: string
  className?: string
  readOnly?: boolean
}

export function CodeEditor({
  value,
  onChange,
  language = 'plaintext',
  height = 420,
  path,
  className,
  readOnly = false,
}: CodeEditorProps) {
  return (
    <div className={cn('overflow-hidden rounded-md border bg-background', className)}>
      <Editor
        height={height}
        language={language}
        path={path}
        value={value}
        onChange={(next) => onChange(next ?? '')}
        theme="vs-dark"
        loading={<div className="p-3 text-sm text-muted-foreground">编辑器加载中…</div>}
        options={{
          automaticLayout: true,
          fontSize: 13,
          minimap: { enabled: false },
          readOnly,
          scrollBeyondLastLine: false,
          smoothScrolling: true,
          tabSize: 2,
          wordWrap: 'off',
        }}
      />
    </div>
  )
}
