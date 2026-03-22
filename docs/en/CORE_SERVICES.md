# Core services (LLM, storage, formatting, SSE)

Injection point: `pluginhub.RuntimeContext.Core` is `*core.CoreServices` (see [PLUGIN_CONTRACT.md](PLUGIN_CONTRACT.md)).

## LLM router

Package: `internal/core/llm`

- Env: `OPENAI_API_KEY` (no profiles if unset), `OPENAI_BASE_URL` (default `https://api.openai.com/v1`), `OPENAI_MODEL` (default `gpt-4o`), `OPENAI_MODEL_CHEAP` (default `gpt-4o-mini`).
- Profiles: `default`, `heavy` (same as default), `cheap`.
- API: `Router.Chat(ctx, profile, messages, temperature)` — OpenAI-compatible `POST /chat/completions`.

## Storage router

Package: `internal/core/storage`

| Backend | Env | Default behavior |
|---------|-----|------------------|
| SQLite (core DB) | `TRACKER_CORE_SQLITE` (file path) | `database/sql` + `modernc.org/sqlite`, `Ping` + SQL |
| MySQL | `TRACKER_MYSQL_DSN` | DSN recorded + **TCP reachability** (no `go-sql-driver` in default build) |
| MongoDB | `TRACKER_MONGO_URI`, `TRACKER_MONGO_DATABASE` | Parse host + TCP probe |
| Redis | `TRACKER_REDIS_ADDR` | TCP probe |
| Kafka | `TRACKER_KAFKA_BROKERS` (comma-separated) | TCP probe to first broker |

Full read/write clients: add thin wrappers under `internal/core/storage/` after `go get`ing drivers in your deployment (default `go.mod` stays lean for offline builds).

## Formatting

Package: `internal/core/format` — `NormalizeMarkdown`, `StripInvisible`.

## SSE

Package: `internal/core/sse` — `WriteHeaders`, `Event`, `WriteChunk`.

## HTTP

- `GET /api/core/ping` — LLM profiles and storage probe results.

## Degradation

Unconfigured backends may be omitted from `Ping`; plugins should check `Core.Storage` fields and errors before use.
