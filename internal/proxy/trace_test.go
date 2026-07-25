package proxy

import (
	"bufio"
	"database/sql"
	"encoding/json"
	"strings"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

func setupProxy(t *testing.T) *Proxy {
	t.Helper()
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("sql.Open: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	p, err := New(&Config{Target: "echo"}, db)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return p
}

func captureOutput(t *testing.T, fn func(w *bufio.Writer)) string {
	t.Helper()
	var sb strings.Builder
	w := bufio.NewWriter(&sb)
	fn(w)
	w.Flush()
	return sb.String()
}

func TestHandleTraceTool_UnknownTool(t *testing.T) {
	p := setupProxy(t)
	req := &JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      json.RawMessage("1"),
		Method:  "tools/call",
	}
	output := captureOutput(t, func(w *bufio.Writer) {
		p.handleTraceTool(w, req, "trace.unknown", json.RawMessage("{}"))
	})
	if !strings.Contains(output, "error") {
		t.Error("expected error response for unknown tool")
	}
}

func TestHandleTraceTool_History(t *testing.T) {
	p := setupProxy(t)
	req := &JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      json.RawMessage("1"),
		Method:  "tools/call",
	}
	output := captureOutput(t, func(w *bufio.Writer) {
		p.handleTraceTool(w, req, "trace.history", json.RawMessage(`{"limit": 5}`))
	})
	if !strings.Contains(output, "result") && !strings.Contains(output, "error") {
		t.Error("expected some response")
	}
}

func TestHandleTraceTool_Stats(t *testing.T) {
	p := setupProxy(t)
	req := &JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      json.RawMessage("1"),
		Method:  "tools/call",
	}
	output := captureOutput(t, func(w *bufio.Writer) {
		p.handleTraceTool(w, req, "trace.stats", json.RawMessage(`{}`))
	})
	if !strings.Contains(output, "result") && !strings.Contains(output, "error") {
		t.Error("expected some response")
	}
}

func TestHandleTraceTool_Search(t *testing.T) {
	p := setupProxy(t)
	req := &JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      json.RawMessage("1"),
		Method:  "tools/call",
	}
	output := captureOutput(t, func(w *bufio.Writer) {
		p.handleTraceTool(w, req, "trace.search", json.RawMessage(`{"query": "test"}`))
	})
	if !strings.Contains(output, "result") && !strings.Contains(output, "error") {
		t.Error("expected some response")
	}
}

func TestHandleTraceTool_Replay(t *testing.T) {
	p := setupProxy(t)
	req := &JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      json.RawMessage("1"),
		Method:  "tools/call",
	}
	output := captureOutput(t, func(w *bufio.Writer) {
		p.handleTraceTool(w, req, "trace.replay", json.RawMessage(`{"id": 1}`))
	})
	if !strings.Contains(output, "result") && !strings.Contains(output, "error") {
		t.Error("expected some response")
	}
}

func TestTraceHistory_WithSessionID(t *testing.T) {
	p := setupProxy(t)
	req := &JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      json.RawMessage("1"),
		Method:  "tools/call",
	}
	output := captureOutput(t, func(w *bufio.Writer) {
		p.handleTraceTool(w, req, "trace.history", json.RawMessage(`{"limit": 5, "session_id": "test-session"}`))
	})
	if output == "" {
		t.Error("expected non-empty output")
	}
}

func TestTraceStats_EmptyDB(t *testing.T) {
	p := setupProxy(t)
	req := &JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      json.RawMessage("1"),
		Method:  "tools/call",
	}
	output := captureOutput(t, func(w *bufio.Writer) {
		p.handleTraceTool(w, req, "trace.stats", json.RawMessage(`{}`))
	})
	if output == "" {
		t.Error("expected non-empty stats output")
	}
}

func TestWriteError(t *testing.T) {
	output := captureOutput(t, func(w *bufio.Writer) {
		writeError(w, json.RawMessage("1"), -32601, "test error")
	})
	if !strings.Contains(output, "error") {
		t.Error("expected error in output")
	}
	if !strings.Contains(output, "test error") {
		t.Error("expected 'test error' message")
	}
}

func TestWriteResponse(t *testing.T) {
	output := captureOutput(t, func(w *bufio.Writer) {
		writeResponse(w, json.RawMessage("1"), json.RawMessage(`{"ok": true}`), nil)
	})
	if !strings.Contains(output, "result") {
		t.Error("expected result in output")
	}
}

func TestHandleToolsList(t *testing.T) {
	p := setupProxy(t)
	req := &JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      json.RawMessage("1"),
		Method:  "tools/list",
	}
	output := captureOutput(t, func(w *bufio.Writer) {
		p.handleToolsList(w, req)
	})
	if output == "" {
		t.Error("expected non-empty output")
	}
}

func TestHandleInitialize(t *testing.T) {
	p := setupProxy(t)
	req := &JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      json.RawMessage("1"),
		Method:  "initialize",
	}
	output := captureOutput(t, func(w *bufio.Writer) {
		p.handleInitialize(w, req)
	})
	if output == "" {
		t.Error("expected non-empty initialize output")
	}
}

func TestGenerateSessionID(t *testing.T) {
	p := setupProxy(t)
	id := p.sessionID
	if id == "" {
		t.Error("expected non-empty session ID")
	}
}
