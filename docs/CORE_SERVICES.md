# Core 服务（LLM / 存储 / 格式化 / SSE）

注入位置：`pluginhub.RuntimeContext.Core` 为 `*core.CoreServices`（见 [PLUGIN_CONTRACT.md](./PLUGIN_CONTRACT.md)）。

## LLM Router

包：`internal/core/llm`

- 环境变量：`OPENAI_API_KEY`（必填才有 profile）、`OPENAI_BASE_URL`（默认 `https://api.openai.com/v1`）、`OPENAI_MODEL`（默认 `gpt-4o`）、`OPENAI_MODEL_CHEAP`（默认 `gpt-4o-mini`）。
- Profile 名：`default`、`heavy`（同 default）、`cheap`。
- API：`Router.Chat(ctx, profile, messages, temperature)` — OpenAI 兼容 `POST /chat/completions`。

## 存储 Router

包：`internal/core/storage`

| 后端 | 环境变量 | 默认实现 |
|------|-----------|----------|
| SQLite（Core 专用库） | `TRACKER_CORE_SQLITE`（文件路径） | `database/sql` + `modernc.org/sqlite`，可 `Ping` / 业务 SQL |
| MySQL | `TRACKER_MYSQL_DSN` | 仅记录 DSN + **TCP 连通性**探测（不引入 `go-sql-driver` 以保持默认构建零新增依赖） |
| MongoDB | `TRACKER_MONGO_URI`、`TRACKER_MONGO_DATABASE` | URI 解析主机 + TCP 探测 |
| Redis | `TRACKER_REDIS_ADDR` | TCP 探测 |
| Kafka | `TRACKER_KAFKA_BROKERS`（逗号分隔） | 对首个 broker 做 TCP 探测 |

完整读写客户端：在部署环境执行 `go get github.com/go-sql-driver/mysql`、`go.mongodb.org/mongo-driver`、`github.com/redis/go-redis/v9`、`github.com/segmentio/kafka-go` 后，可在 `internal/core/storage/` 增加薄封装 PR（本仓库默认 `go.mod` 不强制拉取，避免离线构建失败）。

## 格式化

包：`internal/core/format` — `NormalizeMarkdown`、`StripInvisible`。

## SSE

包：`internal/core/sse` — `WriteHeaders`、`Event`、`WriteChunk`。

## HTTP

- `GET /api/core/ping` — 返回 LLM profile 列表与各存储 TCP/SQLite ping 结果。

## 降级

未配置的后端在 `Ping` 中不出现或跳过；插件使用 `Core.Storage` 前应检查字段与错误。
