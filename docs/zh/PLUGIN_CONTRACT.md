# 插件契约（Manifest 与 PluginHub）

与 [PLUGIN_ECOSYSTEM.md](PLUGIN_ECOSYSTEM.md)、[PIPELINE_GRAPH.md](PIPELINE_GRAPH.md) 一致。

## 1. Manifest

每个内置插件在 `internal/plugins/registry/manifests.go` 注册 `plugin.Manifest`。

主要字段：`id`（与 Name 一致）、`version`、`kind`（source、processor、summary、interest、dispatch）、`config_schema`（保存管道时校验节点 config）、`input_schema` 与 `output_schema`（预留）、`input_formats` 与 `output_formats`（格式令牌，用于边 IO 校验）、可选 `validator_ref`。

插件可实现 `plugin.ManifestProvider`；内置插件当前以集中清单为准。

## 2. 格式令牌与边校验

默认文章载荷为 `tracker.item.v1`。`input_formats` 为空表示不限制上游；含星号表示接受任意上游。边上要求上游输出格式与下游输入格式有交集。

写入前 `Hub.ValidatePipelineGraph` 会校验图。

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
