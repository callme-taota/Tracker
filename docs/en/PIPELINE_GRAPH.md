# Pipeline graph (dynamic config and node editor)

This document describes the **dynamically configured** pipeline model and how it maps to a **node-based editor** (similar in spirit to node editors in DCC tools or blueprint graphs).

## 1. Product goals

| Goal | Description |
|------|-------------|
| **Dynamic config** | Pipeline definitions persist (SQLite + API); not YAML-only + restart; multiple pipelines and a default pointer. |
| **Graph** | A pipeline is a **DAG** of nodes and edges; a linear `stages` list is a special case. |
| **Visual editing** | Drag nodes and edges in the browser; save writes the full graph (including `position`). |
| **Execution** | **Topological order**; `source` emits items, `process`-like nodes handle merged upstream items, `dispatch` consumes items. |

## 2. Data model (JSON / DB)

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

- **`on_condition`**: `on_success` or empty — part of the main DAG topology and data flow; `on_failure` — reserved conditional edge (**not** used today for `GraphRunner` merge or topo; only validated); non-`source` nodes need at least one **success** incoming edge.
- **Persistence**: table `pipeline_definitions` (`name`, `graph_json`, `is_default`, timestamps); `is_default=1` is preferred for `GET/POST /api/pipeline/*`, else YAML fallback.
- **Jobs**: `jobs` — `POST /api/pipelines/{id}/run-async` enqueues; `internal/worker` polls in `serve`; `GET /api/jobs/{id}`; retries via `max_attempt`.
- **Probe**: `POST /api/pipelines/{id}/probe` — Hub validation plus per-node manifest/config checks.
- **Compat**: legacy YAML `stages[]` converts via `internal/pipeline.LinearToGraph`.
- **Validation**: acyclic success subgraph; at least one `source`; edge endpoints must exist.

## 3. Executor (backend)

- `internal/core/graph_runner.go`: `GraphRunner.Run` — topo sort, then `AgentRunner` per node kind.
- Multiple predecessors into `process`: upstream items are **concatenated** then processed (merge policies can evolve).
- `Engine.RunGraph` and linear `Engine.Run` share the same `GraphRunner` semantics (linear is converted to a chain graph).

## 4. REST API

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/pipelines` | List pipeline summaries |
| GET | `/api/pipelines/{id}` | Full graph JSON |
| POST | `/api/pipelines` | Create pipeline |
| PUT | `/api/pipelines/{id}` | Update graph (including positions) |
| DELETE | `/api/pipelines/{id}` | Delete |
| POST | `/api/pipelines/{id}/run` | Run synchronously |
| POST | `/api/pipelines/{id}/run-async` | Enqueue async job |
| POST | `/api/pipelines/{id}/probe` | Probe |
| POST | `/api/pipelines/{id}/rerun` | Rerun slice (query params `from` / `to`) |
| GET | `/api/jobs/{id}` | Job status |

`GET /api/pipeline/status` and `POST /api/pipeline/run`: if a DB row has `is_default=1`, its `graph_json` is used; otherwise `config.yaml` / `configs/default.yaml` linear YAML.

See `internal/storage/pipelines.go` and `internal/api/server.go`.

## 5. Frontend (node editor)

- Stack: **React Flow** (`@xyflow/react`) + Tailwind / shadcn-style UI.
- Route: **`/pipelines/:id`** (`web/src/pages/PipelineEditor.tsx`).
- Palette drag-create; **Save** → `PUT /api/pipelines/{id}`.
- UI ↔ API details: [WEB_UI.md](WEB_UI.md).

## 6. Alignment with ARC (“interface layer”)

The web tier is the interface: pipelines, plugins, editing, runs, and monitoring go through **HTTP APIs** only.

## 7. Bootstrapping a DB-backed default pipeline

With an empty table, `/api/pipeline/run` still falls back to YAML. To run fully from DB, call:

`POST /api/pipelines` with `is_default: true` and a valid `graph`.

`tracker run` / `internal/scheduler` remain YAML-first; a single switch to “execute DB default graph” is a possible follow-up.
