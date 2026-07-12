package proxy

import (
	"bufio"
	"bytes"
	"database/sql"
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/valtors/observer/internal/store"
)

const testCanary = "CANARY_NOT_A_REAL_SECRET_7f61"

func TestObserverEchoHelper(t *testing.T) {
	if os.Getenv("GO_WANT_OBSERVER_ECHO") != "1" {
		return
	}

	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		var req JSONRPCRequest
		if err := json.Unmarshal(scanner.Bytes(), &req); err != nil {
			continue
		}

		var result any
		switch req.Method {
		case "tools/call":
			result = map[string]any{"content": []map[string]string{{"type": "text", "text": "echo response"}}}
		default:
			result = map[string]any{}
		}

		response := map[string]any{"jsonrpc": "2.0", "id": json.RawMessage(req.ID), "result": result}
		_ = json.NewEncoder(os.Stdout).Encode(response)
	}
	os.Exit(0)
}

func TestCanaryHiddenByDefaultAndVisibleWithRawOptIn(t *testing.T) {
	p, db := newSecurityTestProxy(t, nil)
	callEcho(t, p, testCanary)

	defaultSearch := callTrace(t, p, "trace.search", map[string]any{"query": testCanary})
	if strings.Contains(defaultSearch, testCanary) {
		t.Fatalf("default trace.search disclosed canary: %s", defaultSearch)
	}
	if !strings.Contains(defaultSearch, "input_hash") {
		t.Fatalf("default trace.search did not return metadata hash: %s", defaultSearch)
	}

	defaultHistory := callTrace(t, p, "trace.history", map[string]any{"limit": 10})
	if strings.Contains(defaultHistory, testCanary) {
		t.Fatalf("default trace.history disclosed canary: %s", defaultHistory)
	}

	p.config.RawPayload = true
	rawSearch := callTrace(t, p, "trace.search", map[string]any{"query": testCanary})
	if !strings.Contains(rawSearch, testCanary) {
		t.Fatalf("raw opt-in trace.search did not return canary: %s", rawSearch)
	}

	assertStoredInput(t, db, testCanary)
}

func TestDatabaseCreatedWithOwnerOnlyPermissions(t *testing.T) {
	dbPath := t.TempDir() + "/trace.db"
	db, err := store.Open(dbPath)
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	info, err := os.Stat(dbPath)
	if err != nil {
		t.Fatalf("stat database: %v", err)
	}
	if got := info.Mode().Perm(); got != 0o600 {
		t.Fatalf("database permissions = %04o, want 0600", got)
	}
}

func TestTraceHistoryIsIsolatedBySessionByDefault(t *testing.T) {
	p, _ := newSecurityTestProxy(t, nil)
	p.sessionID = "session-one"
	callEcho(t, p, testCanary)

	secondSession := &Proxy{
		config:    &Config{},
		db:        p.db,
		sessionID: "session-two",
		toolCache: make(map[string]ToolDef),
	}
	history := callTrace(t, secondSession, "trace.history", map[string]any{"limit": 10})
	if strings.Contains(history, testCanary) || strings.Contains(history, "session-one") {
		t.Fatalf("second session saw first session history: %s", history)
	}
	if strings.TrimSpace(history) != "[]" {
		t.Fatalf("second session history = %s, want []", history)
	}
}

func TestRedactionOccursBeforePersistence(t *testing.T) {
	p, db := newSecurityTestProxy(t, []string{testCanary})
	callEcho(t, p, testCanary)

	var input string
	if err := db.QueryRow(`SELECT input FROM tool_calls ORDER BY id DESC LIMIT 1`).Scan(&input); err != nil {
		t.Fatalf("query persisted input: %v", err)
	}
	if strings.Contains(input, testCanary) {
		t.Fatalf("persisted input contains unredacted canary: %s", input)
	}
	if !strings.Contains(input, "[REDACTED]") {
		t.Fatalf("persisted input does not contain redaction marker: %s", input)
	}
}

func newSecurityTestProxy(t *testing.T, redactPatterns []string) (*Proxy, *sql.DB) {
	t.Helper()
	t.Setenv("GO_WANT_OBSERVER_ECHO", "1")

	dbPath := t.TempDir() + "/trace.db"
	db, err := store.Open(dbPath)
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	if err := store.Migrate(db); err != nil {
		_ = db.Close()
		t.Fatalf("migrate database: %v", err)
	}

	executable, err := os.Executable()
	if err != nil {
		_ = db.Close()
		t.Fatalf("resolve test executable: %v", err)
	}
	p, err := New(&Config{
		Target:         executable + " -test.run=^TestObserverEchoHelper$",
		RedactPatterns: redactPatterns,
	}, db)
	if err != nil {
		_ = db.Close()
		t.Fatalf("start proxy: %v", err)
	}
	t.Cleanup(func() {
		if p.cmd != nil && p.cmd.Process != nil {
			_ = p.cmd.Process.Kill()
			_ = p.cmd.Wait()
		}
		_ = db.Close()
	})
	return p, db
}

func callEcho(t *testing.T, p *Proxy, message string) {
	t.Helper()
	params, err := json.Marshal(map[string]any{
		"name":      "echo",
		"arguments": map[string]string{"message": message},
	})
	if err != nil {
		t.Fatalf("marshal echo params: %v", err)
	}
	req := &JSONRPCRequest{JSONRPC: "2.0", ID: json.RawMessage(`1`), Method: "tools/call", Params: params}
	var output bytes.Buffer
	p.handleToolCall(bufio.NewWriter(&output), req, nil)
	if strings.Contains(output.String(), `"error"`) && !strings.Contains(output.String(), `"error":null`) {
		t.Fatalf("echo call failed: %s", output.String())
	}
}

func callTrace(t *testing.T, p *Proxy, name string, arguments map[string]any) string {
	t.Helper()
	args, err := json.Marshal(arguments)
	if err != nil {
		t.Fatalf("marshal trace arguments: %v", err)
	}
	req := &JSONRPCRequest{JSONRPC: "2.0", ID: json.RawMessage(`2`), Method: "tools/call"}
	var output bytes.Buffer
	p.handleTraceTool(bufio.NewWriter(&output), req, name, args)

	var response JSONRPCResponse
	if err := json.Unmarshal(output.Bytes(), &response); err != nil {
		t.Fatalf("decode trace response %q: %v", output.String(), err)
	}
	if response.Error != nil {
		t.Fatalf("trace call failed: %s", response.Error.Message)
	}
	var result struct {
		Content []struct {
			Text string `json:"text"`
		} `json:"content"`
	}
	if err := json.Unmarshal(response.Result, &result); err != nil {
		t.Fatalf("decode trace result %q: %v", string(response.Result), err)
	}
	if len(result.Content) != 1 {
		t.Fatalf("trace content count = %d, want 1", len(result.Content))
	}
	return result.Content[0].Text
}

func assertStoredInput(t *testing.T, db *sql.DB, want string) {
	t.Helper()
	var input string
	if err := db.QueryRow(`SELECT input FROM tool_calls ORDER BY id DESC LIMIT 1`).Scan(&input); err != nil {
		t.Fatalf("query persisted input: %v", err)
	}
	if !strings.Contains(input, want) {
		t.Fatalf("persisted input %q does not contain %q", input, want)
	}
}
