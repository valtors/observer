# Observer

Your agent makes 30 tool calls. You see the final text response. Do you know which tools were called? What arguments were passed? Which calls failed? How long they took? You don't.

Observer fixes this. It sits between your MCP client (Claude Desktop, Cline, Goose, Codex, whatever) and the actual MCP server. It logs every tool call to SQLite. Then it exposes trace tools so your agent can query its own call history.

## Why

1665 discussions across 8 agent communities. The #1 problem: nobody can see what their agent is doing. People debug agent behavior with console.log. In 2026.

Observer fixes this.

## Features

- **Transparent proxy** - Drop-in replacement for any MCP server. Your client doesn't know it's talking to a proxy.
- **Tool call logging** - Every call logged. Input. Output. Duration. Error. Token estimate. All of it.
- **Trace tools** - 4 MCP tools injected into your agent's tool list:
  - `trace.history` - recent tool calls
  - `trace.stats` - usage statistics, per-tool breakdown
  - `trace.search` - search through call history
  - `trace.replay` - replay a previous tool call by ID
- **Tool filtering** - Hide tools from the client. Less tokens, less confusion. Set OBSERVER_FILTER and they're gone.
- **SSE transport** - Run Observer as an HTTP server with /sse and /message endpoints. For remote setups.
- **SQLite storage** - All data local. No external dependencies. Your data doesn't leave your machine.
- **Zero config** - One binary. One env var. That's it.

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
