const API = ''
const PIPELINE_API_KEY = (import.meta.env.VITE_TRACKER_API_KEY as string | undefined)?.trim() ?? ''

function withAuthHeaders(base?: Record<string, string>): Record<string, string> {
  const headers: Record<string, string> = { ...(base ?? {}) }
  if (PIPELINE_API_KEY) headers['X-Tracker-Api-Key'] = PIPELINE_API_KEY
  return headers
}

async function readError(r: Response): Promise<string> {
  const text = (await r.text()).trim()
  if (r.status === 401 || r.status === 403) {
    return text || '鉴权失败：请检查 VITE_TRACKER_API_KEY 与服务端 TRACKER_API_KEY 是否一致'
  }
  if (r.status === 503 && /TRACKER_API_KEY not set/i.test(text)) {
    return '服务端未配置 TRACKER_API_KEY，管道相关接口不可用'
  }
  return text || `请求失败(${r.status})`
}

export type Source = { id: number; url: string; type: string; config: string; created_at: string }
export type Interest = { id: number; name: string; keywords: string; config: string; created_at: string }
export type Plugin = {
  name: string
  version: string
  type: string
  /** DAG：是否向下游产出 item 流（服务端推断 + manifest 可覆盖） */
  emits_items?: boolean
  accepts_items?: boolean
  allow_outbound_edges?: boolean
  allow_no_incoming?: boolean
  input_formats?: string[]
  output_formats?: string[]
  compatible_with?: string[]
}
export type PluginPackage = {
  id: number
  plugin_id: string
  name: string
  runtime: string
  source_kind: string
  review_status: string
  risk_level: string
  enabled: boolean
  current_version_id?: number | null
  last_build_status?: string
  last_build_log?: string
  last_build_at?: string | null
  last_run_status?: string
  last_run_log?: string
  last_run_at?: string | null
  created_at: string
  updated_at: string
}
export type PluginPackageVersion = {
  id: number
  package_id: number
  version: string
  manifest_json: string
  entry_file: string
  build_command: string
  run_command: string
  review_report_json?: string
  source_checksum?: string
  code_dir?: string
  created_at: string
}
export type PluginPackageDetail = {
  package: PluginPackage
  current_version: PluginPackageVersion | null
  runtime_status?: {
    enabled: boolean
    loaded: boolean
    health_status?: string
    health_details?: string
    health_error?: string
  }
}
export type PluginImportRequest = {
  plugin_id: string
  name: string
  runtime: string
  source_kind?: string
  version?: string
  manifest_json: string
  entry_file: string
  build_command?: string
  run_command?: string
  files: Record<string, string>
  enabled?: boolean
}
export type ReviewFinding = { severity: string; code: string; message: string; path?: string }
export type ReviewReport = { status: string; risk_level: string; summary: string; findings: ReviewFinding[] | null }
export type PluginRunResult = {
  ok: boolean
  log: string
  error?: string
  last_build_status?: string
  last_run_status?: string
}
export type PluginGroup = {
  id: number
  name: string
  description: string
  graph: PipelineGraphDTO
  io?: Record<string, unknown>
  current_version_id?: number | null
  current_version?: string
  version_count: number
  reference_count: number
  outdated_ref_count: number
  created_at: string
  updated_at: string
}
export type PluginGroupVersion = {
  id: number
  group_id: number
  version: string
  graph: PipelineGraphDTO
  io?: Record<string, unknown>
  source_pipeline_id?: number | null
  change_note?: string
  created_at: string
}
export type PluginQualityRow = {
  name: string
  type: string
  runtime: string
  score: number
  category: string
  issues: Array<{ code: string; message: string }> | null
  has_schema: boolean
  has_tester: boolean
  needs_tester?: boolean
  has_pipeline_io?: boolean
}
export type Stage = { name: string; plugin_id: string; type: string }
export type GraphNodeDTO = {
  id: string
  plugin_type: string
  plugin_id: string
  config?: Record<string, unknown>
  position?: { x: number; y: number }
}
export type GraphEdgeDTO = {
  id: string
  source: string
  target: string
  sourceHandle?: string
  targetHandle?: string
  on_condition?: string
}
export type GraphGroupRefDTO = {
  group_id: number
  group_name?: string
  group_version_id: number
  group_version?: string
  node_ids?: string[]
}
export type PipelineGraphDTO = { name: string; nodes: GraphNodeDTO[]; edges: GraphEdgeDTO[]; group_refs?: GraphGroupRefDTO[] }
export type PipelineSummary = { id: number; name: string; is_default: boolean; updated_at: string }
export type PipelineDetail = {
  id: number
  name: string
  is_default: boolean
  created_at: string
  updated_at: string
  graph: PipelineGraphDTO
}
export type PipelineStatus =
  | {
      source: 'db'
      id: number
      name: string
      is_default: boolean
      status: string
      graph: PipelineGraphDTO
    }
  | {
      source: 'file'
      config_path?: string
      name: string
      status: string
      stages?: Stage[]
      linear_stages?: Stage[]
      error?: string
    }
export type Item = {
  id: number
  source_id: number | null
  title: string
  url: string
  content: string
  summary: string
  timestamp: string
  raw: string
  created_at: string
}
export type SummaryRow = {
  id: number
  item_id: number
  item_title: string
  item_url: string
  summary: string
  key_points: string
  created_at: string
}
export type Stats = {
  sources: number
  items: number
  interests: number
  summaries: number
  /** false 时未打开 SQLite，管道列表恒为空且无法保存多管道 */
  storage_enabled?: boolean
}

export type JobRow = {
  id: number
  pipeline_id: number
  kind: string
  status: string
  payload?: string
  attempt: number
  max_attempt: number
  error?: string
  created_at: string
  updated_at: string
}

export type PluginManifest = Record<string, unknown>

export type PluginTestResult = {
  ok: boolean
  /** 插件是否实现了服务端测试（未实现时 UI 可隐藏或提示） */
  supported: boolean
  error?: string
}

async function get<T>(path: string): Promise<T> {
  const r = await fetch(API + path, { headers: withAuthHeaders() })
  if (!r.ok) throw new Error(await readError(r))
  return r.json()
}
async function post<T>(path: string, body?: object): Promise<T> {
  const r = await fetch(API + path, {
    method: 'POST',
    headers: withAuthHeaders({ 'Content-Type': 'application/json' }),
    body: body ? JSON.stringify(body) : undefined,
  })
  if (!r.ok) throw new Error(await readError(r))
  return r.status === 204 ? (null as T) : r.json()
}
async function put(path: string, body: object): Promise<void> {
  const r = await fetch(API + path, {
    method: 'PUT',
    headers: withAuthHeaders({ 'Content-Type': 'application/json' }),
    body: JSON.stringify(body),
  })
  if (!r.ok) throw new Error(await readError(r))
}
async function del(path: string): Promise<void> {
  const r = await fetch(API + path, { method: 'DELETE', headers: withAuthHeaders() })
  if (!r.ok) throw new Error(await readError(r))
}

export const api = {
  getSources: () => get<Source[]>('/api/sources'),
  addSource: (url: string, type: string, config: string) => post<{ id: number }>('/api/sources', { url, type, config }),
  deleteSource: (id: number) => del(`/api/sources/${id}`),
  getInterests: () => get<Interest[]>('/api/interests'),
  addInterest: (name: string, keywords: string, config?: string) => post<{ id: number }>('/api/interests', { name, keywords, config: config || '' }),
  deleteInterest: (id: number) => del(`/api/interests/${id}`),
  getPlugins: () => get<Plugin[]>('/api/plugins'),
  getPluginQualityReport: () => get<PluginQualityRow[]>('/api/plugins/quality-report'),
  listPluginPackages: () => get<PluginPackage[]>('/api/plugin-packages'),
  createPluginPackage: (body: PluginImportRequest) => post<PluginPackageDetail & { review: ReviewReport }>('/api/plugin-packages', body),
  importPluginPackage: (body: PluginImportRequest) => post<PluginPackageDetail & { review: ReviewReport }>('/api/plugin-packages/import', body),
  getPluginPackage: (id: number) => get<PluginPackageDetail>(`/api/plugin-packages/${id}`),
  updatePluginPackage: (id: number, body: Partial<PluginImportRequest> & { enabled?: boolean }) =>
    put(`/api/plugin-packages/${id}`, body),
  getPluginPackageFiles: (id: number) => get<Record<string, string>>(`/api/plugin-packages/${id}/files`),
  putPluginPackageFile: (id: number, path: string, content: string) =>
    put(`/api/plugin-packages/${id}/files`, { path, content }),
  deletePluginPackageFile: (id: number, path: string) => del(`/api/plugin-packages/${id}/files?path=${encodeURIComponent(path)}`),
  reviewPluginPackage: (id: number) => post<ReviewReport>(`/api/plugin-packages/${id}/review`),
  buildPluginPackage: (id: number) => post<PluginRunResult>(`/api/plugin-packages/${id}/build`),
  runPluginPackage: (id: number) => post<PluginRunResult>(`/api/plugin-packages/${id}/run`),
  stopPluginPackage: (id: number) => post<null>(`/api/plugin-packages/${id}/stop`),
  exportPluginPackage: (id: number) =>
    get<{ package: PluginPackage; version: PluginPackageVersion; files: Record<string, string> }>(`/api/plugin-packages/${id}/export`),
  listPluginGroups: () => get<PluginGroup[]>('/api/plugin-groups'),
  createPluginGroup: (name: string, description: string, graph: PipelineGraphDTO, io?: Record<string, unknown>) =>
    post<PluginGroup>('/api/plugin-groups', { name, description, graph, io }),
  extractPluginGroupFromPipeline: (pipeline_id: number, name: string, description: string) =>
    post<PluginGroup>('/api/plugin-groups/extract', { pipeline_id, name, description }),
  getPluginGroup: (id: number) => get<PluginGroup>(`/api/plugin-groups/${id}`),
  listPluginGroupVersions: (id: number) => get<PluginGroupVersion[]>(`/api/plugin-groups/${id}/versions`),
  updatePluginGroup: (id: number, name: string, description: string, graph: PipelineGraphDTO, io?: Record<string, unknown>) =>
    put(`/api/plugin-groups/${id}`, { name, description, graph, io }),
  deletePluginGroup: (id: number) => del(`/api/plugin-groups/${id}`),
  getItems: (limit?: number, offset?: number) =>
    get<{ items: Item[]; total: number }>(`/api/items?limit=${limit ?? 50}&offset=${offset ?? 0}`),
  getSummaries: (limit?: number, offset?: number) =>
    get<{ summaries: SummaryRow[]; total: number }>(`/api/summaries?limit=${limit ?? 50}&offset=${offset ?? 0}`),
  getStats: () => get<Stats>('/api/stats'),
  getPipelineStatus: () => get<PipelineStatus>('/api/pipeline/status'),
  runPipeline: () => post<{ items_processed: number; saved?: boolean; source?: string }>('/api/pipeline/run'),
  listPipelines: () => get<PipelineSummary[]>('/api/pipelines'),
  getPipeline: (id: number) => get<PipelineDetail>(`/api/pipelines/${id}`),
  createPipeline: (name: string, graph: PipelineGraphDTO, is_default?: boolean) =>
    post<{ id: number }>('/api/pipelines', { name, graph, is_default: !!is_default }),
  updatePipeline: (id: number, name: string, graph: PipelineGraphDTO, is_default?: boolean) =>
    put(`/api/pipelines/${id}`, { name, graph, is_default: !!is_default }),
  deletePipeline: (id: number) => del(`/api/pipelines/${id}`),
  runPipelineById: (id: number) =>
    post<{ items_processed: number; saved?: boolean; pipeline_id?: number }>(`/api/pipelines/${id}/run`),
  enqueuePipelineRun: (id: number) => post<{ job_id: number }>(`/api/pipelines/${id}/run-async`),
  probePipeline: (id: number) => post<{ ok: boolean; error?: string; nodes?: unknown[] }>(`/api/pipelines/${id}/probe`),
  rerunPipeline: (id: number, from?: string, to?: string) => {
    const q = new URLSearchParams()
    if (from) q.set('from', from)
    if (to) q.set('to', to)
    const qs = q.toString()
    return post<{ job_id: number; payload?: Record<string, string> }>(
      `/api/pipelines/${id}/rerun${qs ? `?${qs}` : ''}`,
    )
  },
  getJob: (id: number) => get<JobRow>(`/api/jobs/${id}`),
  getCorePing: () => get<{ llm_profiles: string[]; storage_ping: Record<string, string> }>('/api/core/ping'),
  getPluginManifest: (pluginId: string) => get<PluginManifest>(`/api/plugins/${encodeURIComponent(pluginId)}/manifest`),
  testPluginConfig: async (pluginId: string, config: Record<string, unknown>): Promise<PluginTestResult> => {
    const r = await fetch(
      API + `/api/plugins/${encodeURIComponent(pluginId)}/test`,
      {
        method: 'POST',
        headers: withAuthHeaders({ 'Content-Type': 'application/json' }),
        body: JSON.stringify({ config }),
      },
    )
    const text = (await r.text()).trim()
    let data: { ok?: boolean; supported?: boolean; error?: string } = {}
    try {
      data = JSON.parse(text) as typeof data
    } catch {
      /* plain text error */
    }
    if (r.status === 501) {
      return { ok: false, supported: false, error: data.error || text || '不支持在线测试' }
    }
    if (!r.ok) {
      if (r.status === 401 || r.status === 403) {
        return { ok: false, supported: data.supported !== false, error: '鉴权失败：请检查 API Key 配置' }
      }
      if (r.status === 503 && /TRACKER_API_KEY not set/i.test(text)) {
        return { ok: false, supported: data.supported !== false, error: '服务端未配置 TRACKER_API_KEY' }
      }
      return { ok: false, supported: data.supported !== false, error: data.error || text }
    }
    return {
      ok: data.ok === true,
      supported: data.supported !== false,
      error: data.error,
    }
  },
}
