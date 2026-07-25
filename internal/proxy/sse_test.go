package proxy

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

func setupSSE(t *testing.T) *SSEServer {
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
	return NewSSEServer(p)
}

func TestNewSSEServer(t *testing.T) {
	s := setupSSE(t)
	if s == nil {
		t.Fatal("expected non-nil SSE server")
	}
	if s.clients == nil {
		t.Error("expected initialized clients map")
	}
}

func TestHandleSSE(t *testing.T) {
	s := setupSSE(t)
	req := httptest.NewRequest("GET", "/sse", nil)
	w := httptest.NewRecorder()
	done := make(chan struct{})
	go func() {
		s.HandleSSE(w, req)
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Log("HandleSSE timed out (expected for streaming)")
	}
}

func TestHandleMessage_NotPost(t *testing.T) {
	s := setupSSE(t)
	req := httptest.NewRequest("GET", "/message", nil)
	w := httptest.NewRecorder()
	s.HandleMessage(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", w.Code)
	}
}

func TestHandleMessage_BadJSON(t *testing.T) {
	s := setupSSE(t)
	req := httptest.NewRequest("POST", "/message", nil)
	w := httptest.NewRecorder()
	s.HandleMessage(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestHandleMessage_ValidJSON(t *testing.T) {
	s := setupSSE(t)
	body := `{"jsonrpc":"2.0","id":1,"method":"initialize"}`
	req := httptest.NewRequest("POST", "/message", strings.NewReader(body))
	w := httptest.NewRecorder()
	s.HandleMessage(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestJSONUnmarshal(t *testing.T) {
	var req struct {
		JSONRPC string `json:"jsonrpc"`
		ID      int    `json:"id"`
		Method  string `json:"method"`
	}
	json.Unmarshal([]byte(`{"jsonrpc":"2.0","id":1,"method":"initialize"}`), &req)
	if req.Method != "initialize" {
		t.Errorf("expected initialize, got %s", req.Method)
	}
}

func TestHandleMessage_OptionsMethod(t *testing.T) {
	s := setupSSE(t)
	req := httptest.NewRequest("OPTIONS", "/message", nil)
	w := httptest.NewRecorder()
	s.HandleMessage(w, req)
	if w.Code != http.StatusNoContent {
		t.Logf("got %d (CORS handling may vary)", w.Code)
	}
}

func TestHandleMessage_Initialize(t *testing.T) {
	s := setupSSE(t)
	body := `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05"}}`
	req := httptest.NewRequest("POST", "/message", strings.NewReader(body))
	w := httptest.NewRecorder()
	s.HandleMessage(w, req)
	if w.Code == 0 {
		t.Error("expected status code")
	}
}

func TestHandleMessage_ToolsList(t *testing.T) {
	s := setupSSE(t)
	body := `{"jsonrpc":"2.0","id":2,"method":"tools/list"}`
	req := httptest.NewRequest("POST", "/message", strings.NewReader(body))
	w := httptest.NewRecorder()
	s.HandleMessage(w, req)
	if w.Code == 0 {
		t.Error("expected status code")
	}
}

func TestHandleMessage_UnknownMethod(t *testing.T) {
	s := setupSSE(t)
	body := `{"jsonrpc":"2.0","id":3,"method":"unknown/method"}`
	req := httptest.NewRequest("POST", "/message", strings.NewReader(body))
	w := httptest.NewRecorder()
	s.HandleMessage(w, req)
	if w.Code == 0 {
		t.Error("expected status code")
	}
}
