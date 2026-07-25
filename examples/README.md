# observer examples

## wrap any mcp server

```bash
# instead of running:
npx -y @modelcontextprotocol/server-filesystem /tmp

# run:
OBSERVER_TARGET="npx -y @modelcontextprotocol/server-filesystem /tmp" observer
```

your agent now sees the same tools, but every call is logged.

## claude desktop

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

## multiple servers

```json
{
  "mcpServers": {
    "filesystem": {
      "command": "observer",
      "env": {
        "OBSERVER_TARGET": "npx -y @modelcontextprotocol/server-filesystem /tmp"
      }
    },
    "git": {
      "command": "observer",
      "env": {
        "OBSERVER_TARGET": "npx -y @modelcontextprotocol/server-git /repo"
      }
    }
  }
}
```

## filter noisy tools

```bash
OBSERVER_FILTER="list_files,read_file" \
OBSERVER_TARGET="npx -y @modelcontextprotocol/server-filesystem /tmp" \
observer
```

the agent never sees `list_files` or `read_file`. fewer tokens, less confusion.

## sse transport (remote)

```bash
OBSERVER_TARGET="npx -y @modelcontextprotocol/server-filesystem /tmp" \
OBSERVER_SSE=1 \
observer -port 3000
```

connect from anywhere:
```
http://your-server:3000/sse
```

## query trace history from your agent

after running observer, your agent gets 4 new tools:

```
trace.history({"limit": 10})      # last 10 tool calls
trace.stats({})                    # usage breakdown
trace.search({"query": "delete"})  # search for "delete" in history
trace.replay({"call_id": 42})      # get details of call #42
```

the agent can now see what it did. and debug itself.

## audit log

all calls stored in `~/.observer/trace.db`:

```bash
sqlite3 ~/.observer/trace.db "SELECT tool_name, duration_ms, error FROM tool_calls ORDER BY id DESC LIMIT 20"
```
