# Plugin ecosystem and capability model

Aligned with [ARC.md](../../ARC.md) and [PLAN.md](../../PLAN.md): **all capabilities ship as plugins**, including built-ins.

## 1. Core principles

| Principle | Description |
|-----------|-------------|
| **Plugins first** | Business and integrations do not live in `internal/core`; core orchestrates registry and runners. |
| **Built-ins are plugins** | RSS, cleaning, summaries, interests, channels are separate packages behind one **registry**; the engine must not import concrete `plugins/*` implementations. |
| **Capability contract** | Plugins implement `plugin.Plugin` and expose `SourceCapability`, `ProcessCapability`, or `DispatchCapability` (`internal/plugin/capability.go`). |
| **Extensible** | Third-party plugins and a marketplace extend the same registration and lifecycle (dynamic loading later). |

## 2. Layout and registration

- **Implementations**: `plugins/<name>/`.
- **Registry**: `internal/plugins/registry/registry.go` — `RegisterAll(reg PluginRegistrar)` aggregates built-ins (`PluginRegistrar` avoids `registry` ↔ `core` cycles); `internal/core/engine.go` **only** calls `registry.RegisterAllWithHub`, **no** direct imports of `plugins/*`.
- **Discovery**: `internal/plugin/discovery.go` lists built-in IDs.
- **Contract and Hub**: [PLUGIN_CONTRACT.md](PLUGIN_CONTRACT.md); code in `internal/pluginhub`, `internal/plugins/registry/manifests.go`.

## 3. Plugin kinds (same as pipeline node kinds)

| Kind | Capability | Typical use |
|------|------------|-------------|
| `source` | `SourceCapability` | RSS, news, API fetch |
| `processor` | `ProcessCapability` | Normalize, clean |
| `summary` | `ProcessCapability` | LLM summary |
| `interest` | `ProcessCapability` | Interest scoring |
| `dispatch` | `DispatchCapability` | Telegram, Feishu, Discord, … |

## 4. Relationship to the pipeline graph

Pipelines are **DAGs**; each node binds `plugin_id`, `plugin_type`, and config JSON. The runner schedules in topological order; data flows on edges ([PIPELINE_GRAPH.md](PIPELINE_GRAPH.md)).

## 5. Roadmap (PLAN Phase 8)

- **Now**: built-in plugins, registry, graph persistence, APIs.
- **Next**: `tracker plugin install`, remote packages, signatures, community index.
- **Interests feeding filters**: extend interests / subscription storage + `RuntimeContext.Payload` or shared config.
