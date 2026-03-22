# 管道图（动态配置与节点编辑器）

本文描述**动态可配置**的管道模型，以及与节点编辑器（类达芬奇节点、蓝图式图）的产品与技术对齐。

## 1. 产品目标

| 目标 | 说明 |
|------|------|
| **动态配置** | 管道定义可持久化（SQLite + API），不必只改 YAML 重启；支持多管道与默认管道指针。 |
| **节点图** | 管道由 **节点 + 边** 构成 **DAG**；线性 `stages` 只是 DAG 的特例。 |
| **可视化编辑** | Web 端可拖拽节点、连线；保存时写回后端图模型（`position` 一并保存）。 |
| **执行语义** | 引擎按 **拓扑序** 执行；`source` 产出条目列表，`process` 类处理上游合并后的条目，`dispatch` 消费条目。 |

## 2. 数据模型（JSON / 数据库）

```json
{
  "name": "default",
  "nodes": [
    { "id": "n1", "plugin_type": "source", "plugin_id": "rss", "config": {}, "position": { "x": 0, "y": 0 } },
    { "id": "n2", "plugin_type": "processor", "plugin_id": "clean", "config": {}, "position": { "x": 200, "y": 0 } }
  ],
  "edges": [
    { "id": "e1", "source": "n1", "target": "n2" },
    { "id": "e2", "source": "n1", "target": "n3", "on_condition": "on_failure" }
  ]
}
```

- **`on_condition`**：`on_success` 或空 — 参与主 DAG 拓扑与数据流；`on_failure` — 条件边（当前 **不参与** 拓扑与 `GraphRunner` 数据合并，仅做存在性校验）；非 `source` 节点须至少有一条 **success** 入边。
- **持久化**：表 `pipeline_definitions`（`name`、`graph_json`、`is_default`、时间戳）；`is_default=1` 的行被 `GET/POST /api/pipeline/*` 优先使用，否则回退 YAML。
- **任务**：`jobs` — `POST /api/pipelines/{id}/run-async` 入队，`internal/worker` 在 `serve` 模式下轮询；`GET /api/jobs/{id}` 查状态；失败按 `max_attempt` 重试。
- **探针**：`POST /api/pipelines/{id}/probe` — Hub 校验与各节点 manifest/config 探测。
- **兼容**：旧版 YAML `stages[]` 可经 `internal/pipeline.LinearToGraph` 转为单链图。
- **校验**：success 子图无环；至少一个 `source`；边端点必须存在。

## 3. 执行器（后端）

- `internal/core/graph_runner.go`：`GraphRunner.Run` — 拓扑排序后按节点类型调用 `AgentRunner`。
- 多前驱汇入 `process`：上游条目列表 **拼接** 后逐条处理（后续可扩展合并策略）。
- `Engine.RunGraph` 与 `Engine.Run`（线性）共用同一 `GraphRunner` 语义。

## 4. REST API

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/pipelines` | 列出管道摘要 |
| GET | `/api/pipelines/{id}` | 获取完整图 JSON |
| POST | `/api/pipelines` | 新建管道 |
| PUT | `/api/pipelines/{id}` | 更新图（含节点坐标） |
| DELETE | `/api/pipelines/{id}` | 删除 |
| POST | `/api/pipelines/{id}/run` | 同步按图执行 |
| POST | `/api/pipelines/{id}/run-async` | 异步入队 |
| POST | `/api/pipelines/{id}/probe` | 探针 |
| POST | `/api/pipelines/{id}/rerun` | 按查询参数从某节点重跑 |
| GET | `/api/jobs/{id}` | 任务状态 |

`GET /api/pipeline/status`、`POST /api/pipeline/run`：若存在 `is_default=1` 的数据库管道则使用其 `graph_json`，否则使用 `config.yaml` / `configs/default.yaml` 中的线性 YAML。

实现见 `internal/storage/pipelines.go` 与 `internal/api/server.go`。

## 5. 前端（节点编辑器）

- 技术：**React Flow**（`@xyflow/react`）+ Tailwind / shadcn 风格。
- 路由：**`/pipelines/:id`**（`web/src/pages/PipelineEditor.tsx`）。
- 左侧插件库拖入画布添加节点；**保存** 调用 `PUT /api/pipelines/{id}`。
- 更多界面与 API 对应关系见 [WEB_UI.md](WEB_UI.md)。

## 6. 与 ARC「接口层」对齐

Web 平台作为接口层：管道管理、插件列表、节点编辑、运行与监控均通过 **HTTP API** 与核心交互。

## 7. 首次用数据库承载默认管道

表为空时，`/api/pipeline/run` 仍回退 YAML。若希望完全走数据库图，可：

`POST /api/pipelines`，body 含 `name`、`is_default: true` 与通过校验的 `graph`。

`tracker run` / `internal/scheduler` 仍以 YAML 为主；与 Web 默认图完全对齐可作为后续开关项。
