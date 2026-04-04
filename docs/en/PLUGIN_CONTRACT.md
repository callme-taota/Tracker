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
| `pipeline_io` | Optional DAG semantics (see below); omitted fields are inferred from `kind` and formats |
| `compatible_with` | If non-empty, only listed upstream plugin ids may connect to this node |
| `validator_ref` | Optional named validator on the Hub |

Plugins may implement `plugin.ManifestProvider`; built-ins use the central manifest table.

## 2. Format tokens and edge validation

- Default article payload: `tracker.item.v1` (maps to `model.Item`; see model for `extra`).
- **Item flow**: Edges represent item streams. Downstream must **accept** items (`accepts_items`, inferred: all kinds except `source`). Upstream must **emit** items (`emits_items`, inferred: `source` and `processor`/`summary`/`interest`; `dispatch` does not emit). Upstream must **allow outbound edges** (inferred: same as emits; sinks cannot chain further).
- **Formats**: If downstream `input_formats` is non-empty and not only `"*"`, upstream must declare compatible `output_formats`; empty upstream output with a constrained downstream **fails** (strict pipeline edges).
- Empty `input_formats`: no format restriction (still subject to accept/emit rules).
- Contains `"*"`: accept any upstream output format.

### `pipeline_io` (optional JSON object)

| Field | Meaning |
|-------|---------|
| `emits_items` | Override inferred “produces items for downstream”. |
| `accepts_items` | Override inferred “consumes items from upstream”. |
| `allow_outbound_edges` | Override whether edges **from** this node are allowed. |
| `allow_no_incoming` | If true, a **non-source** node may have zero incoming success edges (bootstrap / self-scheduled side-effect). |

`PUT` and `POST /api/pipelines` call `Hub.ValidatePipelineGraph` before persist. The pipeline editor uses `GET /api/plugins` fields `emits_items`, `accepts_items`, `allow_outbound_edges`, formats, and `compatible_with` for live connection checks.

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
