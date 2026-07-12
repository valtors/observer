# issue: implement tool filtering to reduce token overhead

## Description
Users have reported that passing hundreds of tool definitions to an agent significantly increases the context window usage and token cost (the #2 problem found in our community research).

We need to implement the `OBSERVER_FILTER` configuration, which allows users to provide a comma-separated list of tool names to *hide* from the client.

## Tasks
- [ ] Update `internal/proxy/config.go` to parse `OBSERVER_FILTER`
- [ ] Update `internal/proxy/proxy.go`'s `handleToolsList` to filter out hidden tools
- [ ] Add unit tests for the filtering logic

## Requirements
- The filter should be case-sensitive (matching the tool name)
- If the filter list is empty, all tools are visible
- If a tool name is in the filter list, it must not appear in the `tools/list` response

## Additional Context
This is a `good first issue`. Reducing token bloat is a high-priority feature for agent users.
