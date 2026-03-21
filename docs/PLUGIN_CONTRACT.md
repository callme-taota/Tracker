# Tracker 插件契约（Manifest + PluginHub）

本文档与 [PLUGIN_ECOSYSTEM.md](./PLUGIN_ECOSYSTEM.md)、[PIPELINE_GRAPH.md](./PIPELINE_GRAPH.md) 对齐。

## 1. Manifest

每个内置插件在 [`internal/plugins/registry/manifests.go`](../internal/plugins/registry/manifests.go) 注册 `plugin.Manifest`：

| 字段 | 含义 |
|------|------|
| `id` | 与 `plugin.Name()` 一致 |
| `version` | 清单版本 |
| `kind` | `source` / `processor` / `summary` / `interest` / `dispatch` |
| `config_schema` | JSON（轻量子集）：`type: object`、`required[]` 等，用于保存管道时校验节点 `config` |
| `input_schema` / `output_schema` | 预留完整 JSON Schema；当前用于文档与后续校验扩展 |
| `input_formats` / `output_formats` | **格式令牌**（如 `tracker.item.v1`），用于边上 **IO 兼容性** 检查 |
| `validator_ref` | 可选，注册在 Hub 上的具名校验函数 |

插件可实现 `plugin.ManifestProvider` 自行返回 Manifest；内置插件当前以集中清单为准。

## 2. 格式令牌与边校验

- 默认文章载荷：`tracker.item.v1`（与 `model.Item` 字段对应，`extra` 见模型注释）。
- `input_formats` 为空：表示不限制上游。
- 含 `"*"`：接受任意上游输出。
- 边上要求：上游 `output_formats` 与下游 `input_formats` 至少有一个交集（或上述宽松规则）。

`PUT/POST /api/pipelines` 在写入前调用 `Hub.ValidatePipelineGraph`。

## 3. PluginHub 生命周期

`internal/pluginhub.Hub` 在 [`GraphRunner`](../internal/core/graph_runner.go) 中于以下时机 `Emit`：

- `PipelineStart` / `PipelineEnd`
- `NodeStart` / `NodeEnd`
- `ErrorEvent`（节点失败或未知类型）

订阅：`Hub.Subscribe(func(rt *RuntimeContext, ev Event) { ... })` — 可用于告警、指标、日志（见后续分发/探针阶段）。

## 4. RuntimeContext

`pluginhub.RuntimeContext` 包含：`Ctx`、`RunID`、`PipelineID`、`NodeID`、`JobID`、`Core`（后续 Core 服务注入）、`Payload`（异步任务 / `POST .../rerun` 传入的键值，会合并进各 **source** 节点 `config`，例如 `time_from` / `time_to` RFC3339 供 RSS 等过滤）。

## 5. API

- `GET /api/plugins/{id}/manifest`：返回 JSON Manifest（供前端 Schema 表单使用）。
