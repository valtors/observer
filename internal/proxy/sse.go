package proxy

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"

	"github.com/valtors/observer/internal/store"
)

type SSEServer struct {
	proxy   *Proxy
	clients map[chan string]struct{}
	mu      sync.Mutex
}

func NewSSEServer(p *Proxy) *SSEServer {
	return &SSEServer{
		proxy:   p,
		clients: make(map[chan string]struct{}),
	}
}

func (s *SSEServer) HandleSSE(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}

	ch := make(chan string, 64)
	s.mu.Lock()
	s.clients[ch] = struct{}{}
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		delete(s.clients, ch)
		s.mu.Unlock()
		close(ch)
	}()

	fmt.Fprintf(w, "event: endpoint\n")
	fmt.Fprintf(w, "data: /message?sessionId=%s\n\n", s.proxy.sessionID)
	flusher.Flush()

	for {
		select {
		case msg := <-ch:
			fmt.Fprintf(w, "event: message\n")
			fmt.Fprintf(w, "data: %s\n\n", msg)
			flusher.Flush()
		case <-r.Context().Done():
			return
		}
	}
}

func (s *SSEServer) HandleMessage(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req JSONRPCRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeHTTPError(w, nil, -32700, "Parse error")
		return
	}

	resp := s.proxy.handleHTTPMessage(&req)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (p *Proxy) handleHTTPMessage(req *JSONRPCRequest) *JSONRPCResponse {
	switch req.Method {
	case "initialize":
		resp := p.sendToTarget(req)
		store.UpsertSession(p.db, &store.Session{
			ID:         p.sessionID,
			ServerName: "observer",
		})
		return &resp
	case "tools/list":
		return p.handleToolsListHTTP(req)
	case "tools/call":
		return p.handleToolCallHTTP(req)
	default:
		resp := p.sendToTarget(req)
		return &resp
	}
}

func (p *Proxy) handleToolsListHTTP(req *JSONRPCRequest) *JSONRPCResponse {
	resp := p.sendToTarget(req)
	if resp.Error != nil {
		return &resp
	}

	var result struct {
		Tools []ToolDef `json:"tools"`
	}
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		return &JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error:   &RPCError{Code: -32603, Message: "Failed to parse tools list"},
		}
	}

	for _, tool := range result.Tools {
		p.toolCache[tool.Name] = tool
	}

	filtered := make([]ToolDef, 0, len(result.Tools))
	for _, tool := range result.Tools {
		if !p.isHidden(tool.Name) {
			filtered = append(filtered, tool)
		}
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
	return &JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result:  resultBytes,
	}
}

func (p *Proxy) handleToolCallHTTP(req *JSONRPCRequest) *JSONRPCResponse {
	var params struct {
		Name      string          `json:"name"`
		Arguments json.RawMessage `json:"arguments"`
	}
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return &JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error:   &RPCError{Code: -32602, Message: "Invalid params"},
		}
	}

	if len(params.Name) > 6 && params.Name[:6] == "trace." {
		return p.handleTraceToolHTTP(req, params.Name, params.Arguments)
	}

	resp := p.sendToTarget(req)
	return &resp
}

func (p *Proxy) handleTraceToolHTTP(req *JSONRPCRequest, name string, args json.RawMessage) *JSONRPCResponse {
	switch name {
	case "trace.history":
		return p.traceHistoryHTTP(req, args)
	case "trace.stats":
		return p.traceStatsHTTP(req, args)
	case "trace.search":
		return p.traceSearchHTTP(req, args)
	case "trace.replay":
		return p.traceReplayHTTP(req, args)
	default:
		return &JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error:   &RPCError{Code: -32601, Message: fmt.Sprintf("Unknown trace tool: %s", name)},
		}
	}
}

func (p *Proxy) traceHistoryHTTP(req *JSONRPCRequest, args json.RawMessage) *JSONRPCResponse {
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
		return &JSONRPCResponse{JSONRPC: "2.0", ID: req.ID, Error: &RPCError{Code: -32603, Message: fmt.Sprintf("query error: %v", err)}}
	}
	var content string
	if p.config.RawPayload {
		content = store.FormatJSON(calls)
	} else {
		content = store.FormatJSON(store.ToMetaList(calls))
	}
	return makeToolResult(req.ID, content)
}

func (p *Proxy) traceStatsHTTP(req *JSONRPCRequest, args json.RawMessage) *JSONRPCResponse {
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
		return &JSONRPCResponse{JSONRPC: "2.0", ID: req.ID, Error: &RPCError{Code: -32603, Message: fmt.Sprintf("stats error: %v", err)}}
	}
	content := store.FormatJSON(stats)
	return makeToolResult(req.ID, content)
}

func (p *Proxy) traceSearchHTTP(req *JSONRPCRequest, args json.RawMessage) *JSONRPCResponse {
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
		return &JSONRPCResponse{JSONRPC: "2.0", ID: req.ID, Error: &RPCError{Code: -32603, Message: fmt.Sprintf("search error: %v", err)}}
	}
	var content string
	if p.config.RawPayload {
		content = store.FormatJSON(calls)
	} else {
		content = store.FormatJSON(store.ToMetaList(calls))
	}
	return makeToolResult(req.ID, content)
}

func (p *Proxy) traceReplayHTTP(req *JSONRPCRequest, args json.RawMessage) *JSONRPCResponse {
	var params struct {
		CallID int64 `json:"call_id"`
	}
	json.Unmarshal(args, &params)
	call, err := store.GetCallByID(p.db, params.CallID)
	if err != nil {
		return &JSONRPCResponse{JSONRPC: "2.0", ID: req.ID, Error: &RPCError{Code: -32603, Message: fmt.Sprintf("replay error: %v", err)}}
	}
	var content string
	if p.config.RawPayload {
		content = store.FormatJSON(call)
	} else {
		content = store.FormatJSON(store.ToMeta(*call))
	}
	return makeToolResult(req.ID, content)
}

func makeToolResult(id json.RawMessage, text string) *JSONRPCResponse {
	result := map[string]interface{}{
		"content": []map[string]string{
			{"type": "text", "text": text},
		},
	}
	resultBytes, _ := json.Marshal(result)
	return &JSONRPCResponse{JSONRPC: "2.0", ID: id, Result: resultBytes}
}

func writeHTTPError(w http.ResponseWriter, id json.RawMessage, code int, msg string) {
	resp := JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Error:   &RPCError{Code: code, Message: msg},
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusBadRequest)
	json.NewEncoder(w).Encode(resp)
}
