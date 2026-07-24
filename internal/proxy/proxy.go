package proxy

import (
	"bufio"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/valtors/observer/internal/store"
)

type Proxy struct {
	config    *Config
	db        *sql.DB
	cmd       *exec.Cmd
	stdin     io.WriteCloser
	stdout    io.ReadCloser
	stderr    io.ReadCloser
	sessionID string
	mu        sync.Mutex
	toolCache map[string]ToolDef
}

type ToolDef struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	InputSchema json.RawMessage `json:"inputSchema"`
}

func New(config *Config, db *sql.DB) (*Proxy, error) {
	parts := strings.Fields(config.Target)
	if len(parts) == 0 {
		return nil, fmt.Errorf("empty target command")
	}

	cmd := exec.Command(parts[0], parts[1:]...)

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("stdin pipe: %w", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("stdout pipe: %w", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("start target: %w", err)
	}

	return &Proxy{
		config:    config,
		db:        db,
		cmd:       cmd,
		stdin:     stdin,
		stdout:    stdout,
		stderr:    stderr,
		sessionID: generateSessionID(),
		toolCache: make(map[string]ToolDef),
	}, nil
}

func generateSessionID() string {
	return fmt.Sprintf("obs-%d", time.Now().UnixMilli())
}

type JSONRPCRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params"`
}

type JSONRPCResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *RPCError       `json:"error,omitempty"`
}

type RPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (p *Proxy) Run(ctx context.Context) error {
	go p.pipeStderr()

	scanner := bufio.NewScanner(p.stdout)
	scanner.Buffer(make([]byte, 0, 1024*1024), 10*1024*1024)

	reader := bufio.NewReader(os.Stdin)
	writer := bufio.NewWriter(os.Stdout)

	for {
		select {
		case <-ctx.Done():
			p.cmd.Process.Kill()
			return nil
		default:
		}

		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				p.cmd.Process.Kill()
				return nil
			}
			return fmt.Errorf("read stdin: %w", err)
		}

		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		var req JSONRPCRequest
		if err := json.Unmarshal([]byte(line), &req); err != nil {
			writeError(writer, nil, -32700, "Parse error")
			continue
		}

		switch req.Method {
		case "initialize":
			p.handleInitialize(writer, &req)
		case "tools/list":
			p.handleToolsList(writer, &req)
		case "tools/call":
			p.handleToolCall(writer, &req, scanner)
		case "resources/list":
			p.forwardRequest(writer, &req, scanner)
		case "prompts/list":
			p.forwardRequest(writer, &req, scanner)
		default:
			p.forwardRequest(writer, &req, scanner)
		}
	}
}

func (p *Proxy) handleInitialize(w *bufio.Writer, req *JSONRPCRequest) {
	p.forwardRequest(w, req, nil)

	store.UpsertSession(p.db, &store.Session{
		ID:         p.sessionID,
		ServerName: "observer",
	})
}

func (p *Proxy) handleToolsList(w *bufio.Writer, req *JSONRPCRequest) {
	resp := p.sendToTarget(req)
	if resp.Error != nil {
		writeResponse(w, req.ID, nil, resp.Error)
		return
	}

	var result struct {
		Tools []ToolDef `json:"tools"`
	}
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		writeError(w, req.ID, -32603, "Failed to parse tools list")
		return
	}

	for _, tool := range result.Tools {
		p.toolCache[tool.Name] = tool
		store.UpsertTool(p.db, tool.Name, tool.Description, string(tool.InputSchema))
	}

	for _, f := range p.config.Filter {
		if _, ok := p.toolCache[f]; ok {
			store.SetToolVisible(p.db, f, false)
		}
	}

	filtered := make([]ToolDef, 0, len(result.Tools))
	for _, tool := range result.Tools {
		if p.isHidden(tool.Name) {
			continue
		}
		filtered = append(filtered, tool)
	}

	traceTools := p.getTraceTools()
	for _, t := range traceTools {
		if !p.isHidden(t.Name) {
			filtered = append(filtered, t)
		}
	}

	if p.config.MaxTools > 0 && len(filtered) > p.config.MaxTools {
		filtered = filtered[:p.config.MaxTools]
	}

	out := map[string]interface{}{"tools": filtered}
	resultBytes, _ := json.Marshal(out)
	writeResponse(w, req.ID, resultBytes, nil)
}

func (p *Proxy) handleToolCall(w *bufio.Writer, req *JSONRPCRequest, scanner *bufio.Scanner) {
	var params struct {
		Name      string          `json:"name"`
		Arguments json.RawMessage `json:"arguments"`
	}
	if err := json.Unmarshal(req.Params, &params); err != nil {
		writeError(w, req.ID, -32602, "Invalid params")
		return
	}

	if strings.HasPrefix(params.Name, "trace.") {
		p.handleTraceTool(w, req, params.Name, params.Arguments)
		return
	}

	start := time.Now()
	resp := p.sendToTarget(req)
	duration := time.Since(start).Milliseconds()

	isError := resp.Error != nil

	inputStr := string(params.Arguments)
	outputStr := string(resp.Result)
	if resp.Error != nil {
		errBytes, _ := json.Marshal(resp.Error)
		outputStr = string(errBytes)
	}

	if len(p.config.RedactPatterns) > 0 {
		inputStr = store.Redact(inputStr, p.config.RedactPatterns)
		outputStr = store.Redact(outputStr, p.config.RedactPatterns)
	}

	store.InsertToolCall(p.db, &store.ToolCall{
		SessionID:     p.sessionID,
		ToolName:      params.Name,
		Input:         inputStr,
		Output:        outputStr,
		IsError:       isError,
		DurationMs:    duration,
		TokenEstimate: store.EstimateTokens(inputStr + outputStr),
	})

	if resp.Error != nil {
		writeResponse(w, req.ID, nil, resp.Error)
	} else {
		writeResponse(w, req.ID, resp.Result, nil)
	}
}

func (p *Proxy) forwardRequest(w *bufio.Writer, req *JSONRPCRequest, scanner *bufio.Scanner) {
	resp := p.sendToTarget(req)
	if resp.Error != nil {
		writeResponse(w, req.ID, nil, resp.Error)
	} else {
		writeResponse(w, req.ID, resp.Result, nil)
	}
}

func (p *Proxy) sendToTarget(req *JSONRPCRequest) *JSONRPCResponse {
	p.mu.Lock()
	defer p.mu.Unlock()

	reqBytes, _ := json.Marshal(req)
	reqBytes = append(reqBytes, '\n')

	if _, err := p.stdin.Write(reqBytes); err != nil {
		return &JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error:   &RPCError{Code: -32603, Message: fmt.Sprintf("upstream write error: %v", err)},
		}
	}

	scanner := bufio.NewScanner(p.stdout)
	scanner.Buffer(make([]byte, 0, 1024*1024), 10*1024*1024)
	if !scanner.Scan() {
		return &JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error:   &RPCError{Code: -32603, Message: "upstream closed"},
		}
	}

	var resp JSONRPCResponse
	line := scanner.Text()
	if err := json.Unmarshal([]byte(line), &resp); err != nil {
		return &JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error:   &RPCError{Code: -32603, Message: fmt.Sprintf("upstream parse error: %v", err)},
		}
	}
	return &resp
}

func (p *Proxy) pipeStderr() {
	scanner := bufio.NewScanner(p.stderr)
	for scanner.Scan() {
		log.Printf("[upstream] %s", scanner.Text())
	}
}

func (p *Proxy) isHidden(name string) bool {
	for _, f := range p.config.Filter {
		if name == f {
			return true
		}
	}
	return false
}

func (p *Proxy) getTraceTools() []ToolDef {
	rawNote := ""
	if !p.config.RawPayload {
		rawNote = " Returns metadata only (tool name, hash, duration, error) by default. Set OBSERVER_RAW_PAYLOAD=1 to include raw input/output."
	}
	tools := []ToolDef{
		{
			Name:        "trace.history",
			Description: "List recent tool calls for the current session. Returns metadata, duration, and timestamp." + rawNote + " Pass limit (default 10) and optional session_id to query a different session.",
			InputSchema: json.RawMessage(`{"type":"object","properties":{"limit":{"type":"integer","description":"Number of recent calls to return (default 10, max 100)"},"session_id":{"type":"string","description":"Filter by session ID"}}}`),
		},
		{
			Name:        "trace.stats",
			Description: "Get usage statistics across all tool calls. Returns total calls, unique tools, error count, average duration, and per-tool breakdown.",
			InputSchema: json.RawMessage(`{"type":"object","properties":{"session_id":{"type":"string","description":"Filter stats by session ID"}}}`),
		},
		{
			Name:        "trace.search",
			Description: "Search through tool call history for the current session by tool name, input, or output content. Returns matching tool calls." + rawNote + " Pass query and optional limit/session_id.",
			InputSchema: json.RawMessage(`{"type":"object","properties":{"query":{"type":"string","description":"Search query"},"limit":{"type":"integer","description":"Max results (default 20)"},"session_id":{"type":"string","description":"Filter by session ID"}},"required":["query"]}`),
		},
		{
			Name:        "trace.replay",
			Description: "Replay a previous tool call by its ID. Returns the original input and output for comparison." + rawNote,
			InputSchema: json.RawMessage(`{"type":"object","properties":{"call_id":{"type":"integer","description":"The ID of the tool call to replay"}},"required":["call_id"]}`),
		},
	}
	return tools
}

func (p *Proxy) handleTraceTool(w *bufio.Writer, req *JSONRPCRequest, name string, args json.RawMessage) {
	switch name {
	case "trace.history":
		p.traceHistory(w, req, args)
	case "trace.stats":
		p.traceStats(w, req, args)
	case "trace.search":
		p.traceSearch(w, req, args)
	case "trace.replay":
		p.traceReplay(w, req, args)
	default:
		writeError(w, req.ID, -32601, fmt.Sprintf("Unknown trace tool: %s", name))
	}
}

func (p *Proxy) traceHistory(w *bufio.Writer, req *JSONRPCRequest, args json.RawMessage) {
	var params struct {
		Limit     int    `json:"limit"`
		SessionID string `json:"session_id"`
	}
	json.Unmarshal(args, &params)
	if params.Limit == 0 {
		params.Limit = 10
	}
	if params.Limit > 100 {
		params.Limit = 100
	}

	sessionID := params.SessionID
	if sessionID == "" {
		sessionID = p.sessionID
	}

	calls, err := store.GetRecentCalls(p.db, params.Limit, sessionID)
	if err != nil {
		writeError(w, req.ID, -32603, fmt.Sprintf("query error: %v", err))
		return
	}

	var content string
	if p.config.RawPayload {
		content = store.FormatJSON(calls)
	} else {
		content = store.FormatJSON(store.ToMetaList(calls))
	}
	writeToolResult(w, req.ID, content)
}

func (p *Proxy) traceStats(w *bufio.Writer, req *JSONRPCRequest, args json.RawMessage) {
	var params struct {
		SessionID string `json:"session_id"`
	}
	json.Unmarshal(args, &params)

	sessionID := params.SessionID
	if sessionID == "" {
		sessionID = p.sessionID
	}

	stats, err := store.GetStats(p.db, sessionID)
	if err != nil {
		writeError(w, req.ID, -32603, fmt.Sprintf("stats error: %v", err))
		return
	}

	content := store.FormatJSON(stats)
	writeToolResult(w, req.ID, content)
}

func (p *Proxy) traceSearch(w *bufio.Writer, req *JSONRPCRequest, args json.RawMessage) {
	var params struct {
		Query     string `json:"query"`
		Limit     int    `json:"limit"`
		SessionID string `json:"session_id"`
	}
	json.Unmarshal(args, &params)
	if params.Limit == 0 {
		params.Limit = 20
	}

	sessionID := params.SessionID
	if sessionID == "" {
		sessionID = p.sessionID
	}

	calls, err := store.SearchCalls(p.db, params.Query, sessionID, params.Limit)
	if err != nil {
		writeError(w, req.ID, -32603, fmt.Sprintf("search error: %v", err))
		return
	}

	var content string
	if p.config.RawPayload {
		content = store.FormatJSON(calls)
	} else {
		content = store.FormatJSON(store.ToMetaList(calls))
	}
	writeToolResult(w, req.ID, content)
}

func (p *Proxy) traceReplay(w *bufio.Writer, req *JSONRPCRequest, args json.RawMessage) {
	var params struct {
		CallID int64 `json:"call_id"`
	}
	json.Unmarshal(args, &params)

	call, err := store.GetCallByID(p.db, params.CallID)
	if err != nil {
		writeError(w, req.ID, -32603, fmt.Sprintf("replay error: %v", err))
		return
	}

	var content string
	if p.config.RawPayload {
		content = store.FormatJSON(call)
	} else {
		content = store.FormatJSON(store.ToMeta(*call))
	}
	writeToolResult(w, req.ID, content)
}

func writeResponse(w *bufio.Writer, id json.RawMessage, result json.RawMessage, rpcErr *RPCError) {
	resp := JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result:  result,
		Error:   rpcErr,
	}
	b, _ := json.Marshal(resp)
	w.Write(b)
	w.WriteByte('\n')
	w.Flush()
}

func writeError(w *bufio.Writer, id json.RawMessage, code int, msg string) {
	writeResponse(w, id, nil, &RPCError{Code: code, Message: msg})
}

func writeToolResult(w *bufio.Writer, id json.RawMessage, text string) {
	result := map[string]interface{}{
		"content": []map[string]string{
			{"type": "text", "text": text},
		},
	}
	resultBytes, _ := json.Marshal(result)
	writeResponse(w, id, resultBytes, nil)
}
