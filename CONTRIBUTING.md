# Contributing to Observer

Thanks for your interest in contributing. Observer is a transparent MCP proxy for agent observability, built with Go and SQLite.

## Ways to Contribute

- **Bug fixes** - Check issues labeled `bug`
- **Features** - Check issues labeled `enhancement` or `good first issue`
- **Trace tools** - Add new tools under the `trace.*` namespace
- **Filtering** - Improve tool filtering strategies to reduce token overhead
- **Transports** - Add SSE, HTTP transport support (currently stdio only)
- **Docs** - Improve README, add examples, write guides
- **Tests** - Add test coverage for proxy and store packages

## Setup

```bash
git clone https://github.com/valtors/observer.git
cd observer
go mod tidy
go build .
```

## Project Structure

```
main.go                    Entry point, CLI flags
internal/
  proxy/
    config.go              Configuration from env vars
    proxy.go                MCP proxy core (JSON-RPC, tool interception, trace tools)
  store/
    db.go                   SQLite open + migrations
    queries.go              All database queries (tool calls, sessions, stats)
```

## Development

```bash
# Build
go build .

# Run with a test server
OBSERVER_TARGET="npx -y @modelcontextprotocol/server-filesystem /tmp" go run .

# Test trace tools manually
echo '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}' | go run .
```

## AI Agent Contribution Guide

If you are using an AI coding agent (Claude Code, Codex, Cline, etc.) to contribute:

1. **Read the code first** - Have your agent read `proxy.go` and `queries.go` before making changes. The codebase is small and self-contained.
2. **No comments** - We do not use code comments. The code should be self-documenting.
3. **Test your build** - Run `go build .` before committing. Ensure it compiles.
4. **Keep it minimal** - Observer is intentionally small. Do not add dependencies unless absolutely necessary.
5. **SQLite only** - All storage goes through the `store` package. Do not add other databases.
6. **MCP protocol** - If adding protocol features, reference the MCP spec at modelcontextprotocol.io.

## Pull Requests

1. Fork the repo and create a branch from `main`
2. Make your changes
3. Ensure `go build .` passes
4. If adding a new trace tool, update the README
5. Use the PR template when opening a PR
6. Short, lowercase commit messages (e.g., `fix: handle empty tools list`)

## Code of Conduct

Be respectful. Help others. Focus on the problem, not the person.
