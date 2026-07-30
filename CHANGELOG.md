# Changelog

## v0.1.0 - 2026-07-30

### Added
- MCP proxy that intercepts and logs all tool calls between agents and MCP servers
- Full trace history with searchable SQLite-backed storage
- Token overhead reduction via response summarization
- OBSERVER_RAW_PAYLOAD env var for debugging with full payloads
- Metadata-only trace mode by default (secrets redacted)
- Database permissions locked to 0600
- 84 tests, 68.7% coverage
- CI pipeline
- ARCHITECTURE.md
- SECURITY.md, CODEOWNERS, issue/PR templates
- GitHub Discussions enabled
