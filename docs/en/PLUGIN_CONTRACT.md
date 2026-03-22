# Plugin contract (Manifest and PluginHub)

Aligned with [PLUGIN_ECOSYSTEM.md](PLUGIN_ECOSYSTEM.md) and [PIPELINE_GRAPH.md](PIPELINE_GRAPH.md).

## 1. Manifest

Each built-in plugin registers a `plugin.Manifest` in `internal/plugins/registry/manifests.go`:

| Field | Meaning |
|-------|---------|
| `id` | Same as `plugin.Name()` |
| `version` | Manifest version |
| `kind` | `source` / `processor` / `summary` / `interest` / `dispatch` |
| `config_schema` | JSON (light subset): `type: object`, `required`, etc., validates node `config` on save |
| `input_schema` / `output_schema` | Reserved full JSON Schema; docs and future validation |
| `input_formats` / `output_formats` | **Format tokens** (e.g. `tracker.item.v1`) for **edge IO checks** |
| `validator_ref` | Optional named validator on the Hub |

Plugins may implement `plugin.ManifestProvider`; built-ins use the central manifest table.

## 2. Format tokens and edge validation

- Default article payload: `tracker.item.v1` (maps to `model.Item`; see model for `extra`).
- Empty `input_formats`: no upstream restriction.
- Contains `"*"`: accept any upstream output.
- Edge rule: overlap between upstream `output_formats` and downstream `input_formats`, or relaxed rules above.

`PUT` and `POST /api/pipelines` call `Hub.ValidatePipelineGraph` before persist.

## 3. PluginHub lifecycle

`internal/pluginhub.Hub` emits from `GraphRunner` at:

- `PipelineStart` / `PipelineEnd`
- `NodeStart` / `NodeEnd`
- `ErrorEvent` (node failure or unknown type)

Subscribe with `Hub.Subscribe` for alerts, metrics, logging.

## 4. RuntimeContext

`pluginhub.RuntimeContext` includes: `Ctx`, `RunID`, `PipelineID`, `NodeID`, `JobID`, `Core`, `Payload` (async or rerun payload merged into **source** `config`, e.g. RFC3339 `time_from` / `time_to` for RSS).

## 5. API

- `GET /api/plugins/{id}/manifest`: JSON manifest for dynamic forms.

## 6. Config connectivity test (ConfigTester)

- Interface: `plugin.ConfigTester` with `TestConfig(ctx context.Context, cfg plugin.Config) error`.
- HTTP: `POST /api/plugins/{id}/test` with JSON body containing a `config` object. When implemented: success returns 200 with `ok` and `supported` true; connectivity/validation failure returns 200 with `ok` false and an `error` string; not implemented returns 501 with `supported` false.
- Reference: `plugins/feishu` posts a minimal text payload to `webhook_url`.
