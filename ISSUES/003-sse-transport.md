# issue: add support for SSE transport

## Description
Currently, Observer only supports `stdio` transport (ideal for local tools like Claude Desktop or Cline). However, many production environments require `SSE` (Server-Sent Events) to allow remote MCP clients to connect to an MCP server.

## Tasks
- [ ] Implement an SSE transport handler in `internal/proxy`
- [ ] Add a new configuration option `OBSERVER_LISTEN_ADDR` (e.g., `0.0.0.0:8080`)
- [ ] Ensure SSE mode still maintains full observability and logging

## Requirements
- Use standard Go `net/http` for the SSE server
- The server must still act as a proxy, connecting to the `OBSERVER_TARGET` upstream
- Support `GET` for the event stream and `POST` for client messages

## Additional Context
This is an `enhancement`. Adding SSE will make Observer useful for remote/cloud-based agents.
