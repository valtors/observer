# observer

[![CI](https://github.com/valtors/observer/actions/workflows/ci.yml/badge.svg)](https://github.com/valtors/observer/actions/workflows/ci.yml)
[![License: MIT](https://img.shields.io/badge/license-MIT-0F172A?style=flat-square)](./LICENSE)
[![Go](https://img.shields.io/badge/go-1.23+-00ADD8?style=flat-square)](https://go.dev/)
[![tests](https://img.shields.io/badge/tests-70-green?style=flat-square)]()

your agent makes 30 tool calls. you see the final text response. do you know which tools were called? what arguments were passed? which calls failed? how long they took? you don't.

observer fixes this. it sits between your mcp client (claude desktop, cline, goose, codex, whatever) and the actual mcp server. it logs every tool call to sqlite. then it exposes trace tools so your agent can query its own call history.

[landing](https://valtors.github.io/observer/) - [github](https://github.com/valtors/observer)

## why

1665 discussions across 8 agent communities. the #1 problem: nobody can see what their agent is doing. people debug agent behavior with console.log. in 2026.

observer fixes this.

## why not just X

| | console.log | langfuse | observer |
|---|---|---|---|
| setup | manual | docker + account | one binary |
| protocol awareness | no | no | MCP |
| agent self-query | no | no | trace.* tools |
| runs offline | yes | no | yes |
| cost | free | freemium | free |
| data location | stdout | their cloud | your sqlite |

console.log is debug logging from 1995. langfuse traces everything but needs a cloud account. observer is built for MCP: it understands tool calls, injects trace tools back into the agent, and keeps everything local.

## features

- **transparent proxy** - drop-in replacement for any mcp server. your client doesn't know it's talking to a proxy.
- **tool call logging** - every call logged. input. output. duration. error. token estimate. all of it.
- **trace tools** - 4 mcp tools injected into your agent's tool list:
  - `trace.history` - recent tool calls
  - `trace.stats` - usage statistics, per-tool breakdown
  - `trace.search` - search through call history
  - `trace.replay` - replay a previous tool call by id
- **tool filtering** - hide tools from the client. less tokens, less confusion. set OBSERVER_FILTER and they're gone.
- **sse transport** - run observer as an http server with /sse and /message endpoints. for remote setups.
- **sqlite storage** - all data local. no external dependencies. your data doesn't leave your machine.
- **zero config** - one binary. one env var. that's it.

## quick start

```bash
# install
go install github.com/valtors/observer@latest

# run (wrap any mcp server)
OBSERVER_TARGET="npx -y @modelcontextprotocol/server-filesystem /tmp" observer
```

or with claude desktop:

```json
{
  "mcpServers": {
    "filesystem": {
      "command": "observer",
      "env": {
        "OBSERVER_TARGET": "npx -y @modelcontextprotocol/server-filesystem /tmp"
      }
    }
  }
}
```

or with cline / goose / any mcp client - just replace the server command with `observer` and set `OBSERVER_TARGET` to the original command.

## configuration

all configuration is via environment variables:

- `OBSERVER_TARGET` - command to run the upstream mcp server (required)
- `OBSERVER_DB_PATH` - sqlite database path (default: `~/.observer/trace.db`)
- `OBSERVER_LOG_LEVEL` - log level: debug, info, warn, error (default: `info`)
- `OBSERVER_MAX_TOOLS` - max tools to expose to client (0 = all, default: `0`)
- `OBSERVER_FILTER` - comma-separated tool names to hide (default: none)
- `OBSERVER_RAW_PAYLOAD` - set to 1 to include raw input/output in trace responses (default: `0`)
- `OBSERVER_REDACT_PATTERNS` - comma-separated patterns to redact before storing (default: none)

## how it works

```
mcp client (claude, cline, etc.)
    |
    | json-rpc over stdio
    |
observer (this proxy)
    |-- logs every tool call to sqlite
    |-- injects trace.* tools into tools/list response
    |-- optionally filters tools to reduce token overhead
    |
    | json-rpc over stdio
    |
upstream mcp server (filesystem, git, etc.)
```

observer speaks the mcp protocol on both sides. it intercepts `initialize`, `tools/list`, and `tools/call` to add logging and trace tools. all other requests are passed through transparently.

## trace tools

observer injects 4 extra tools into the `tools/list` response. these are handled locally and never forwarded to the upstream server.

by default, trace tools return **metadata only** (tool name, timestamp, duration, error status, sha-256 hash of input/output). this prevents secrets or prompt injection from old tool results from leaking back into the model's context. set `OBSERVER_RAW_PAYLOAD=1` to include raw input/output in trace responses.

trace tools are **session-scoped by default** - they only return calls from the current observer session. pass an explicit `session_id` to query a different session.

### trace.history

```json
{"name": "trace.history", "arguments": {"limit": 10}}
```

returns the last n tool calls for the current session with metadata, duration, and timestamp.

### trace.stats

```json
{"name": "trace.stats", "arguments": {}}
```

returns total calls, unique tools, error count, average duration, and per-tool breakdown for the current session.

### trace.search

```json
{"name": "trace.search", "arguments": {"query": "filesystem", "limit": 20}}
```

search through tool call history for the current session by tool name, input, or output content.

### trace.replay

```json
{"name": "trace.replay", "arguments": {"call_id": 42}}
```

retrieve a previous tool call by its id for comparison or debugging.

## tests

70 tests. 63.3% coverage. all pass.

```bash
go test ./... -count=1
```

## contributing

see [CONTRIBUTING.md](CONTRIBUTING.md). we welcome contributions of all kinds - bug fixes, new trace tools, filtering strategies, transport support, docs.

good first issues are labeled `good first issue`. we have an ai agent contribution guide for contributors using ai coding tools.

## license

MIT
