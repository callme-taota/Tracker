# Web UI（Pipeline 优先）

## 技术栈

- **Vite + React + TypeScript**
- **Tailwind CSS** + **shadcn/ui 风格组件**（Radix 原语 + `class-variance-authority` + `lucide-react`）
- **React Flow（@xyflow/react）**：画布内拖拽节点、连线 DAG；**从左侧插件库拖入画布**创建节点

## 路由

| 路径 | 说明 |
|------|------|
| `/` | 概览（统计 + 快捷运行） |
| `/pipelines` | 管道列表：编辑、异步运行、探针、任务查询 |
| `/pipelines/:id` | **管道编辑器**（可拖拽 DAG + 侧栏配置） |
| `/pipeline` | 当前默认管道状态（YAML / DB） |
| `/sources`、`/items`、`/summaries`、`/interests`、`/plugins` | 数据与插件（顶栏「数据」下拉） |
| `/plugins/:pluginId` | **插件通用配置页**：manifest 动态表单或 JSON；预设存 `localStorage`，拖入管道节点时合并 |

## 插件配置 UI（自声明 + 通用页）

1. **独立配置页**：插件列表点「配置」→ `/plugins/{id}`，使用 `PluginConfigPanel`：
   - 已 `registerPluginUI` → 自定义组件；
   - 否则 `config_schema` → `DynamicManifestForm`；
   - 无 schema 或 manifest 加载失败 → **JSON 对象编辑器**（`GenericJSONConfig`）。
2. **预设持久化**：配置页「保存预设」写入浏览器 `localStorage`（`tracker_plugin_presets_v1`）；管道编辑器**从左侧拖入**该插件时，会把预设合并进新节点的 `config`。
3. **注册自定义表单**：在 `web/src/plugin-ui/register.ts` 中 `registerPluginUI('插件id', YourComponent)`，组件满足 `PluginConfigProps`。

示例：`rss` 已注册 `RssPluginConfig`（多行 feed URL）。

## API 摘要

- `POST /api/plugins/{id}/test`，Body：`{"config":{...}}` — 若插件实现 `ConfigTester`（如飞书 webhook），返回 `{ ok, supported, error? }`；未实现时 **501** 且 `supported: false`。
- `GET /api/stats`：含 `storage_enabled`（是否已打开 SQLite）；为 `false` 时管道列表为空且无法 `POST /api/pipelines` 落库。
- `GET /api/pipelines`、`GET /api/pipelines/{id}`
- `PUT /api/pipelines/{id}`（保存 `graph`: nodes 含 `position`/`config`，edges 可含 `on_condition`）
- `POST /api/pipelines/{id}/run`、`run-async`、`probe`、`rerun?from=&to=`
- `GET /api/jobs/{id}`
- `GET /api/plugins/{id}/manifest`
- `GET /api/core/ping`

## 本地开发

```bash
cd web && npm install && npm run dev
```

（需本机已安装 Node/npm。）

## 部署

Docker 与多实例说明见 [DEPLOYMENT.md](DEPLOYMENT.md)。
