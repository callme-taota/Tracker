# Tracker

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

**Other commands**

- `./tracker pipeline list` – list pipeline stages from config
- `./tracker plugin list` – list built-in plugins
- `./tracker plugin install [name]` – show built-in plugins (marketplace placeholder)

**配置（全可配置）**

- 应用配置：`tracker.yaml` 或 `config.yaml`，或环境变量覆盖。
- 配置项：`app.db_path`、`app.pipeline_path`、`app.schedule`（cron）、`app.serve_port`、`app.env`（密钥等）。
- 环境变量：`TRACKER_DB_PATH`、`TRACKER_PIPELINE_PATH`、`TRACKER_SCHEDULE`、`TRACKER_PORT`、`TRACKER_CONFIG`。
- 示例：`configs/tracker.example.yaml`、`configs/pipeline.example.yaml`。

**管道自动触发**

在 `tracker.yaml` 中设置 `app.schedule`（cron 表达式，如 `"0 */2 * * *"` 每 2 小时），然后执行 `./tracker serve`，管道会按计划自动运行并写入 DB。

**插件**

- 数据源：`rss`（自定义 feeds）、`news`（主流新闻预设：global / cn，可配 `presets`、`extra_feeds`）。
- 推送：`telegram`、`feishu`（飞书 webhook）、`discord`（Discord webhook）。配置 `webhook_url` 或对应环境变量。

**Web 界面（Vite + React + Tailwind / shadcn 风格）**

```bash
make build-all
# 或: make web-build && make build
./tracker serve
```

说明见 [docs/zh/WEB_UI.md](docs/zh/WEB_UI.md)（英文：[docs/en/WEB_UI.md](docs/en/WEB_UI.md)）。文档索引：[docs/README.md](docs/README.md)。浏览器打开 http://localhost:8080。主要能力：

- **管道列表 / DAG 编辑器**（拖拽节点、连线、保存图）
- **插件配置页**（`/plugins/:id`）与 **连接测试**（`POST /api/plugins/{id}/test`，如飞书 webhook）
- 数据源、条目、摘要、兴趣、概览等

使用 `-p` 指定端口，`-db` 指定 SQLite 路径（默认 `tracker.db`）。未构建前端时，`./tracker serve` 会提示先执行 `make web-build`。

**Docker**

```bash
make docker-build
docker compose up
```

部署与多实例规划见 [docs/zh/DEPLOYMENT.md](docs/zh/DEPLOYMENT.md)（英文：[docs/en/DEPLOYMENT.md](docs/en/DEPLOYMENT.md)）。

**测试**

```bash
go test ./...
```

在 **macOS** 上若出现 `dyld: missing LC_UUID`，请用外部链接跑测试：

```bash
make test
# 或
./test.sh
# 或
go test -ldflags="-linkmode=external" ./...
```

单测覆盖：config、pipeline、plugin、model、core、storage、scheduler、plugins（clean、keyword_interest）。部分测试依赖项目根目录下的 `configs/default.yaml`，建议在项目根执行。

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