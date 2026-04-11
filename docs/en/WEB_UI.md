# Web UI (pipeline-first)

## Stack

- **Vite + React + TypeScript**
- **Tailwind CSS** and **shadcn-style** components (Radix primitives, `class-variance-authority`, `lucide-react`)
- **React Flow (`@xyflow/react`)**: drag nodes on the canvas, wire a DAG; drag plugins from the left palette onto the canvas

## Routes

| Path | Purpose |
|------|---------|
| `/` | Dashboard (stats and shortcuts) |
| `/pipelines` | Pipeline list: edit, async run, probe, job lookup |
| `/pipelines/:id` | Pipeline editor (draggable DAG + side config) |
| `/pipeline` | Default pipeline status (YAML vs DB) |
| `/sources`, `/items`, `/summaries`, `/interests`, `/plugins` | Data and plugins (top bar) |
| `/plugins/:pluginId` | Generic plugin config: dynamic manifest form or JSON; presets in `localStorage`, merged when a node is created from the palette |
| `*` (anything else) | Redirects to `/pipelines` |

## Plugin configuration UI

1. **Dedicated page**: From the plugin list, open **Configure** → `/plugins/{id}`. `PluginConfigPanel` uses a custom component if `registerPluginUI` was used; otherwise `config_schema` drives `DynamicManifestForm`; otherwise a JSON object editor (`GenericJSONConfig`).
2. **Presets**: **Save preset** writes to browser `localStorage` under `tracker_plugin_presets_v1`. Dragging that plugin from the palette into the graph merges the preset into the new node `config`.
3. **Custom forms**: In `web/src/plugin-ui/register.ts`, call `registerPluginUI('plugin_id', YourComponent)` with `PluginConfigProps`.

Example: `rss` registers `RssPluginConfig` (multi-line feed URLs).

## API mapping

- `POST /api/plugins/{id}/test`, body: `{"config":{...}}`. If the plugin implements `ConfigTester` (e.g. Feishu webhook), success returns `{ "ok": true, "supported": true }`; failure returns `ok: false` and an `error` string. If not implemented, HTTP **501** with `supported: false`.
- `GET /api/stats` includes `storage_enabled` (SQLite opened). When `false`, pipeline list is usually empty and `POST /api/pipelines` cannot persist.
- `GET /api/pipelines`, `GET /api/pipelines/{id}`
- `PUT /api/pipelines/{id}` saves `graph` (nodes with `position` / `config`, edges may include `on_condition`)
- `POST /api/pipelines/{id}/run`, `run-async`, `probe`, `rerun?from=&to=`
- `GET /api/jobs/{id}`
- `GET /api/plugins/{id}/manifest`
- `GET /api/core/ping`
- `GET /api/plugins` includes `runtime` (`builtin` | `remote`) and `healthy` for remote plugins
- `GET /api/feature-flags/snapshot` returns only `expose_to_web: true` flags; the shell may show release / experiment badges based on this snapshot
- `POST /api/plugins/external/reload` — hot-reload external plugins from `TRACKER_EXTERNAL_PLUGINS_FILE` (requires `TRACKER_PLUGIN_ADMIN_KEY` + header `X-Tracker-Plugin-Admin-Key`)
- `POST /api/operator/chat` — LLM operator (`llm_operator`); requires `TRACKER_OPERATOR_API_KEY` and header `X-Tracker-Operator-Key`

## Local frontend dev

```bash
cd web && npm install && npm run dev
```

Requires Node.js and npm.

Auth environment variables:

- Backend: `TRACKER_API_KEY`
- Frontend: `VITE_TRACKER_API_KEY` (must match backend key)

Recommended setup:

```bash
cp .env.example .env
cp web/.env.example web/.env.local
```

## Deployment

See [DEPLOYMENT.md](DEPLOYMENT.md) in this folder. Chinese: [../zh/DEPLOYMENT.md](../zh/DEPLOYMENT.md).
