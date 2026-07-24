package proxy

import (
	"database/sql"
	"encoding/json"
	"testing"
)

func TestIsHidden_Filter(t *testing.T) {
	cfg := &Config{Target: "echo", Filter: []string{"read_file", "write_file"}}
	db, _ := sql.Open("sqlite3", ":memory:")
	p, err := New(cfg, db)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if !p.isHidden("read_file") {
		t.Error("expected read_file to be hidden")
	}
	if p.isHidden("other_tool") {
		t.Error("other_tool should not be hidden")
	}
}

func TestIsHidden_NoFilter(t *testing.T) {
	cfg := &Config{Target: "echo"}
	db, _ := sql.Open("sqlite3", ":memory:")
	p, _ := New(cfg, db)
	if p.isHidden("anything") {
		t.Error("nothing should be hidden with empty filter")
	}
}

func TestGetTraceTools_Default(t *testing.T) {
	cfg := &Config{Target: "echo"}
	db, _ := sql.Open("sqlite3", ":memory:")
	p, _ := New(cfg, db)
	tools := p.getTraceTools()
	if len(tools) != 4 {
		t.Errorf("expected 4 trace tools, got %d", len(tools))
	}
	names := map[string]bool{}
	for _, tool := range tools {
		names[tool.Name] = true
	}
	for _, e := range []string{"trace.history", "trace.stats", "trace.search", "trace.replay"} {
		if !names[e] {
			t.Errorf("expected %s in trace tools", e)
		}
	}
}

func TestGetTraceTools_Raw(t *testing.T) {
	cfg := &Config{Target: "echo", RawPayload: true}
	db, _ := sql.Open("sqlite3", ":memory:")
	p, _ := New(cfg, db)
	tools := p.getTraceTools()
	if len(tools) != 4 {
		t.Errorf("expected 4, got %d", len(tools))
	}
}

func TestNewProxy_Basic(t *testing.T) {
	cfg := &Config{Target: "echo"}
	db, _ := sql.Open("sqlite3", ":memory:")
	p, err := New(cfg, db)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if p == nil {
		t.Fatal("expected non-nil proxy")
	}
}

func TestJSONRPCReq(t *testing.T) {
	req := JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      json.RawMessage("1"),
		Method:  "initialize",
	}
	if req.Method != "initialize" {
		t.Errorf("expected initialize, got %s", req.Method)
	}
}

func TestJSONRPCResp(t *testing.T) {
	resp := JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      json.RawMessage("1"),
	}
	if string(resp.ID) != "1" {
		t.Errorf("expected '1', got %s", resp.ID)
	}
}

func TestToolDef_Basic(t *testing.T) {
	tool := ToolDef{Name: "test", Description: "desc"}
	if tool.Name != "test" || tool.Description != "desc" {
		t.Error("ToolDef fields wrong")
	}
}
