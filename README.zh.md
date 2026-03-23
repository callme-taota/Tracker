# Tracker

**英文版说明见 [README.md](README.md)。**

Tracker 是开源的 AI 驱动信息情报平台：从多源采集内容，经可扩展管道处理，用 LLM 提炼要点，并向用户投递个性化摘要。

设计为**插件优先**，便于官方与社区插件生态演进；长期目标是打造帮助用户自动理解趋势与信号的**个人情报引擎**。

---

## 核心能力

**多源采集**  
RSS、Web API、新闻站、社交媒体、自定义数据源；新源可通过插件扩展。

**LLM 摘要**  
用大模型生成摘要与要点（需配置 `OPENAI_API_KEY` 等，否则可能仅做截断类处理）。

**兴趣过滤**  
按用户关心的主题过滤与排序。

**自动投递**  
邮件、Telegram、Slack、Webhook、仪表盘等；可配置频率。

**插件架构**  
数据源、处理器、摘要、兴趣、投递、Agent 等类型；未来将支持插件市场。

---

## 架构概览（数据流）

信息源 → 数据源插件 → 处理管道 → LLM 摘要 → 兴趣引擎 → 投递 → 用户

---

## 快速开始

**编译**

```bash
make build
# 或
go build -o tracker ./cmd/tracker
```

在 **macOS**（尤其 15+）若运行二进制出现 `dyld: missing LC_UUID`，请使用系统链接器：

```bash
go build -ldflags="-linkmode=external" -o tracker ./cmd/tracker
```

（需 Xcode Command Line Tools：`xcode-select --install`。）

**运行管道**

```bash
./tracker run
# 默认使用 configs/default.yaml；或复制后 -c 指定
cp configs/default.yaml config.yaml
./tracker run -c config.yaml
```

常用环境变量（也可写在 `config.yaml`）：

- `OPENAI_API_KEY`：摘要能力（可选）
- `TELEGRAM_BOT_TOKEN`、`TELEGRAM_CHAT_ID`：Telegram 推送（可选）
- `TRACKER_API_KEY`：Web 管道相关接口鉴权 key（必填，前端需提供同值 `VITE_TRACKER_API_KEY`）

**其他命令**

- `./tracker pipeline list`：列出配置中的管道阶段
- `./tracker plugin list`：列出内置插件
- `./tracker plugin install [name]`：内置列表（市场占位）

**配置（全可配置）**

- 应用配置：`tracker.yaml` 或 `config.yaml`，可用环境变量覆盖。
- 配置项：`app.db_path`、`app.pipeline_path`、`app.schedule`（cron）、`app.serve_port`、`app.env` 等。
- 环境变量：`TRACKER_DB_PATH`、`TRACKER_PIPELINE_PATH`、`TRACKER_SCHEDULE`、`TRACKER_PORT`、`TRACKER_CONFIG`。
- 示例：`configs/tracker.example.yaml`、`configs/pipeline.example.yaml`。

**管道定时执行**

在 `tracker.yaml` 中设置 `app.schedule`（cron，例如 `"0 */2 * * *"` 每 2 小时），执行 `./tracker serve` 后按计划运行并写入数据库。

**插件**

- 数据源：`rss`（自定义 feeds）、`news`（预设 global / cn，可配 `presets`、`extra_feeds`）。
- 推送：`telegram`、`feishu`（飞书 webhook）、`discord`。配置 `webhook_url` 或对应环境变量。

**Web 界面（Vite + React + Tailwind / shadcn 风格）**

```bash
make build-all
# 或 make web-build && make build
./tracker serve
```

说明见 [docs/zh/WEB_UI.md](docs/zh/WEB_UI.md)。文档总索引：[docs/README.md](docs/README.md)。浏览器访问 http://localhost:8080。

- 管道列表与 **DAG 编辑器**（拖拽节点、连线、保存图）
- 插件配置页与 **连接测试**（`POST /api/plugins/{id}/test`，如飞书 webhook）
- 数据源、条目、摘要、兴趣、概览等

`-p` 指定端口，`-db` 指定 SQLite 路径（默认 `tracker.db`）。未构建前端时，`serve` 会提示先执行 `make web-build`。

前后端鉴权变量（建议）：

```bash
# 后端
cp .env.example .env
# 编辑 .env，设置 TRACKER_API_KEY

# 前端（本地开发）
cp web/.env.example web/.env.local
# 编辑 web/.env.local，设置 VITE_TRACKER_API_KEY（需与后端一致）
```

如果使用 Docker Compose，请在项目根 `.env` 中同时设置 `TRACKER_API_KEY` 与 `VITE_TRACKER_API_KEY`（构建镜像时注入前端，运行容器时注入后端）。

**Docker**

```bash
make docker-build
docker compose up
```

部署与多实例见 [docs/zh/DEPLOYMENT.md](docs/zh/DEPLOYMENT.md)。

**测试**

```bash
go test ./...
```

在 **macOS** 若测试报 `dyld: missing LC_UUID`，请使用外部链接：

```bash
make test
# 或 ./test.sh
# 或 go test -ldflags="-linkmode=external" ./...
```

部分测试依赖仓库根目录下的 `configs/default.yaml`，请在**项目根**执行 `go test`。

---

## 示例管道（概念）

rss → 清洗 → AI 摘要 → 兴趣过滤 → Telegram 投递

---

## 路线图

插件市场、趋势检测、主题聚类、知识图谱、Web 仪表盘、高级分析等。

---

## 许可证

MIT
