package store

import (
	"testing"
)

func TestUpsertSession(t *testing.T) {
	db := setupTestDB(t)
	s := &Session{
		ID: "s1", ClientName: "claude", ClientVersion: "1.0",
		ServerName: "fs", ServerVersion: "0.1",
	}
	if err := UpsertSession(db, s); err != nil {
		t.Fatalf("UpsertSession: %v", err)
	}
	if err := UpsertSession(db, s); err != nil {
		t.Fatalf("UpsertSession (update): %v", err)
	}
}

func TestUpsertTool(t *testing.T) {
	db := setupTestDB(t)
	if err := UpsertTool(db, "read", "read a file", "{}"); err != nil {
		t.Fatalf("UpsertTool: %v", err)
	}
	if err := UpsertTool(db, "read", "read a file v2", "{}"); err != nil {
		t.Fatalf("UpsertTool (update): %v", err)
	}
}

func TestSetToolVisible(t *testing.T) {
	db := setupTestDB(t)
	if err := UpsertTool(db, "read", "desc", "{}"); err != nil {
		t.Fatalf("UpsertTool: %v", err)
	}
	if err := SetToolVisible(db, "read", false); err != nil {
		t.Fatalf("SetToolVisible false: %v", err)
	}
	if err := SetToolVisible(db, "read", true); err != nil {
		t.Fatalf("SetToolVisible true: %v", err)
	}
}

func TestGetVisibleTools(t *testing.T) {
	db := setupTestDB(t)
	if err := UpsertTool(db, "read", "desc", "{}"); err != nil {
		t.Fatalf("UpsertTool: %v", err)
	}
	if err := UpsertTool(db, "write", "desc2", "{}"); err != nil {
		t.Fatalf("UpsertTool: %v", err)
	}
	if err := SetToolVisible(db, "write", false); err != nil {
		t.Fatalf("SetToolVisible: %v", err)
	}
	tools, err := GetVisibleTools(db)
	if err != nil {
		t.Fatalf("GetVisibleTools: %v", err)
	}
	if len(tools) != 1 {
		t.Fatalf("expected 1 visible tool, got %d", len(tools))
	}
	if tools[0].Name != "read" {
		t.Errorf("expected 'read', got %s", tools[0].Name)
	}
}

func TestGetAllTools(t *testing.T) {
	db := setupTestDB(t)
	if err := UpsertTool(db, "read", "desc", "{}"); err != nil {
		t.Fatalf("UpsertTool: %v", err)
	}
	if err := UpsertTool(db, "write", "desc2", "{}"); err != nil {
		t.Fatalf("UpsertTool: %v", err)
	}
	tools, err := GetAllTools(db)
	if err != nil {
		t.Fatalf("GetAllTools: %v", err)
	}
	if len(tools) != 2 {
		t.Fatalf("expected 2 tools, got %d", len(tools))
	}
}

func TestGetCallByID_NotFound(t *testing.T) {
	db := setupTestDB(t)
	call, err := GetCallByID(db, 999)
	if call != nil {
		t.Error("expected nil call for non-existent ID")
	}
	if err == nil {
		t.Error("expected error for non-existent ID")
	}
}

func TestInsertAndGetCallByID(t *testing.T) {
	db := setupTestDB(t)
	if err := UpsertSession(db, &Session{ID: "s1"}); err != nil {
		t.Fatalf("UpsertSession: %v", err)
	}
	call := &ToolCall{
		SessionID: "s1", ToolName: "read",
		Input: "input", Output: "output",
		InputHash: "h1", OutputHash: "h2",
		IsError: false, DurationMs: 100,
	}
	if err := InsertToolCall(db, call); err != nil {
		t.Fatalf("InsertToolCall: %v", err)
	}
	calls, err := GetRecentCalls(db, 10, "")
	if err != nil || len(calls) != 1 {
		t.Fatalf("GetRecentCalls: %v, len=%d", err, len(calls))
	}
	found, err := GetCallByID(db, calls[0].ID)
	if err != nil {
		t.Fatalf("GetCallByID: %v", err)
	}
	if found == nil || found.ToolName != "read" {
		t.Error("unexpected call")
	}
}
