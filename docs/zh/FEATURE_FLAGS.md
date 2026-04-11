# 功能开关与实验更新

## 目标

Tracker 使用 feature flag 包裹新行为，保证新逻辑上线时不会直接破坏旧路径。今后的运行时、API、前端、插件、发布链路变更，都应先挂开关，再逐步放量。

## 强制要求

1. 所有会影响用户行为或运行时语义的更新，必须先有开关或 kill-switch，不能直接替换旧逻辑。
2. 在灰度窗口结束、确认无需回滚之前，旧逻辑必须保留。
3. 每个开关至少要声明：
   - `key`
   - `description`
   - `default_variant`
   - rollout 变体与权重
   - 是否暴露给前端
4. 长生命周期开关必须有清理责任人和预期移除时间。
5. 紧急修 bug 可以使用短期 kill-switch，而不是完整 AB 实验，但仍需写入文档。

## 当前运行时模型

- 配置来源：`tracker.yaml` / `config.yaml`
- 应用级字段：
  - `app.release.channel`
  - `app.release.ring`
  - `app.release.version`
  - `app.release.instance`
  - `app.feature_flags`
- 环境变量覆盖：
  - `TRACKER_RELEASE_CHANNEL`
  - `TRACKER_RELEASE_RING`
  - `TRACKER_RELEASE_VERSION`
  - `TRACKER_RELEASE_INSTANCE`
  - `TRACKER_FLAG_OVERRIDES`

## 当前内置基线开关

- `runtime.executor_v2`
  - 控制 API、worker、scheduler、CLI 是否走统一执行链。
  - 变体：`legacy`、`unified`
- `web.runtime_experiments`
  - 控制 Web 顶栏是否显示实验/发布通道徽标。
  - 变体：`off`、`on`
- `runtime.legacy_formats_compat`
  - 给旧的宽松格式兼容逻辑保留临时护栏。

## 请求级覆盖

可通过以下方式临时指定变体：

- 请求头：`X-Tracker-Experiment: flag_a=variant1,flag_b=variant2`
- Query：`?experiment=flag_a=variant1,flag_b=variant2`

此能力仅用于调试、QA、受控金丝雀验证，不应作为最终放量方式。

## 保留运行时元数据

每个管道节点都可能收到一个保留配置字段：

- `__tracker_runtime`

其中包含 release 信息与当前命中的实验快照。已有插件必须忽略未知字段；新插件可以读取它做观测，但不应修改它。

## 前端暴露规则

只有标记了 `expose_to_web: true` 的开关，才会通过 `GET /api/feature-flags/snapshot` 返回给前端。纯运行时开关必须留在服务端。

## 清理策略

全量成功后执行：

1. 将默认值切到胜出变体。
2. 继续观察一个完整发布窗口。
3. 删除败选逻辑路径。
4. 删除开关定义。
5. 同步更新文档与 SOP。
