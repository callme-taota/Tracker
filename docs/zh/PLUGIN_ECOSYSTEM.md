# 插件生态与能力模型

与根目录 [ARC.md](../../ARC.md)、[PLAN.md](../../PLAN.md) 一致：内置能力也必须以插件形态出现。

## 1. 核心原则

| 原则 | 说明 |
|------|------|
| **插件优先** | 业务与集成能力不写在 `internal/core`；核心负责编排、注册表、执行器。 |
| **内置即插件** | RSS、清洗、摘要、兴趣、各渠道推送等均为独立插件包，经**统一注册表**挂载；引擎不直接 `import` 各 `plugins/*` 实现。 |
| **能力契约** | 插件实现 `plugin.Plugin` 并暴露 `SourceCapability` / `ProcessCapability` / `DispatchCapability` 之一（见 `internal/plugin/capability.go`）。 |
| **可扩展** | 第三方插件与市场在同一注册与生命周期模型上扩展（动态加载为后续阶段）。 |

## 2. 目录与注册

- **实现**：`plugins/<name>/`。
- **注册**：`internal/plugins/registry/registry.go` — `RegisterAll(reg PluginRegistrar)` 聚合全部内置插件（`PluginRegistrar` 避免 `registry` 与 `core` 循环依赖）；`internal/core/engine.go` **仅**调用 `registry.RegisterAllWithHub`，**禁止**再直接 import 各 `plugins/*`。
- **发现**：`internal/plugin/discovery.go` 提供内置 ID 列表。
- **契约与 Hub**：见 [PLUGIN_CONTRACT.md](PLUGIN_CONTRACT.md)；实现位于 `internal/pluginhub`、`internal/plugins/registry/manifests.go`。

## 3. 插件类型（与管道节点类型一致）

| 类型 | 能力接口 | 典型用途 |
|------|----------|----------|
| `source` | `SourceCapability` | RSS、新闻、API 拉取 |
| `processor` | `ProcessCapability` | 清洗、归一化 |
| `summary` | `ProcessCapability` | LLM 摘要 |
| `interest` | `ProcessCapability` | 兴趣打分 |
| `dispatch` | `DispatchCapability` | Telegram、飞书、Discord 等 |

## 4. 与管道图的关系

管道为 **DAG**；每节点绑定 `plugin_id`、`plugin_type` 与配置 JSON。执行器按拓扑调度，数据沿边流动（见 [PIPELINE_GRAPH.md](PIPELINE_GRAPH.md)）。

## 5. 路线图（与 PLAN Phase 8）

- **当前**：内置插件 + 注册表 + 图状管道持久化 + API。
- **后续**：`tracker plugin install`、远程包、签名校验、社区目录。
- **兴趣反向影响筛选**：可扩展 interests / 订阅表 + `RuntimeContext.Payload` 或共享配置。
