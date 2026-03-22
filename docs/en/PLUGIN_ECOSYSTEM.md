# Plugin ecosystem and capability model

Aligned with [ARC.md](../../ARC.md) and [PLAN.md](../../PLAN.md): **all capabilities ship as plugins**, including built-ins.

## 1. Core principles

| Principle | Description |
|-----------|-------------|
| **Plugins first** | Business and integrations do not live in `internal/core`; core orchestrates registry and runners. |
| **Built-ins are plugins** | RSS, cleaning, summaries, interests, channels are separate packages behind one **registry**; the engine must not import concrete `plugins/*` implementations. |
| **Capability contract** | Plugins implement `plugin.Plugin` and expose `SourceCapability`, `ProcessCapability`, `DispatchCapability`, or `OperatorCapability` (`internal/plugin/capability.go`, `operator.go`). |
| **Extensible** | Third-party plugins extend the same registration and lifecycle via `TRACKER_EXTERNAL_PLUGINS_FILE` (subprocess + hot reload); marketplace remains on the roadmap. |

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
| `operator` | `OperatorCapability` | LLM-driven control (`llm_operator`); **not** a valid pipeline node kind |

## 4. Relationship to the pipeline graph

Pipelines are **DAGs**; each node binds `plugin_id`, `plugin_type`, and config JSON. The runner schedules in topological order; data flows on edges ([PIPELINE_GRAPH.md](PIPELINE_GRAPH.md)).

## 5. Roadmap (PLAN Phase 8)

- **Now**: built-in plugins, registry, graph persistence, APIs.
- **Next**: `tracker plugin install`, remote packages, signatures, community index.
- **Interests feeding filters**: extend interests / subscription storage + `RuntimeContext.Payload` or shared config.

## 6. External / hot-loaded plugins (subprocess)

Tracker can load **additional** plugins at process start from a YAML file pointed to by `TRACKER_EXTERNAL_PLUGINS_FILE`. Each entry runs as a **subprocess** that listens on loopback TCP, prints `{"listen_addr":"host:port"}` on the first line of **stdout**, then speaks a **length-prefixed JSON** protocol (4-byte big-endian length + UTF-8 JSON frame). Logical operations match `internal/pluginruntime/ops.go`: `handshake`, `health`, `ready`, `init`, `execute_source`, `execute_process`, `execute_dispatch`, `test_config`.

- **Host**: `internal/pluginruntime` — registers shims in `PluginManager` + manifests in `PluginHub`; supports `Engine.ReloadExternalPlugins` via `POST /api/plugins/external/reload` when `TRACKER_PLUGIN_ADMIN_KEY` is set (header `X-Tracker-Plugin-Admin-Key`).
- **Go SDK**: `sdk/plugin-go/server` and example `sdk/plugin-go/examples/echo_plugin`.
- **TypeScript (Deno)**: `sdk/plugin-ts/main.ts` — same framing; run with `deno run -A main.ts`.
- **Rust**: `sdk/plugin-rust/` — `serde_json` + `TcpStream` (no tonic required for the default wire format).
- **Proto note**: `proto/tracker/plugin/v1/plugin.proto` documents a compact `Call` RPC shape; the shipped wire format is TCP+JSON so plugins work without `protoc`.

### LLM operator (built-in)

`llm_operator` uses `internal/operator.Bridge` (DB + engine only, no HTTP loop). Chat HTTP: `POST /api/operator/chat` with header `X-Tracker-Operator-Key` matching `TRACKER_OPERATOR_API_KEY` (required). See [WEB_UI.md](WEB_UI.md) / API lists for related routes.
