# Tracker Development Plan

Goal: build a minimal but extensible AI information pipeline platform.

The first version must be small, runnable, and architecturally clean.

---

Phase 1 — Core Framework

Implement the core engine.

Components:

plugin manager  
agent runner  
pipeline runner

The core engine should not implement business logic.

All capabilities must come from plugins.

---

Phase 2 — Plugin System

Implement plugin architecture.

Requirements:

plugin registration  
plugin discovery  
plugin capability injection  
plugin lifecycle

Plugin interface example:

Name  
Version  
Type  
Init

Plugins should expose capabilities.

---

Phase 3 — Pipeline Engine

Implement the pipeline execution system.

Pipeline should run sequential stages.

Example pipeline:

rss source
→ processor
→ summary
→ interest
→ dispatch

The pipeline runner should pass structured data between stages.

---

Phase 4 — Built-in Official Plugins

Create the first official plugins.

Required plugins:

rss source plugin  
content clean plugin  
openai summary plugin  
keyword interest plugin  
telegram dispatch plugin

These will form the default pipeline.

---

Phase 5 — CLI Interface

Create a simple CLI.

Commands:

tracker run  
tracker pipeline list  
tracker plugin list  
tracker plugin install

CLI should start pipelines and manage plugins.

---

Phase 6 — Storage

Add basic storage.

Use SQLite for MVP.

Tables:

sources  
items  
summaries  
interests

Future support:

vector database  
redis cache

---

Phase 7 — Web Interface

Create a basic web interface.

Features:

view pipeline status  
manage sources  
manage interests  
manage plugins

The web interface will communicate with the core engine via API.

**Phase 7b — Dynamic pipeline graph & node editor (aligned with ARC):**

- [x] Pipelines stored as **node graphs** (DAG) in `pipeline_definitions`; CRUD `GET/POST/PUT/DELETE /api/pipelines` + `POST /api/pipelines/{id}/run`.
- [x] Execution: **GraphRunner**; linear YAML → `LinearToGraph` → same path; `POST/GET /api/pipeline/*` prefers DB default when set.
- [x] **Built-in features remain plugins**; `registry.RegisterAll` only (core does not import `plugins/*` directly).
- [x] Web: **React Flow** editor at `/pipelines/:id`; persist positions via `PUT` graph JSON.
- Detail: [docs/en/PIPELINE_GRAPH.md](docs/en/PIPELINE_GRAPH.md), [docs/en/PLUGIN_ECOSYSTEM.md](docs/en/PLUGIN_ECOSYSTEM.md).

**Phase 7c — Jobs + probe + success/failure edges (docs aligned):**

- [x] `jobs` 表、`worker` 轮询、`run-async` / `GET /api/jobs/{id}`、`probe` API。
- [x] 边上 `on_condition`；拓扑与数据流仅使用 success 边。

---

Phase 8 — Plugin Marketplace

Introduce plugin marketplace concept.

Plugins should be installable through:

tracker plugin install <name>

Marketplace categories:

official plugins  
community plugins

---

Phase 9 — Advanced Features

After MVP stability:

deduplication  
semantic similarity filtering  
trend detection  
topic clustering

---

Architecture Principle

Core engine must stay small.

Capabilities must live in plugins.

The system should resemble a platform rather than a tool.

---

End Goal

Tracker should evolve into an open-source personal intelligence platform with:

plugin ecosystem  
agent capabilities  
web platform  
plugin marketplace