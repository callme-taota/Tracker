# Tracker Agent Specification

This document defines how agents and plugins operate inside Tracker.

Tracker processes information through a pipeline of agents.

Pipeline structure:

source
→ process
→ summary
→ interest
→ dispatch

Each step is implemented as a capability provided by plugins.

---

## Data Structure

Data objects flow through the pipeline.

Fields include:

title  
content  
summary  
url  
timestamp  
metadata

Each agent receives data and returns modified data.

---

## Agent Responsibilities

Source Agent

Collects raw information from external sources.

Examples:

rss source  
api source  
web crawler

Output fields:

title  
content  
url  
timestamp  
source

---

Processing Agent

Responsible for data cleaning and normalization.

Tasks include:

remove html  
normalize text  
deduplicate  
detect language

---

Summary Agent

Generates summaries using LLMs.

Responsibilities:

extract key points  
generate concise summary  
structure information

Output fields:

summary  
key_points

---

Interest Agent

Determines relevance based on user interests.

Strategies include:

keyword match  
embedding similarity  
llm classification

Example output:

score  
matched_topics

---

Dispatch Agent

Delivers processed information to users.

Channels include:

email  
telegram  
slack  
webhook  
dashboard

---

## Plugin System

All capabilities are implemented via plugins.

Plugin types:

source plugin  
processor plugin  
summary plugin  
interest plugin  
dispatch plugin  
agent plugin

Plugins must register themselves with the plugin manager.

---

## Agent Execution Model

Agents run sequentially inside pipelines.

Execution flow:

data → source → process → summary → interest → dispatch

Agents should be:

stateless  
modular  
configurable

---

## Error Handling

Agents should return structured errors.

Example:

error: source_unreachable  
message: rss feed timeout

---

## Future Agent Types

trend detection agent  
topic clustering agent  
knowledge graph agent  
anomaly detection agent