# Tracker Architecture

Tracker is an AI-powered information intelligence platform designed to collect, process, analyze, and deliver insights from multiple information sources.

The system is designed with a **plugin-first architecture** and a modular pipeline engine.

The goal is to allow both official and third-party developers to extend the system through plugins.

---

## High Level Architecture

Tracker is composed of five main layers.

Interface Layer
Core Engine
Pipeline System
Plugin Runtime
Storage Layer

System overview:

Information Sources
↓
Source Plugins
↓
Pipeline Engine
↓
Agent Capabilities
↓
Dispatch System
↓
User

---

## Core Design Principles

Tracker follows several core design principles.

Plugin-first architecture  
Core engine remains minimal  
Capabilities implemented through plugins  
Pipeline-based execution model  
Extensible agent system

The core engine is responsible for orchestration rather than functionality.

---

## Core Components

The system is composed of the following major components.

Core Engine

Responsible for:

system initialization  
plugin loading  
pipeline execution  
agent orchestration

The core engine does not implement business logic.

---

Pipeline Engine

The pipeline engine defines how data flows through the system.

Pipelines are composed of ordered processing stages.

Example pipeline:

source
→ process
→ summary
→ interest
→ dispatch

Each stage receives data and outputs processed data.

---

Agent System

Agents are execution units that perform tasks in the pipeline.

Examples:

source agent  
processing agent  
summary agent  
interest agent  
dispatch agent

Agents are implemented through plugin capabilities.

---

Plugin System

The plugin system enables the platform to be extended.

Plugins provide capabilities that the pipeline engine can use.

Plugin types include:

source plugin  
processor plugin  
summary plugin  
interest plugin  
dispatch plugin  
agent plugin

Plugins can be official or third-party.

---

Plugin Manager

The plugin manager is responsible for:

plugin discovery  
plugin registration  
plugin lifecycle management  
capability injection

Plugins register themselves with the manager during initialization.

---

Capability Model

Each plugin exposes one or more capabilities.

Capabilities represent executable units used by agents.

Example capabilities:

fetch source data  
clean content  
generate summary  
calculate interest score  
dispatch notification

Capabilities are invoked by the pipeline engine.

---

Data Flow

The system processes information as structured data objects.

Typical flow:

collect raw content
↓
clean and normalize
↓
generate AI summary
↓
calculate relevance
↓
dispatch results

Data objects typically contain fields such as:

title  
content  
summary  
url  
timestamp  
metadata

---

Plugin Architecture

Plugins are self-contained modules that extend the system.

A plugin usually contains:

plugin metadata  
capability implementation  
configuration

Example plugin structure:

plugin.yaml
binary
config

Example plugin metadata:

name: tracker-source-rss
version: 1.0
type: source
author: tracker

---

Plugin Loading

Plugins are loaded at runtime.

Loading process:

scan plugin directory  
load plugin metadata  
register plugin capabilities  
activate plugin

This allows new functionality to be added without modifying the core system.

---

Official Plugins

Official plugins are maintained by the core project.

Examples:

rss source plugin  
openai summary plugin  
telegram dispatch plugin

Official plugins provide baseline functionality.

---

Community Plugins

Community plugins are developed by third-party contributors.

Examples:

twitter source plugin  
reddit source plugin  
custom llm provider plugin

These plugins expand the ecosystem.

---

Plugin Marketplace

Future versions will include a plugin marketplace.

Marketplace features:

plugin discovery  
plugin installation  
plugin updates  
plugin ratings

Plugins may be installed using commands such as:

tracker plugin install rss

---

Storage Layer

Tracker stores processed data and configuration.

For MVP, SQLite is sufficient.

Typical tables:

sources  
items  
summaries  
interests  
pipeline_definitions (DAG 管道 JSON，支持默认管道指针)

Future versions may include:

vector database for embeddings  
redis for caching  
analytics storage

---

Web Platform

The web platform provides a graphical interface for interacting with Tracker.

Features include:

pipeline management  
plugin management  
source configuration  
interest configuration  
analytics dashboard

The web platform communicates with the core engine via API.

**Aligned implementation (see `docs/`):**

- **Plugin ecosystem**: All built-in capabilities are **plugins** only; the core engine registers them through a single **registry** (`internal/plugins/registry`). **Manifest + PluginHub** (IO format tokens, lifecycle, `RuntimeContext`) — [docs/PLUGIN_CONTRACT.md](docs/PLUGIN_CONTRACT.md). See also [docs/PLUGIN_ECOSYSTEM.md](docs/PLUGIN_ECOSYSTEM.md).
- **Pipeline as a graph**: Pipelines are **DAGs** of nodes and edges (not only linear lists); persisted dynamically via storage + API; the Web UI provides a **draggable node editor** (React Flow), similar in spirit to node-based tools (e.g. color nodes, blueprint-style graphs). See [docs/PIPELINE_GRAPH.md](docs/PIPELINE_GRAPH.md).
- **Core services**: Shared **LLM router** (multi-profile), **storage router** (SQLite core DB + DSN/TCP checks for MySQL/Mongo/Redis/Kafka), **markdown** + **SSE** helpers; exposed on `RuntimeContext.Core`. See [docs/CORE_SERVICES.md](docs/CORE_SERVICES.md), `GET /api/core/ping`.
- **Web UI**: Pipeline-first pages and API mapping — [docs/WEB_UI.md](docs/WEB_UI.md).
- **部署 / 多实例**: Docker、任务抢占、连接池与扩展路径 — [docs/DEPLOYMENT.md](docs/DEPLOYMENT.md)。
- **Linear YAML** remains supported as an import/compat path: linear `stages[]` is converted to a chain graph for execution.

---

CLI Interface

Tracker also provides a command line interface.

Typical commands:

tracker run
tracker pipeline list
tracker plugin list
tracker plugin install

The CLI allows automation and scripting.

---

Future Architecture Extensions

Several architectural extensions are planned.

Trend detection engine  
Topic clustering system  
Semantic knowledge graph  
Agent-based automation

These systems will build on top of the existing plugin and pipeline infrastructure.

---

Architecture Summary

Tracker is designed as a platform rather than a single application.

Core engine
+
Plugin ecosystem (all capabilities as plugins; unified registry)
+
Pipeline execution model (graph/DAG + linear compat)
+
Agent capabilities
+
Web platform (API + node-based pipeline editor)

This architecture allows Tracker to evolve into a full **AI information intelligence platform**.

**Canonical detail docs:** [docs/PLUGIN_ECOSYSTEM.md](docs/PLUGIN_ECOSYSTEM.md), [docs/PIPELINE_GRAPH.md](docs/PIPELINE_GRAPH.md).