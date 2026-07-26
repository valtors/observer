package store

import (
	"os"
	"testing"
)

func TestOpen_InvalidPath(t *testing.T) {
	_, err := Open("/nonexistent/path/that/does/not/exist/test.db")
	if err == nil {
		t.Fatal("expected error for invalid path")
	}
}

func TestOpen_FilePath(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "observer-test")
	defer os.RemoveAll(tmpDir)
	dbPath := tmpDir + "/test.db"
	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer db.Close()
	if err := Migrate(db); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
}

func TestGetStats_WithErrors(t *testing.T) {
	db := setupTestDB(t)
	for i := 0; i < 5; i++ {
		call := &ToolCall{
			SessionID:   "s1",
			ToolName:    "read",
			Input:       "{}",
			Output:      "{}",
			DurationMs:  int64(i * 10),
		}
		InsertToolCall(db, call)
	}
	call := &ToolCall{
		SessionID:   "s1",
		ToolName:    "write",
		Input:       "{}",
		Output:      "disk full",
		DurationMs:  50,
		IsError:     true,
	}
	InsertToolCall(db, call)
	stats, err := GetStats(db, "")
	if err != nil {
		t.Fatalf("GetStats: %v", err)
	}
	if stats.TotalCalls != 6 {
		t.Fatalf("expected 6 total calls, got %d", stats.TotalCalls)
	}
	if stats.ErrorCount != 1 {
		t.Fatalf("expected 1 error, got %d", stats.ErrorCount)
	}
}

func TestGetStats_FilterBySession(t *testing.T) {
	db := setupTestDB(t)
	call := &ToolCall{
		SessionID:   "s1",
		ToolName:    "read",
		Input:       "{}",
		Output:      "{}",
		DurationMs:  10,
	}
	InsertToolCall(db, call)
	call2 := &ToolCall{
		SessionID:   "s2",
		ToolName:    "write",
		Input:       "{}",
		Output:      "{}",
		DurationMs:  20,
	}
	InsertToolCall(db, call2)
	stats, err := GetStats(db, "s1")
	if err != nil {
		t.Fatalf("GetStats: %v", err)
	}
	if stats.TotalCalls != 1 {
		t.Fatalf("expected 1 call for s1, got %d", stats.TotalCalls)
	}
}

func TestFormatJSON_InvalidJSON(t *testing.T) {
	call := &ToolCall{
		Input:  "not valid json{",
		Output: "also not json",
	}
	result := FormatJSON(call)
	if result == "" {
		t.Fatal("FormatJSON should return non-empty string even for invalid JSON")
	}
}

func TestInsertToolCall_WithHash(t *testing.T) {
	db := setupTestDB(t)
	call := &ToolCall{
		SessionID:     "s1",
		ToolName:      "search",
		Input:         `{"query":"test"}`,
		Output:        `{"results":["a","b"]}`,
		DurationMs:    25,
		TokenEstimate: 50,
	}
	if err := InsertToolCall(db, call); err != nil {
		t.Fatalf("InsertToolCall: %v", err)
	}
	calls, err := GetRecentCalls(db, 1, "s1")
	if err != nil {
		t.Fatalf("GetRecentCalls: %v", err)
	}
	if len(calls) != 1 {
		t.Fatalf("expected 1 call, got %d", len(calls))
	}
	if calls[0].InputHash == "" {
		t.Fatal("expected non-empty input hash")
	}
	if calls[0].TokenEstimate != 50 {
		t.Fatalf("expected 50 tokens, got %d", calls[0].TokenEstimate)
	}
}
