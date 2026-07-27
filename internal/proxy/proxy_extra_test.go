package proxy

import (
	"bufio"
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/valtors/observer/internal/store"
)

func setupTestProxy(t *testing.T) *Proxy {
	db, err := store.Open(":memory:")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if err := store.Migrate(db); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return &Proxy{
		config:    &Config{Target: "echo test"},
		db:        db,
		sessionID: "test-session",
		toolCache: make(map[string]ToolDef),
	}
}

func TestGetTraceTools_Count(t *testing.T) {
	p := setupTestProxy(t)
	tools := p.getTraceTools()
	if len(tools) != 4 {
		t.Fatalf("expected 4 trace tools, got %d", len(tools))
	}
	expected := []string{"trace.history", "trace.stats", "trace.search", "trace.replay"}
	for i, exp := range expected {
		if tools[i].Name != exp {
			t.Fatalf("tool[%d]: expected %s, got %s", i, exp, tools[i].Name)
		}
	}
}

func TestGetTraceTools_RawPayloadNote(t *testing.T) {
	p := setupTestProxy(t)
	p.config.RawPayload = true
	tools := p.getTraceTools()
	for _, tool := range tools {
		if strings.Contains(tool.Description, "Set OBSERVER_RAW_PAYLOAD=1") {
			t.Fatal("raw payload note should not appear when RawPayload is true")
		}
	}
}

func TestGetTraceTools_MetadataNote(t *testing.T) {
	p := setupTestProxy(t)
	p.config.RawPayload = false
	tools := p.getTraceTools()
	found := false
	for _, tool := range tools {
		if strings.Contains(tool.Description, "metadata only") {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("metadata note should appear when RawPayload is false")
	}
}

func TestTraceReplay_NotFound(t *testing.T) {
	p := setupTestProxy(t)
	var buf bytes.Buffer
	w := bufio.NewWriter(&buf)
	req := &JSONRPCRequest{JSONRPC: "2.0", ID: json.RawMessage("1")}
	p.traceReplay(w, req, json.RawMessage(`{"call_id":999}`))
	w.Flush()
	out := buf.String()
	if !strings.Contains(out, "replay error") {
		t.Fatalf("expected replay error, got: %s", out)
	}
}

func TestTraceReplay_Found(t *testing.T) {
	p := setupTestProxy(t)
	call := &store.ToolCall{
		SessionID:  "test-session",
		ToolName:   "read_file",
		Input:      `{"path":"/tmp/test.txt"}`,
		Output:     `{"content":"hello"}`,
		DurationMs: 15,
	}
	err := store.InsertToolCall(p.db, call)
	if err != nil {
		t.Fatalf("insert call: %v", err)
	}
	var buf bytes.Buffer
	w := bufio.NewWriter(&buf)
	req := &JSONRPCRequest{JSONRPC: "2.0", ID: json.RawMessage("1")}
	p.traceReplay(w, req, json.RawMessage(`{"call_id":1}`))
	w.Flush()
	out := buf.String()
	if !strings.Contains(out, "read_file") {
		t.Fatalf("expected read_file in output, got: %s", out)
	}
}

func TestTraceReplay_RawPayload(t *testing.T) {
	p := setupTestProxy(t)
	p.config.RawPayload = true
	call := &store.ToolCall{
		SessionID:  "test-session",
		ToolName:   "read_file",
		Input:      `{"path":"/tmp/test.txt"}`,
		Output:     `{"content":"hello"}`,
		DurationMs: 15,
	}
	err := store.InsertToolCall(p.db, call)
	if err != nil {
		t.Fatalf("insert call: %v", err)
	}
	var buf bytes.Buffer
	w := bufio.NewWriter(&buf)
	req := &JSONRPCRequest{JSONRPC: "2.0", ID: json.RawMessage("1")}
	p.traceReplay(w, req, json.RawMessage(`{"call_id":1}`))
	w.Flush()
	out := buf.String()
	if !strings.Contains(out, "hello") {
		t.Fatalf("expected raw output content in result, got: %s", out)
	}
}

func TestTraceHistory_DefaultLimit(t *testing.T) {
	p := setupTestProxy(t)
	for i := 0; i < 5; i++ {
		call := &store.ToolCall{
			SessionID:  "test-session",
			ToolName:   "read_file",
			Input:      "{}",
			Output:     "{}",
			DurationMs: int64(i),
		}
		store.InsertToolCall(p.db, call)
	}
	var buf bytes.Buffer
	w := bufio.NewWriter(&buf)
	req := &JSONRPCRequest{JSONRPC: "2.0", ID: json.RawMessage("1")}
	p.traceHistory(w, req, json.RawMessage(`{}`))
	w.Flush()
	out := buf.String()
	if !strings.Contains(out, "read_file") {
		t.Fatalf("expected read_file in history, got: %s", out)
	}
}

func TestTraceStats_WithData(t *testing.T) {
	p := setupTestProxy(t)
	for i := 0; i < 3; i++ {
		call := &store.ToolCall{
			SessionID:  "test-session",
			ToolName:   "read_file",
			Input:      "{}",
			Output:     "{}",
			DurationMs: 10,
		}
		store.InsertToolCall(p.db, call)
	}
	call := &store.ToolCall{
		SessionID:  "test-session",
		ToolName:   "write_file",
		Input:      "{}",
		Output:     "permission denied",
		DurationMs: 20,
		IsError:    true,
	}
	store.InsertToolCall(p.db, call)

	var buf bytes.Buffer
	w := bufio.NewWriter(&buf)
	req := &JSONRPCRequest{JSONRPC: "2.0", ID: json.RawMessage("1")}
	p.traceStats(w, req, json.RawMessage(`{}`))
	w.Flush()
	out := buf.String()
	if !strings.Contains(out, "total") {
		t.Fatalf("expected stats output, got: %s", out)
	}
	if !strings.Contains(out, "write_file") {
		t.Fatalf("expected write_file in stats, got: %s", out)
	}
}

func TestTraceSearch_WithQuery(t *testing.T) {
	p := setupTestProxy(t)
	call := &store.ToolCall{
		SessionID:  "test-session",
		ToolName:   "search_files",
		Input:      `{"pattern":"*.go"}`,
		Output:     `{"results":["main.go"]}`,
		DurationMs: 5,
	}
	store.InsertToolCall(p.db, call)

	var buf bytes.Buffer
	w := bufio.NewWriter(&buf)
	req := &JSONRPCRequest{JSONRPC: "2.0", ID: json.RawMessage("1")}
	p.traceSearch(w, req, json.RawMessage(`{"query":"search"}`))
	w.Flush()
	out := buf.String()
	if !strings.Contains(out, "search_files") {
		t.Fatalf("expected search_files in results, got: %s", out)
	}
}

func TestTraceSearch_NoMatch(t *testing.T) {
	p := setupTestProxy(t)
	var buf bytes.Buffer
	w := bufio.NewWriter(&buf)
	req := &JSONRPCRequest{JSONRPC: "2.0", ID: json.RawMessage("1")}
	p.traceSearch(w, req, json.RawMessage(`{"query":"nonexistent"}`))
	w.Flush()
	out := buf.String()
	if !strings.Contains(out, "[]") && !strings.Contains(out, "calls") {
		t.Fatalf("expected empty results, got: %s", out)
	}
}

func TestHandleTraceTool_Dispatch(t *testing.T) {
	p := setupTestProxy(t)
	var buf bytes.Buffer
	w := bufio.NewWriter(&buf)
	req := &JSONRPCRequest{JSONRPC: "2.0", ID: json.RawMessage("1")}
	p.handleTraceTool(w, req, "trace.history", json.RawMessage(`{}`))
	w.Flush()
	if !strings.Contains(buf.String(), "result") {
		t.Fatalf("expected trace.history result, got: %s", buf.String())
	}
}

func TestMakeToolResult(t *testing.T) {
	resp := makeToolResult(json.RawMessage("1"), "hello world")
	if resp.JSONRPC != "2.0" {
		t.Fatalf("expected jsonrpc 2.0, got %s", resp.JSONRPC)
	}
	if string(resp.ID) != "1" {
		t.Fatalf("expected id 1, got %s", resp.ID)
	}
	if !bytes.Contains(resp.Result, []byte("hello world")) {
		t.Fatalf("expected result to contain text, got %s", resp.Result)
	}
}

func TestTraceReplayHTTP_NotFound(t *testing.T) {
	p := setupTestProxy(t)
	req := &JSONRPCRequest{JSONRPC: "2.0", ID: json.RawMessage("1")}
	resp := p.traceReplayHTTP(req, json.RawMessage(`{"call_id":999}`))
	if resp.Error == nil {
		t.Fatal("expected error for non-existent call")
	}
}

func TestTraceReplayHTTP_Found(t *testing.T) {
	p := setupTestProxy(t)
	call := &store.ToolCall{
		SessionID:  "test-session",
		ToolName:   "read_file",
		Input:      `{"path":"/tmp"}`,
		Output:     `{"content":"hi"}`,
		DurationMs: 10,
	}
	store.InsertToolCall(p.db, call)
	resp := p.traceReplayHTTP(&JSONRPCRequest{JSONRPC: "2.0", ID: json.RawMessage("1")}, json.RawMessage(`{"call_id":1}`))
	if resp.Error != nil {
		t.Fatalf("unexpected error: %v", resp.Error)
	}
	if !bytes.Contains(resp.Result, []byte("read_file")) {
		t.Fatalf("expected read_file in result, got %s", resp.Result)
	}
}

func TestTraceHistoryHTTP_WithData(t *testing.T) {
	p := setupTestProxy(t)
	for i := 0; i < 3; i++ {
		store.InsertToolCall(p.db, &store.ToolCall{
			SessionID:  "test-session",
			ToolName:   "read_file",
			Input:      "{}",
			Output:     "{}",
			DurationMs: int64(i),
		})
	}
	resp := p.traceHistoryHTTP(&JSONRPCRequest{JSONRPC: "2.0", ID: json.RawMessage("1")}, json.RawMessage(`{}`))
	if resp.Error != nil {
		t.Fatalf("unexpected error: %v", resp.Error)
	}
	if !bytes.Contains(resp.Result, []byte("read_file")) {
		t.Fatalf("expected read_file in result, got %s", resp.Result)
	}
}

func TestTraceSearchHTTP_WithQuery(t *testing.T) {
	p := setupTestProxy(t)
	store.InsertToolCall(p.db, &store.ToolCall{
		SessionID:  "test-session",
		ToolName:   "search_files",
		Input:      `{"pattern":"*.go"}`,
		Output:     `{"results":["main.go"]}`,
		DurationMs: 5,
	})
	resp := p.traceSearchHTTP(&JSONRPCRequest{JSONRPC: "2.0", ID: json.RawMessage("1")}, json.RawMessage(`{"query":"search"}`))
	if resp.Error != nil {
		t.Fatalf("unexpected error: %v", resp.Error)
	}
	if !bytes.Contains(resp.Result, []byte("search_files")) {
		t.Fatalf("expected search_files in result, got %s", resp.Result)
	}
}

func TestTraceStatsHTTP_WithData(t *testing.T) {
	p := setupTestProxy(t)
	store.InsertToolCall(p.db, &store.ToolCall{
		SessionID:  "test-session",
		ToolName:   "read_file",
		Input:      "{}",
		Output:     "{}",
		DurationMs: 10,
	})
	resp := p.traceStatsHTTP(&JSONRPCRequest{JSONRPC: "2.0", ID: json.RawMessage("1")}, json.RawMessage(`{}`))
	if resp.Error != nil {
		t.Fatalf("unexpected error: %v", resp.Error)
	}
	if !bytes.Contains(resp.Result, []byte("total")) {
		t.Fatalf("expected total in result, got %s", resp.Result)
	}
}

func TestHandleTraceToolHTTP_UnknownTool(t *testing.T) {
	p := setupTestProxy(t)
	resp := p.handleTraceToolHTTP(&JSONRPCRequest{JSONRPC: "2.0", ID: json.RawMessage("1")}, "trace.unknown", json.RawMessage(`{}`))
	if resp.Error == nil {
		t.Fatal("expected error for unknown trace tool")
	}
}
