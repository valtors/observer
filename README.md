# Observer

Transparent MCP proxy for agent observability.

Observer sits between your MCP client (Claude Desktop, Cline, Goose, Codex, etc.) and the actual MCP server. It logs every tool call to SQLite and exposes trace history through additional MCP tools, so you can see exactly what your agent is doing.

## Why

Across 1665 discussions in 8 major AI agent communities (MCP, Codex, Cline, Goose, Anthropic SDK, OpenAI Python), the #1 recurring problem is agent observability. Nobody can see what their agent is actually doing. There is no standard way to trace agent actions, tool calls, or decisions.

Observer fixes this.

## Features

- **Transparent proxy** - Drop-in replacement for any MCP server. Your client does not know it is talking to a proxy.
- **Tool call logging** - Every tool call is logged with input, output, duration, error status, and token estimate.
- **Trace tools** - Exposes 4 additional MCP tools for querying history:
  - `trace.history` - List recent tool calls
  - `trace.stats` - Usage statistics with per-tool breakdown
  - `trace.search` - Search through tool call history
  - `trace.replay` - Replay a previous tool call by ID
- **Tool filtering** - Hide tools from the client to reduce token overhead (addresses tool bloat, the #2 problem found in the same research).
- **SQLite storage** - All data stored locally in SQLite. No external dependencies.
- **Zero config** - One binary, one env var to point at your upstream server.

## Quick Start

```bash
# Install
go install github.com/valtors/observer@latest

# Run (wrap any MCP server)
OBSERVER_TARGET="npx -y @modelcontextprotocol/server-filesystem /tmp" observer
```

Or with Claude Desktop:

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

Or with Cline / Goose / any MCP client - just replace the server command with `observer` and set `OBSERVER_TARGET` to the original command.

## Configuration

All configuration is via environment variables:

| Variable | Description | Default |
|---|---|---|
| `OBSERVER_TARGET` | Command to run the upstream MCP server (required) | - |
| `OBSERVER_DB_PATH` | SQLite database path | `~/.observer/trace.db` |
| `OBSERVER_LOG_LEVEL` | Log level: debug, info, warn, error | `info` |
| `OBSERVER_MAX_TOOLS` | Max tools to expose to client (0 = all) | `0` |
| `OBSERVER_FILTER` | Comma-separated tool names to hide | - |
| `OBSERVER_RAW_PAYLOAD` | Set to 1 to include raw input/output in trace responses | `0` |
| `OBSERVER_REDACT_PATTERNS` | Comma-separated patterns to redact before storing | - |

## How It Works

```
MCP Client (Claude, Cline, etc.)
    |
    | JSON-RPC over stdio
    |
Observer (this proxy)
    |-- logs every tool call to SQLite
    |-- injects trace.* tools into tools/list response
    |-- optionally filters tools to reduce token overhead
    |
    | JSON-RPC over stdio
    |
Upstream MCP Server (filesystem, git, etc.)
```

Observer speaks the MCP protocol on both sides. It intercepts `initialize`, `tools/list`, and `tools/call` to add logging and trace tools. All other requests are passed through transparently.

## Trace Tools

Observer injects 4 extra tools into the `tools/list` response. These are handled locally and never forwarded to the upstream server.

By default, trace tools return **metadata only** (tool name, timestamp, duration, error status, SHA-256 hash of input/output). This prevents secrets or prompt injection from old tool results from leaking back into the model's context. Set `OBSERVER_RAW_PAYLOAD=1` to include raw input/output in trace responses.

Trace tools are **session-scoped by default** - they only return calls from the current Observer session. Pass an explicit `session_id` to query a different session.

### trace.history

```json
{"name": "trace.history", "arguments": {"limit": 10}}
```

Returns the last N tool calls for the current session with metadata, duration, and timestamp.

### trace.stats

```json
{"name": "trace.stats", "arguments": {}}
```

Returns total calls, unique tools, error count, average duration, and per-tool breakdown for the current session.

### trace.search

```json
{"name": "trace.search", "arguments": {"query": "filesystem", "limit": 20}}
```

Search through tool call history for the current session by tool name, input, or output content.

### trace.replay

```json
{"name": "trace.replay", "arguments": {"call_id": 42}}
```

Retrieve a previous tool call by its ID for comparison or debugging.

## Database Schema

Three tables:

- `tool_calls` - Every tool invocation with input, output, duration, error status
- `sessions` - MCP sessions with client/server info
- `tool_registry` - Discovered tools with visibility flags and call counts

All data is stored locally. Nothing leaves your machine.

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md). We welcome contributions of all kinds - bug fixes, new trace tools, filtering strategies, transport support, docs.

Good first issues are labeled `good first issue`. We have an AI agent contribution guide for contributors using AI coding tools.

## License

MIT
