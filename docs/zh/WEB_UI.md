# Web 界面（管道优先）

## 技术栈

- **Vite + React + TypeScript**
- **Tailwind CSS** 与 **shadcn/ui 风格组件**（Radix、`class-variance-authority`、`lucide-react`）
- **React Flow（@xyflow/react）**：画布内拖拽节点、连接成 DAG；可从左侧插件面板拖入画布创建节点

## 路由

| 路径 | 说明 |
|------|------|
| `/` | 概览（统计与快捷操作） |
| `/pipelines` | 管道列表：编辑、异步运行、探针、任务查询 |
| `/pipelines/:id` | 管道编辑器（可拖拽 DAG + 侧栏配置） |
| `/pipeline` | 当前默认管道状态（YAML 与数据库对比） |
| `/sources`、`/items`、`/summaries`、`/interests`、`/plugins` | 数据与插件（顶栏「数据」入口） |
| `/plugins/:pluginId` | 插件通用配置页：根据 manifest 动态表单或 JSON；预设保存在 `localStorage`，拖入管道节点时合并到 `config` |
| `*`（未匹配路径） | 重定向到 `/pipelines` |

## 插件配置界面

1. **独立配置页**：在插件列表进入「配置」→ `/plugins/{id}`，使用 `PluginConfigPanel`：已 `registerPluginUI` 则渲染自定义组件；否则若有 `config_schema` 则 `DynamicManifestForm`；否则为 JSON 对象编辑器（`GenericJSONConfig`）。
2. **预设**：「保存预设」写入浏览器 `localStorage`（键 `tracker_plugin_presets_v1`）；在管道编辑器从左侧拖入该插件时，预设会合并进新节点的 `config`。
3. **自定义表单**：在 `web/src/plugin-ui/register.ts` 中调用 `registerPluginUI('插件id', YourComponent)`，组件需满足 `PluginConfigProps`。

示例：`rss` 已注册 `RssPluginConfig`（多行 feed URL）。

## 与后端对应的 API

- `POST /api/plugins/{id}/test`，请求体：`{"config":{...}}`。若插件实现 `ConfigTester`（如飞书 webhook），成功时返回 `{ "ok": true, "supported": true }`；失败时 `ok: false` 且带 `error`。未实现时 HTTP **501**，JSON 含 `supported: false`。
- `GET /api/stats`：包含 `storage_enabled`（是否已打开 SQLite）。为 `false` 时管道列表通常为空且无法 `POST /api/pipelines` 持久化。
- `GET /api/pipelines`、`GET /api/pipelines/{id}`
- `PUT /api/pipelines/{id}`：保存 `graph`（节点含 `position`、`config`，边可含 `on_condition`）
- `POST /api/pipelines/{id}/run`、`run-async`、`probe`、`rerun?from=&to=`
- `GET /api/jobs/{id}`
- `GET /api/plugins/{id}/manifest`
- `GET /api/core/ping`

## 本地开发前端

```bash
cd web && npm install && npm run dev
```

需本机已安装 Node.js 与 npm。

## 部署

见同目录 [DEPLOYMENT.md](DEPLOYMENT.md)。英文版见 [../en/DEPLOYMENT.md](../en/DEPLOYMENT.md)。
