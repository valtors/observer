# Architecture

## Overview

Observer is a transparent MCP proxy. It sits between an MCP client (Claude, Cursor, any MCP-compatible agent) and an MCP server, intercepting every tool call, logging inputs and outputs to SQLite, and injecting four trace tools back into the client's tool list. The agent gets observability for free without changing how it works.

```
  MCP Client         Observer              MCP Server
  (Claude)          (proxy)               (filesystem)
     |                 |                       |
     |  initialize     |  initialize           |
     |---------------->|---------------------->|
     |  tools/list     |  tools/list           |
     |---------------->|---------------------->|
     |                 |  inject trace tools   |
     |<----------------|                       |
     |                 |                       |
     |  tools/call     |  intercept + log       |
     |---------------->|                       |
     |                 |  forward              |
     |                 |---------------------->|
     |                 |  result               |
     |                 |<----------------------|
     |                 |  store to SQLite      |
     |  result         |                       |
     |<----------------|                       |
```

## Design Principles

1. **Transparent proxy.** The agent does not know Observer exists. It sees the same tools, same schemas, same responses. Observer adds four trace tools on top.
2. **Local-first.** All traces live in SQLite on disk. Nothing is exported to an OTel backend unless you explicitly configure it. DB permissions are 0600; directory is 0700.
3. **Metadata by default.** Trace tools return hashes, not payloads. Raw input/output requires `OBSERVER_RAW_PAYLOAD=1`. This prevents secrets from re-entering agent context.
4. **Redaction at write time.** `OBSERVER_REDACT_PATTERNS` redacts configured patterns before persistence. The raw value never touches the database.

## Components

### proxy (`internal/proxy`)

The core interception layer. Spawns the target MCP server as a child process and bridges JSON-RPC messages bidirectionally.

**Startup:**
1. Parse `OBSERVER_TARGET` into command + args.
2. Spawn child process with stdin/stdout/stderr pipes.
3. Create `Proxy` struct with DB connection, session ID, tool cache.
4. Start forwarding messages between client and target.

**Message flow (client -> server):**
- `tools/list`: Observer intercepts the response from the target server, injects `trace.history`, `trace.stats`, `trace.replay`, `trace.search` into the tool list, and returns the augmented list to the client.
- `tools/call`: Observer forwards the call, measures duration, stores the input/output/duration/error to SQLite, then returns the original response to the client. If the tool name starts with `trace.`, Observer handles it locally (does not forward to target).
- All other JSON-RPC methods: pass through untouched.

**Tool filtering:** `OBSERVER_FILTER` hides specific tools from the client. `OBSERVER_MAX_TOOLS` caps the number of tools exposed. Both operate on the response to `tools/list`.

**Thread safety:** `sync.Mutex` protects the tool cache and stdin pipe (JSON-RPC is request/response, so only one message is in flight at a time).

### store (`internal/store`)

SQLite persistence layer. Three tables: `tool_calls`, `sessions`, `tool_registry`.

**Schema:**
```sql
CREATE TABLE tool_calls (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    session_id TEXT NOT NULL,
    tool_name TEXT NOT NULL,
    input TEXT,
    output TEXT,
    input_hash TEXT,
    output_hash TEXT,
    is_error INTEGER DEFAULT 0,
    duration_ms INTEGER DEFAULT 0,
    token_estimate INTEGER DEFAULT 0,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE sessions (
    id TEXT PRIMARY KEY,
    client_name TEXT,
    client_version TEXT,
    server_name TEXT,
    server_version TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    last_active DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE tool_registry (
    name TEXT PRIMARY KEY,
    description TEXT,
    input_schema TEXT,
    visible INTEGER DEFAULT 1,
    call_count INTEGER DEFAULT 0,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
```

**Integrity without disclosure:** Every tool call stores `input_hash` and `output_hash` (SHA-256, first 16 bytes hex-encoded). Trace tools return `ToolCallMeta` (hash, duration, error status) by default. The actual `input` and `output` fields are omitted unless `OBSERVER_RAW_PAYLOAD=1`.

**Redaction:** `Redact(text, patterns)` replaces configured patterns with `[REDACTED]` before storage. Applied to both input and output at insert time. The raw value is never persisted.

**Token estimation:** `EstimateTokens(text)` = `len(text) / 4`. Rough heuristic for cost tracking without a tokenizer dependency.

### config (`internal/proxy/config`)

Environment-based configuration. No config files, no flags. Every setting is an env var.

| Env var | Default | Purpose |
|---|---|---|
| `OBSERVER_TARGET` | required | Command to run the upstream MCP server |
| `OBSERVER_DB_PATH` | `~/.observer/trace.db` | SQLite database path |
| `OBSERVER_LOG_LEVEL` | `info` | Log verbosity |
| `OBSERVER_MAX_TOOLS` | `0` (all) | Cap number of tools exposed to client |
| `OBSERVER_FILTER` | none | Comma-separated tool names to hide |
| `OBSERVER_RAW_PAYLOAD` | `false` | Include raw input/output in trace responses |
| `OBSERVER_REDACT_PATTERNS` | none | Comma-separated strings to redact before storage |
| `OBSERVER_LISTEN_ADDR` | none | If set, start SSE server instead of stdio |

### SSE server (`internal/proxy/sse`)

HTTP transport alternative. When `OBSERVER_LISTEN_ADDR` is set, Observer starts an HTTP server with two endpoints:
- `/sse` -- Server-Sent Events stream. Pushes tool call events to connected clients in real time.
- `/message` -- Receives JSON-RPC messages from the client.

Clients are tracked in a map protected by `sync.Mutex`. Events are buffered (channel size 64) per client.

## Trace Tools

Four tools injected into the client's tool list:

| Tool | Input | Output (metadata mode) | Output (raw mode) |
|---|---|---|---|
| `trace.history` | `limit` (default 20) | `ToolCallMeta[]` -- id, tool name, hash, duration, error, timestamp | Full `ToolCall[]` with input/output |
| `trace.search` | `query`, `limit` | Matching `ToolCallMeta[]` | Matching `ToolCall[]` |
| `trace.replay` | `call_id` | `ToolCallMeta` for the specified call | Full `ToolCall` with original input/output |
| `trace.stats` | none | Per-tool call counts and error rates | Same |

The metadata-only default is a security boundary: a prompt-injection string in an old tool result cannot become active again through `trace.history` or `trace.search` because the raw payload is never returned to the agent's context.

## Process Model

Observer runs as a single process with two transport modes:
- **stdio** (default): MCP proxy over stdin/stdout. The child MCP server also communicates over stdio through Observer's pipes.
- **SSE** (`OBSERVER_LISTEN_ADDR`): HTTP server with SSE for real-time event streaming.

Signal handling: SIGINT/SIGTERM cancels the context, which triggers graceful shutdown of the proxy and SSE server.

## Data Flow

```
1. LoadConfig() reads env vars
2. store.Open(dbPath) -> SQLite DB (0600, WAL mode)
3. store.Migrate(db) -> create tables + indexes
4. proxy.New(config, db) -> spawn target MCP server as child process
5. proxy.Run(ctx):
   a. Read JSON-RPC from client stdin
   b. Forward to target server stdin
   c. Read response from target stdout
   d. If tools/list: inject trace tools
   e. If tools/call: store to DB, return result
   f. If trace.*: handle locally
   g. Write response to client stdout
6. On signal: cancel ctx, close DB, kill child process
```

## File Layout

```
~/.observer/
  trace.db          # SQLite database (mode 0600)
  trace.db-wal      # Write-ahead log
  trace.db-shm      # Shared memory
```

## Testing

84 tests, 68.7% coverage. Race detector enabled in CI. Test categories:
- Security: DB permissions, redaction, metadata-only enforcement, raw payload opt-in
- Proxy: tool injection, message forwarding, tool filtering
- Store: CRUD, hash computation, redaction, token estimation
- Integration: echo server end-to-end test (spawns a real MCP server, proxies through Observer)
- SSE: client connection, event delivery, multi-client broadcast

## Dependencies

- `github.com/mattn/go-sqlite3` -- SQLite (CGO, system SQLite)
- Go stdlib for JSON-RPC, HTTP, SSE, process management
