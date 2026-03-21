# Tracker 插件生态与能力模型

本文档与 [ARC.md](../ARC.md)、[PLAN.md](../PLAN.md) 对齐，描述**完整插件生态**与**内置能力也必须以插件形态出现**的原则。

---

## 1. 核心原则

| 原则 | 说明 |
|------|------|
| **插件优先** | 业务与集成能力**不得**写在 `internal/core`；核心只做编排、注册表、执行器。 |
| **内置即插件** | RSS、清洗、摘要、兴趣、各渠道推送等**全部**为独立插件包，通过**统一注册表**挂载；引擎不 `import` 具体插件实现。 |
| **能力契约** | 插件实现 `plugin.Plugin` 并暴露 `SourceCapability` / `ProcessCapability` / `DispatchCapability` 之一（见 `internal/plugin/capability.go`）。 |
| **可扩展** | 未来第三方插件与「市场」在**同一注册与生命周期模型**上扩展（动态加载为后续阶段）。 |

---

## 2. 目录与注册

- **插件实现**：`plugins/<name>/`（内置官方插件）。
- **统一注册**：`internal/plugins/registry/registry.go` — 唯一聚合点，通过 `RegisterAll(reg PluginRegistrar)` 注册**全部**内置插件（`PluginRegistrar` 接口避免 `registry` ↔ `core` 包循环依赖）；`internal/core/engine.go` **仅**调用 `registry.RegisterAll(pm)`，**禁止**再直接 import 各 `plugins/*` 包。
- **发现与元数据**：`internal/plugin/discovery.go` 提供内置 ID 列表；未来可扩展为扫描 `plugin.yaml` + 二进制/so。
- **契约与 Hub**：Manifest、格式令牌、边上 IO 校验、生命周期与 `RuntimeContext` 见 [PLUGIN_CONTRACT.md](./PLUGIN_CONTRACT.md)；实现位于 `internal/pluginhub`、`internal/plugins/registry/manifests.go`。

---

## 3. 插件类型（与管道节点类型一致）

| Type | 能力接口 | 典型用途 |
|------|----------|----------|
| `source` | `SourceCapability` | RSS、新闻聚合、API 拉取 |
| `processor` | `ProcessCapability` | 清洗、归一化 |
| `summary` | `ProcessCapability` | LLM 摘要 |
| `interest` | `ProcessCapability` | 兴趣打分 |
| `dispatch` | `DispatchCapability` | Telegram、飞书、Discord 等 |

---

## 4. 与管道图的关系

管道由**有向无环图（DAG）**描述；图中**每个节点**绑定一个 `plugin_id` + `plugin_type` + 配置 JSON。执行器按拓扑序调度，数据在边上流动（见 [PIPELINE_GRAPH.md](./PIPELINE_GRAPH.md)）。

---

## 5. 路线图（与 PLAN Phase 8 对齐）

- **当前**：内置插件 + 注册表 + 图状管道持久化 + API。
- **后续**：`tracker plugin install`、远程包、签名校验、社区插件目录。
- **兴趣订阅反向影响筛选器**：可通过扩展 `interests` / 专用订阅表 + `RuntimeContext.Payload` 或共享配置，让抓取/筛选插件读取用户订阅关键词（与 `keyword_interest` 演进方向一致）。
