# Tracker

**[Simplified Chinese README → README.zh.md](README.zh.md)**

Tracker is an open-source AI-powered information intelligence platform.

It collects information from multiple sources, processes it through an extensible pipeline, summarizes insights using LLMs, and delivers personalized digests to users.

Tracker is designed as a **plugin-first platform**, enabling a future ecosystem of official and community plugins.

The long-term goal is to build a **Personal Intelligence Engine** that helps users understand trends, news, and important signals automatically.

---

## Core Features

Multi-source Information Collection

Tracker can collect information from:

RSS feeds  
Web APIs  
News websites  
Social media  
Custom data providers

New sources can be added via plugins.

---

LLM-Powered Summarization

Content is summarized using large language models to extract key insights.

Example result:

Title: OpenAI releases new model

Summary:
OpenAI released a new generation model with improved reasoning and multimodal capability.

Key points:
- Faster inference
- Improved reasoning
- Lower operating cost

---

Interest Filtering

Users can define topics they care about.

Example configuration:

interest:
- ai
- startup
- open source

Tracker filters and ranks information based on relevance.

---

Automated Dispatch

Summarized results can be sent through:

Email  
Telegram  
Slack  
Webhook  
Dashboard

Dispatch frequency can be configured.

---

Plugin Architecture

Tracker is built around a modular plugin system.

Plugin types include:

source plugins  
processor plugins  
summary plugins  
interest plugins  
dispatch plugins  
agent plugins

Plugins can be official or third-party.

Future versions will include a **plugin marketplace**.

---

## Architecture Overview

Information Sources
↓
Source Plugins
↓
Processing Pipeline
↓
LLM Summary
↓
Interest Engine
↓
Dispatch System
↓
User

---

## Quick Start

**Build**

```bash
make build
# or
go build -o tracker ./cmd/tracker
```

On **macOS** (especially 15+), if you see `dyld: missing LC_UUID load command` when running the binary, use the system linker:

```bash
go build -ldflags="-linkmode=external" -o tracker ./cmd/tracker
```

(Requires Xcode Command Line Tools: `xcode-select --install`.)

**Run the pipeline**

```bash
# Use default config (configs/default.yaml)
./tracker run

# Or copy and edit config, then run
cp configs/default.yaml config.yaml
./tracker run -c config.yaml
```

Optional environment variables (or set in `config.yaml`):

- `OPENAI_API_KEY` – for AI summary (optional; without it, content is truncated only)
- `TELEGRAM_BOT_TOKEN` and `TELEGRAM_CHAT_ID` – for Telegram dispatch (optional; skipped if unset)
- `TRACKER_API_KEY` – required auth key for pipeline-related Web APIs (frontend must use matching `VITE_TRACKER_API_KEY`)

**Other commands**

- `./tracker pipeline list` – list pipeline stages from config
- `./tracker plugin list` – list built-in plugins
- `./tracker plugin install [name]` – show built-in plugins (marketplace placeholder)

**Configuration (fully configurable)**

- App config: `tracker.yaml` or `config.yaml`, with optional environment overrides.
- Keys: `app.db_path`, `app.pipeline_path`, `app.schedule` (cron), `app.serve_port`, `app.env` (secrets, etc.).
- Environment: `TRACKER_DB_PATH`, `TRACKER_PIPELINE_PATH`, `TRACKER_SCHEDULE`, `TRACKER_PORT`, `TRACKER_CONFIG`.
- Examples: `configs/tracker.example.yaml`, `configs/pipeline.example.yaml`.

**Scheduled pipeline runs**

Set `app.schedule` in `tracker.yaml` (cron, e.g. `"0 */2 * * *"` every 2 hours), then run `./tracker serve`. The pipeline runs on that schedule and writes to the DB.

**Plugins**

- Sources: `rss` (custom feeds), `news` (presets: global / cn; `presets`, `extra_feeds`).
- Dispatch: `telegram`, `feishu` (Lark webhook), `discord` (Discord webhook). Set `webhook_url` or the matching env vars.

**Web UI (Vite + React + Tailwind / shadcn-style)**

```bash
make build-all
# or: make web-build && make build
./tracker serve
```

See [docs/en/WEB_UI.md](docs/en/WEB_UI.md). Documentation index: [docs/README.md](docs/README.md). Open http://localhost:8080 in a browser.

- Pipeline list and **DAG editor** (drag nodes, edges, save graph)
- Plugin config pages (`/plugins/:id`) and **connection test** (`POST /api/plugins/{id}/test`, e.g. Feishu webhook)
- Sources, items, summaries, interests, dashboard

Use `-p` for port and `-db` for SQLite path (default `tracker.db`). If the frontend is not built, `./tracker serve` shows a hint to run `make web-build` first.

Recommended auth env setup:

```bash
# Backend
cp .env.example .env
# edit .env and set TRACKER_API_KEY

# Frontend local dev
cp web/.env.example web/.env.local
# edit web/.env.local and set VITE_TRACKER_API_KEY (same value)
```

If you deploy with Docker Compose, set both `TRACKER_API_KEY` and `VITE_TRACKER_API_KEY` in the project root `.env` file (frontend key is injected at build time, backend key at runtime).

**Docker**

```bash
make docker-build
docker compose up
```

Deployment and multi-instance notes: [docs/en/DEPLOYMENT.md](docs/en/DEPLOYMENT.md).

**Tests**

```bash
go test ./...
```

On **macOS**, if you hit `dyld: missing LC_UUID` when running tests, use external linking:

```bash
make test
# or
./test.sh
# or
go test -ldflags="-linkmode=external" ./...
```

Tests cover config, pipeline, plugin, model, core, storage, scheduler, and plugins (e.g. clean, keyword_interest). Some tests expect `configs/default.yaml` at the **repository root**; run `go test` from the project root.

---

## Example Pipeline

rss-source
↓
content-clean
↓
ai-summary
↓
interest-filter
↓
telegram-dispatch

---

## Roadmap

Plugin marketplace  
Trend detection  
Topic clustering  
Knowledge graph generation  
Web dashboard  
Advanced analytics

---

## License

MIT
