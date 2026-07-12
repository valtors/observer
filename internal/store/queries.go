package store

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

type ToolCall struct {
	ID            int64  `json:"id"`
	SessionID     string `json:"session_id"`
	ToolName      string `json:"tool_name"`
	Input         string `json:"input,omitempty"`
	Output        string `json:"output,omitempty"`
	InputHash     string `json:"input_hash"`
	OutputHash    string `json:"output_hash"`
	IsError       bool   `json:"is_error"`
	DurationMs    int64  `json:"duration_ms"`
	TokenEstimate int    `json:"token_estimate"`
	CreatedAt     string `json:"created_at"`
}

type ToolCallMeta struct {
	ID            int64  `json:"id"`
	SessionID     string `json:"session_id"`
	ToolName      string `json:"tool_name"`
	InputHash     string `json:"input_hash"`
	OutputHash    string `json:"output_hash"`
	IsError       bool   `json:"is_error"`
	DurationMs    int64  `json:"duration_ms"`
	TokenEstimate int    `json:"token_estimate"`
	CreatedAt     string `json:"created_at"`
}

func HashString(s string) string {
	h := sha256.Sum256([]byte(s))
	return hex.EncodeToString(h[:16])
}

func Redact(text string, patterns []string) string {
	out := text
	for _, p := range patterns {
		out = strings.ReplaceAll(out, p, "[REDACTED]")
	}
	return out
}

type Session struct {
	ID            string `json:"id"`
	ClientName    string `json:"client_name"`
	ClientVersion string `json:"client_version"`
	ServerName    string `json:"server_name"`
	ServerVersion string `json:"server_version"`
	CreatedAt     string `json:"created_at"`
	LastActive    string `json:"last_active"`
}

type ToolInfo struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	InputSchema string `json:"input_schema"`
	Visible     bool   `json:"visible"`
	CallCount   int    `json:"call_count"`
}

func InsertToolCall(db *sql.DB, call *ToolCall) error {
	call.InputHash = HashString(call.Input)
	call.OutputHash = HashString(call.Output)

	_, err := db.Exec(
		`INSERT INTO tool_calls (session_id, tool_name, input, output, input_hash, output_hash, is_error, duration_ms, token_estimate) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		call.SessionID, call.ToolName, call.Input, call.Output, call.InputHash, call.OutputHash, call.IsError, call.DurationMs, call.TokenEstimate,
	)
	if err != nil {
		return fmt.Errorf("insert tool call: %w", err)
	}

	_, err = db.Exec(
		`UPDATE tool_registry SET call_count = call_count + 1 WHERE name = ?`,
		call.ToolName,
	)
	return err
}

func GetRecentCalls(db *sql.DB, limit int, sessionID string) ([]ToolCall, error) {
	var rows *sql.Rows
	var err error

	if sessionID != "" {
		rows, err = db.Query(
			`SELECT id, session_id, tool_name, input, output, input_hash, output_hash, is_error, duration_ms, token_estimate, created_at
			FROM tool_calls WHERE session_id = ? ORDER BY created_at DESC LIMIT ?`,
			sessionID, limit,
		)
	} else {
		rows, err = db.Query(
			`SELECT id, session_id, tool_name, input, output, input_hash, output_hash, is_error, duration_ms, token_estimate, created_at
			FROM tool_calls ORDER BY created_at DESC LIMIT ?`,
			limit,
		)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanToolCalls(rows)
}

func SearchCalls(db *sql.DB, query, sessionID string, limit int) ([]ToolCall, error) {
	pattern := "%" + query + "%"
	var rows *sql.Rows
	var err error

	if sessionID != "" {
		rows, err = db.Query(
			`SELECT id, session_id, tool_name, input, output, input_hash, output_hash, is_error, duration_ms, token_estimate, created_at
			FROM tool_calls WHERE session_id = ? AND (tool_name LIKE ? OR input LIKE ? OR output LIKE ?)
			ORDER BY created_at DESC LIMIT ?`,
			sessionID, pattern, pattern, pattern, limit,
		)
	} else {
		rows, err = db.Query(
			`SELECT id, session_id, tool_name, input, output, input_hash, output_hash, is_error, duration_ms, token_estimate, created_at
			FROM tool_calls WHERE tool_name LIKE ? OR input LIKE ? OR output LIKE ?
			ORDER BY created_at DESC LIMIT ?`,
			pattern, pattern, pattern, limit,
		)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanToolCalls(rows)
}

func GetCallByID(db *sql.DB, id int64) (*ToolCall, error) {
	row := db.QueryRow(
		`SELECT id, session_id, tool_name, input, output, input_hash, output_hash, is_error, duration_ms, token_estimate, created_at
		FROM tool_calls WHERE id = ?`,
		id,
	)
	var c ToolCall
	var isErr int
	err := row.Scan(&c.ID, &c.SessionID, &c.ToolName, &c.Input, &c.Output, &c.InputHash, &c.OutputHash, &isErr, &c.DurationMs, &c.TokenEstimate, &c.CreatedAt)
	if err != nil {
		return nil, err
	}
	c.IsError = isErr == 1
	return &c, nil
}

type Stats struct {
	TotalCalls     int            `json:"total_calls"`
	UniqueTools    int            `json:"unique_tools"`
	ErrorCount     int            `json:"error_count"`
	AvgDurationMs  float64        `json:"avg_duration_ms"`
	TotalDurationMs int64         `json:"total_duration_ms"`
	ToolBreakdown  []ToolStat     `json:"tool_breakdown"`
}

type ToolStat struct {
	ToolName      string  `json:"tool_name"`
	CallCount     int     `json:"call_count"`
	ErrorCount    int     `json:"error_count"`
	AvgDurationMs float64 `json:"avg_duration_ms"`
}

func GetStats(db *sql.DB, sessionID string) (*Stats, error) {
	s := &Stats{}

	var where string
	var args []interface{}
	if sessionID != "" {
		where = " WHERE session_id = ?"
		args = append(args, sessionID)
	}

	err := db.QueryRow(
		`SELECT COUNT(*), COALESCE(SUM(CASE WHEN is_error = 1 THEN 1 ELSE 0 END), 0), COALESCE(AVG(duration_ms), 0), COALESCE(SUM(duration_ms), 0) FROM tool_calls`+where,
		args...,
	).Scan(&s.TotalCalls, &s.ErrorCount, &s.AvgDurationMs, &s.TotalDurationMs)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}

	err = db.QueryRow(
		`SELECT COUNT(DISTINCT tool_name) FROM tool_calls`+where,
		args...,
	).Scan(&s.UniqueTools)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}

	var rows *sql.Rows
	if sessionID != "" {
		rows, err = db.Query(
			`SELECT tool_name, COUNT(*), SUM(CASE WHEN is_error = 1 THEN 1 ELSE 0 END), AVG(duration_ms)
			FROM tool_calls WHERE session_id = ? GROUP BY tool_name ORDER BY COUNT(*) DESC`,
			sessionID,
		)
	} else {
		rows, err = db.Query(
			`SELECT tool_name, COUNT(*), SUM(CASE WHEN is_error = 1 THEN 1 ELSE 0 END), AVG(duration_ms)
			FROM tool_calls GROUP BY tool_name ORDER BY COUNT(*) DESC`,
		)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var ts ToolStat
		if err := rows.Scan(&ts.ToolName, &ts.CallCount, &ts.ErrorCount, &ts.AvgDurationMs); err != nil {
			return nil, err
		}
		s.ToolBreakdown = append(s.ToolBreakdown, ts)
	}

	return s, nil
}

func UpsertSession(db *sql.DB, s *Session) error {
	_, err := db.Exec(
		`INSERT INTO sessions (id, client_name, client_version, server_name, server_version)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET last_active = CURRENT_TIMESTAMP`,
		s.ID, s.ClientName, s.ClientVersion, s.ServerName, s.ServerVersion,
	)
	return err
}

func UpsertTool(db *sql.DB, name, description, inputSchema string) error {
	_, err := db.Exec(
		`INSERT INTO tool_registry (name, description, input_schema, visible, call_count)
		VALUES (?, ?, ?, 1, 0)
		ON CONFLICT(name) DO UPDATE SET description = ?, input_schema = ?`,
		name, description, inputSchema, name, description, inputSchema,
	)
	return err
}

func SetToolVisible(db *sql.DB, name string, visible bool) error {
	v := 0
	if visible {
		v = 1
	}
	_, err := db.Exec(`UPDATE tool_registry SET visible = ? WHERE name = ?`, v, name)
	return err
}

func GetVisibleTools(db *sql.DB) ([]ToolInfo, error) {
	rows, err := db.Query(
		`SELECT name, description, input_schema, visible, call_count FROM tool_registry WHERE visible = 1 ORDER BY name`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tools []ToolInfo
	for rows.Next() {
		var t ToolInfo
		var vis int
		if err := rows.Scan(&t.Name, &t.Description, &t.InputSchema, &vis, &t.CallCount); err != nil {
			return nil, err
		}
		t.Visible = vis == 1
		tools = append(tools, t)
	}
	return tools, nil
}

func GetAllTools(db *sql.DB) ([]ToolInfo, error) {
	rows, err := db.Query(
		`SELECT name, description, input_schema, visible, call_count FROM tool_registry ORDER BY name`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tools []ToolInfo
	for rows.Next() {
		var t ToolInfo
		var vis int
		if err := rows.Scan(&t.Name, &t.Description, &t.InputSchema, &vis, &t.CallCount); err != nil {
			return nil, err
		}
		t.Visible = vis == 1
		tools = append(tools, t)
	}
	return tools, nil
}

func scanToolCalls(rows *sql.Rows) ([]ToolCall, error) {
	var calls []ToolCall
	for rows.Next() {
		var c ToolCall
		var isErr int
		if err := rows.Scan(&c.ID, &c.SessionID, &c.ToolName, &c.Input, &c.Output, &c.InputHash, &c.OutputHash, &isErr, &c.DurationMs, &c.TokenEstimate, &c.CreatedAt); err != nil {
			return nil, err
		}
		c.IsError = isErr == 1
		calls = append(calls, c)
	}
	return calls, rows.Err()
}

func ToMeta(c ToolCall) ToolCallMeta {
	return ToolCallMeta{
		ID:            c.ID,
		SessionID:     c.SessionID,
		ToolName:      c.ToolName,
		InputHash:     c.InputHash,
		OutputHash:    c.OutputHash,
		IsError:       c.IsError,
		DurationMs:    c.DurationMs,
		TokenEstimate: c.TokenEstimate,
		CreatedAt:     c.CreatedAt,
	}
}

func ToMetaList(calls []ToolCall) []ToolCallMeta {
	result := make([]ToolCallMeta, len(calls))
	for i, c := range calls {
		result[i] = ToMeta(c)
	}
	return result
}

func EstimateTokens(text string) int {
	return len(text) / 4
}

func FormatJSON(v interface{}) string {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Sprintf("%v", v)
	}
	return string(b)
}

func FormatDuration(ms int64) string {
	if ms < 1000 {
		return fmt.Sprintf("%dms", ms)
	}
	d := time.Duration(ms) * time.Millisecond
	return d.String()
}
