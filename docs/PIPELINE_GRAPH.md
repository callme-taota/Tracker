# Tracker 管道图（动态配置 + 节点编辑器）

本文档描述**动态可配置**的管道模型，以及与**脑图 / 节点编辑器**（类达芬奇节点、UE 蓝图）的产品与技术对齐。

---

## 1. 产品目标

| 目标 | 说明 |
|------|------|
| **动态配置** | 管道定义可持久化（SQLite + API），不依赖仅改 YAML 重启；支持多管道、默认管道指针。 |
| **节点图** | 管道 = **节点（Node）+ 边（Edge）** 构成的 DAG，非仅线性列表（线性是 DAG 特例）。 |
| **可视化编辑** | Web 端 **可拖拽** 节点、连线；存盘即写入后端图模型（位置 `position` 一并保存，便于再次打开布局一致）。 |
| **执行语义** | 引擎按 **拓扑序** 执行；`source` 产出 `[]Item`，`process` 类按上游合并后的条目逐条处理，`dispatch` 消费条目。 |

---

## 2. 数据模型（JSON / DB）

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

- **`on_condition`**：`on_success` 或空 — 参与主 DAG 拓扑与数据流；`on_failure` — 预留条件边（当前 **不参与** 拓扑与 `GraphRunner` 数据合并，仅校验存在性）；非 `source` 节点必须至少有一条 **success** 入边。
- **持久化**：SQLite 表 `pipeline_definitions`（`name`, `graph_json`, `is_default`, 时间戳）；`is_default=1` 的行被 `GET/POST /api/pipeline/*` 优先使用，否则回退 YAML。
- **任务表**：`jobs` — `POST /api/pipelines/{id}/run-async` 入队，`internal/worker` 在 `serve` 模式下轮询执行；`GET /api/jobs/{id}` 查状态；失败按 `max_attempt` 重试。
- **探针**：`POST /api/pipelines/{id}/probe` — Hub 校验 + 每节点 manifest/config 探测报告。
- **兼容**：旧版 YAML `stages[]` 可 **无损转换为** 单链图（`internal/pipeline.LinearToGraph`），便于迁移。
- **校验**：success 子图无环；至少一个 `source` 节点；边端点必须存在。

---

## 3. 执行器（后端）

- `internal/core/graph_runner.go`：`GraphRunner.Run(graph)` — 拓扑排序 → 按节点类型调用 `AgentRunner`。
- 多前驱汇入 `process`：上游 `Item` 列表 **拼接** 后逐条处理（后续可扩展 merge 策略）。
- `Engine.RunGraph` 与 `Engine.Run`（线性）：线性 YAML 经 `LinearToGraph` 后走 **同一** `GraphRunner`，避免两套语义。

---

## 4. API

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/pipelines` | 列出管道摘要 |
| GET | `/api/pipelines/{id}` | 获取完整图 JSON |
| POST | `/api/pipelines` | 新建管道 |
| PUT | `/api/pipelines/{id}` | 更新图（含节点坐标） |
| DELETE | `/api/pipelines/{id}` | 删除 |
| POST | `/api/pipelines/{id}/run` | 按图同步执行 |
| POST | `/api/pipelines/{id}/run-async` | 入队异步任务 |
| POST | `/api/pipelines/{id}/probe` | 探针校验 |
| GET | `/api/jobs/{id}` | 任务状态 |

`GET /api/pipeline/status` / `POST /api/pipeline/run`：**若存在** `is_default=1` 的 DB 管道则使用其 `graph_json`；否则使用 `config.yaml` / `configs/default.yaml` 线性 YAML。

**实现状态（与代码对齐）**：上述 REST 已实现；存储 CRUD 见 `internal/storage/pipelines.go`。

---

## 5. 前端（节点编辑器）

- 技术：**React Flow**（`@xyflow/react`）+ Tailwind / shadcn 风格 UI。
- 路由：**`/pipelines/:id`**（`web/src/pages/PipelineEditor.tsx`）；从管道列表点「编辑」进入。
- 左侧插件库 **拖入画布** 添加节点；节点可拖动、连线；**保存图** 写回 `PUT /api/pipelines/{id}`。
- 与后端：列表 `GET /api/pipelines`；详情 `GET /api/pipelines/{id}`；运行 `POST /api/pipelines/{id}/run`；插件配置页 `GET/POST /api/plugins/...`（详见 [WEB_UI.md](WEB_UI.md)）。

---

## 6. 与 ARC「接口层」对齐

Web 平台 = 接口层：管道管理、插件列表、节点编辑、运行与监控均 **只通过 HTTP API** 与核心交互，不绕过引擎直写业务逻辑。

---

## 7. 首次创建 DB 默认管道

表为空时，`/api/pipeline/run` 仍回退 YAML。若希望 **完全走 DB 图**，可调用：

`POST /api/pipelines`，body：`{ "name": "default", "is_default": true, "graph": { "name": "default", "nodes": [...], "edges": [...] } }`  

其中 `graph` 须通过 `Validate()`（无环、至少一个 `source`、边引用合法）。可将现有 YAML 用 CLI/工具转为链式图后再粘贴。

**后续对齐**：`tracker run` / `internal/scheduler` 仍以 YAML 路径为主；若需与 Web 默认图一致，可再增加「从 DB 默认图执行」开关（与 `config_path` 协调）。
