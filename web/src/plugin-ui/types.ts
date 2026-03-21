export type PluginConfigProps = {
  /** 当前节点 config（与后端 graph node.config 一致） */
  value: Record<string, unknown>
  onChange: (next: Record<string, unknown>) => void
  /** 可选：展示用 */
  disabled?: boolean
}
