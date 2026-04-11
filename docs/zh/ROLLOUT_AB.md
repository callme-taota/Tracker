# AB 灰度与发布流程

## 范围

Tracker 现在采用两层 rollout：

1. 同一构建内的代码路径 AB
2. 跨环境或实例 ring 的发布灰度

这两层必须使用同一套 flag key 和同一套回滚语义。

## 标准放量流程

1. 定义开关并保留 legacy 路径。
2. 默认仍走旧逻辑。
3. 先用请求级 override 做本地或受控验证。
4. 在一个 ring 或少量 subject 上做 canary。
5. 观察运行健康度、任务成功率、pipeline 输出与 Web/API 回归。
6. 逐步提高权重。
7. 只有在新路径稳定后，才允许切默认值。
8. 在清理窗口结束后再删除旧逻辑和 flag。

## 发布元数据

建议统一使用：

- `release.channel`：如 `stable`、`staging`、`canary`
- `release.ring`：如 `global`、`ring-1`、`ring-2`
- `release.version`：发布标识，便于审计和排查
- `release.instance`：稳定的进程实例标识

## 回滚顺序

优先使用以下方式：

1. 通过 `TRACKER_FLAG_OVERRIDES` 强制回到旧变体
2. 将新变体权重降到 0
3. 将流量切回稳定 ring
4. 如仍不满足，再回滚到上一构建版本

## 禁止事项

- 未经过完整观察窗口，不得提前删除旧路径
- 纯运行时开关不得直接暴露给前端
- 发布级灰度必须有可执行回滚命令，不能只写原则
- 不得复用已退役 flag key 去承载无关功能
