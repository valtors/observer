package store

import (
	"database/sql"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

func setupTestDB(t *testing.T) *sql.DB {
	db, err := Open(":memory:")
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}
	if err := Migrate(db); err != nil {
		t.Fatalf("failed to migrate test db: %v", err)
	}
	return db
}

func TestOpenAndMigrate(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	// Migration runs in Open, so if it succeeds, migration passed.
	if db == nil {
		t.Fatal("expected db to be initialized")
	}
}

func TestInsertAndGetRecentCalls(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	// Empty DB
	calls, err := GetRecentCalls(db, 10, "")
	if err != nil {
		t.Fatalf("expected no error for empty db, got %v", err)
	}
	if len(calls) != 0 {
		t.Errorf("expected 0 calls, got %d", len(calls))
	}

	call := &ToolCall{
		SessionID:     "sess1",
		ToolName:      "testTool",
		Input:         "input",
		Output:        "output",
		IsError:       false,
		DurationMs:    100,
		TokenEstimate: 10,
	}

	err = InsertToolCall(db, call)
	if err != nil {
		t.Fatalf("InsertToolCall failed: %v", err)
	}

	calls, err = GetRecentCalls(db, 10, "")
	if err != nil {
		t.Fatalf("GetRecentCalls failed: %v", err)
	}
	if len(calls) != 1 {
		t.Errorf("expected 1 call, got %d", len(calls))
	}
	if calls[0].ToolName != "testTool" {
		t.Errorf("expected toolName 'testTool', got '%s'", calls[0].ToolName)
	}

	// Test with SessionID filter
	calls, err = GetRecentCalls(db, 10, "otherSess")
	if err != nil {
		t.Fatalf("GetRecentCalls failed: %v", err)
	}
	if len(calls) != 0 {
		t.Errorf("expected 0 calls for other session, got %d", len(calls))
	}
}

func TestGetStats(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	stats, err := GetStats(db, "")
	if err != nil {
		t.Fatalf("GetStats on empty db failed: %v", err)
	}
	if stats.TotalCalls != 0 {
		t.Errorf("expected 0 total calls, got %d", stats.TotalCalls)
	}

	call1 := &ToolCall{
		SessionID:     "sess1",
		ToolName:      "toolA",
		Input:         "{}",
		Output:        "{}",
		IsError:       false,
		DurationMs:    100,
		TokenEstimate: 10,
	}
	call2 := &ToolCall{
		SessionID:     "sess1",
		ToolName:      "toolA",
		Input:         "{}",
		Output:        "{}",
		IsError:       true,
		DurationMs:    200,
		TokenEstimate: 10,
	}
	call3 := &ToolCall{
		SessionID:     "sess2",
		ToolName:      "toolB",
		Input:         "{}",
		Output:        "{}",
		IsError:       false,
		DurationMs:    300,
		TokenEstimate: 10,
	}

	InsertToolCall(db, call1)
	InsertToolCall(db, call2)
	InsertToolCall(db, call3)

	stats, err = GetStats(db, "")
	if err != nil {
		t.Fatalf("GetStats failed: %v", err)
	}
	if stats.TotalCalls != 3 {
		t.Errorf("expected 3 total calls, got %d", stats.TotalCalls)
	}
	if stats.ErrorCount != 1 {
		t.Errorf("expected 1 error count, got %d", stats.ErrorCount)
	}
	if stats.UniqueTools != 2 {
		t.Errorf("expected 2 unique tools, got %d", stats.UniqueTools)
	}
}

func TestSearchCalls(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	calls, err := SearchCalls(db, "test", "", 10)
	if err != nil {
		t.Fatalf("SearchCalls on empty db failed: %v", err)
	}
	if len(calls) != 0 {
		t.Errorf("expected 0 calls, got %d", len(calls))
	}

	InsertToolCall(db, &ToolCall{
		SessionID: "sess1",
		ToolName:  "fetch_data",
		Input:     "user=123",
		Output:    "success",
	})
	InsertToolCall(db, &ToolCall{
		SessionID: "sess2",
		ToolName:  "update_data",
		Input:     "user=456",
		Output:    "error",
	})

	calls, err = SearchCalls(db, "fetch", "", 10)
	if err != nil {
		t.Fatalf("SearchCalls failed: %v", err)
	}
	if len(calls) != 1 {
		t.Errorf("expected 1 call, got %d", len(calls))
	}
	if calls[0].ToolName != "fetch_data" {
		t.Errorf("expected fetch_data, got %s", calls[0].ToolName)
	}

	calls, err = SearchCalls(db, "user", "", 10)
	if err != nil {
		t.Fatalf("SearchCalls failed: %v", err)
	}
	if len(calls) != 2 {
		t.Errorf("expected 2 calls, got %d", len(calls))
	}
}
