# 插件契约（Manifest 与 PluginHub）

与 [PLUGIN_ECOSYSTEM.md](PLUGIN_ECOSYSTEM.md)、[PIPELINE_GRAPH.md](PIPELINE_GRAPH.md) 一致。

## 1. Manifest

每个内置插件在 `internal/plugins/registry/manifests.go` 注册 `plugin.Manifest`。

主要字段：`id`（与 Name 一致）、`version`、`kind`（source、processor、summary、interest、dispatch）、`config_schema`（保存管道时校验节点 config）、`input_schema` 与 `output_schema`（预留）、`input_formats` 与 `output_formats`（格式令牌，用于边 IO 校验）、可选 `pipeline_io`（DAG 语义覆盖）、可选 `compatible_with`（仅允许列出的上游插件 id 连入）、可选 `validator_ref`。

插件可实现 `plugin.ManifestProvider`；内置插件当前以集中清单为准。

## 2. 格式令牌与边校验

- 边表示 **item 数据流**：下游须 **接受** 上游数据（`accepts_items`，默认除 `source` 外为 true）；上游须 **向下游产出** 数据流（`emits_items`，默认 `source` 与 processor/summary/interest 为 true，`dispatch` 为 false）；上游还须 **允许出边**（默认与是否产出一致，投递类不能再接后续节点）。
- **格式**：若下游 `input_formats` 非空且不全为 `"*"`，则上游 `output_formats` 须与之兼容；上游无输出格式而下游有约束时 **校验失败**。
- `input_formats` 为空：不做格式级限制（仍受接受/产出规则约束）。
- 含 `"*"`：接受任意上游输出格式。

### `pipeline_io`（可选）

| 字段 | 含义 |
|------|------|
| `emits_items` | 覆盖是否向下游传递 item 流 |
| `accepts_items` | 覆盖是否从上游接收 item 流 |
| `allow_outbound_edges` | 覆盖是否允许从此节点连出边 |
| `allow_no_incoming` | 为 true 时，**非 source** 节点可无入边（管道启动时即参与执行的引导类节点） |

写入前 `Hub.ValidatePipelineGraph` 会校验图。前端编辑器根据 `GET /api/plugins` 返回的 `emits_items`、`accepts_items`、`allow_outbound_edges`、格式与 `compatible_with` 在连线时预校验。

## 3. PluginHub 生命周期

`internal/pluginhub.Hub` 在 GraphRunner 中发出 PipelineStart、PipelineEnd、NodeStart、NodeEnd、ErrorEvent。可 Subscribe 做告警与日志。

## 4. RuntimeContext

含 Ctx、RunID、PipelineID、NodeID、JobID、Core、Payload。Payload 会合并进各 source 节点的 config（如 time_from、time_to）。

## 5. API

`GET /api/plugins/{id}/manifest` 返回 JSON 清单。

## 6. ConfigTester（配置连通性测试）

插件可选实现 `plugin.ConfigTester`，方法为 `TestConfig(ctx context.Context, cfg plugin.Config) error`。

HTTP 入口：`POST /api/plugins/{id}/test`，JSON 体中的 `config` 字段为对象，会传给 `TestConfig`。行为约定：已实现接口时 HTTP 200，成功则 `ok` 与 `supported` 均为真；探测失败则 `ok` 为假并附 `error` 字符串。未实现接口时 HTTP 501，且 `supported` 为假。

参考：`plugins/feishu` 使用 `webhook_url` 向飞书自定义机器人发送一条最小文本消息以验证可达性。
