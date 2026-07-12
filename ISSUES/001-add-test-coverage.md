# issue: add test coverage for store package

## Description
Currently, the `store` package does not have unit tests. We need to ensure that SQLite operations (inserting tool calls, retrieving history, stats, etc.) are working correctly.

## Tasks
- [ ] Create `internal/store/queries_test.go`
- [ ] Add test for `Open` and `Migrate`
- [ ] Add test for `InsertToolCall` and `GetRecentCalls`
- [ ] Add test for `GetStats`
- [ ] Add test for `SearchCalls`

## Requirements
- Use `:memory:` SQLite database for tests
- Ensure all edge cases (empty DB, single call, multiple calls) are covered
- Use `go test ./...` to verify

## Additional Context
This is a `good first issue` to help improve code quality.
