package store

import (
	"testing"
)

func TestRedact(t *testing.T) {
	out := Redact("my password is password123", []string{"password"})
	if out != "my [REDACTED] is [REDACTED]123" {
		t.Errorf("unexpected: %s", out)
	}
}

func TestRedact_NoPatterns(t *testing.T) {
	out := Redact("hello world", nil)
	if out != "hello world" {
		t.Errorf("expected unchanged, got %s", out)
	}
}

func TestRedact_MultiplePatterns(t *testing.T) {
	out := Redact("password and token here", []string{"password", "token"})
	if out != "[REDACTED] and [REDACTED] here" {
		t.Errorf("unexpected: %s", out)
	}
}

func TestToMeta(t *testing.T) {
	c := ToolCall{
		ID: 1, SessionID: "s1", ToolName: "read",
		Input: "in", Output: "out",
		InputHash: "h1", OutputHash: "h2",
		IsError: true, DurationMs: 42,
		TokenEstimate: 10, CreatedAt: "2024-01-01",
	}
	m := ToMeta(c)
	if m.ID != 1 || m.SessionID != "s1" || m.ToolName != "read" {
		t.Error("meta mismatch")
	}
	if m.InputHash != "h1" || m.OutputHash != "h2" {
		t.Error("hash mismatch")
	}
	if !m.IsError || m.DurationMs != 42 || m.TokenEstimate != 10 {
		t.Error("field mismatch")
	}
	if m.CreatedAt != "2024-01-01" {
		t.Error("timestamp mismatch")
	}
}

func TestToMetaList(t *testing.T) {
	calls := []ToolCall{
		{ID: 1, ToolName: "a"},
		{ID: 2, ToolName: "b"},
	}
	l := ToMetaList(calls)
	if len(l) != 2 {
		t.Fatalf("expected 2, got %d", len(l))
	}
	if l[0].ID != 1 || l[1].ID != 2 {
		t.Error("id mismatch")
	}
}

func TestToMetaList_Empty(t *testing.T) {
	l := ToMetaList(nil)
	if len(l) != 0 {
		t.Errorf("expected empty, got %d", len(l))
	}
}

func TestEstimateTokens(t *testing.T) {
	if EstimateTokens("hello") != 1 {
		t.Errorf("expected 1, got %d", EstimateTokens("hello"))
	}
	if EstimateTokens("hello world test") != 4 {
		t.Errorf("expected 3, got %d", EstimateTokens("hello world test"))
	}
}

func TestFormatJSON(t *testing.T) {
	s := FormatJSON(map[string]int{"a": 1})
	if s == "" {
		t.Error("expected non-empty")
	}
}

func TestFormatDuration(t *testing.T) {
	if s := FormatDuration(50); s != "50ms" {
		t.Errorf("expected '50ms', got %s", s)
	}
	if s := FormatDuration(1500); s != "1.5s" {
		t.Errorf("expected '1.5s', got %s", s)
	}
	if s := FormatDuration(60000); s != "1m0s" {
		t.Errorf("expected '1m0s', got %s", s)
	}
}

func TestHashString_Deterministic(t *testing.T) {
	h1 := HashString("test")
	h2 := HashString("test")
	if h1 != h2 {
		t.Error("hash should be deterministic")
	}
	if h1 == HashString("other") {
		t.Error("different inputs should produce different hashes")
	}
}

func TestHashString_EmptyString(t *testing.T) {
	h := HashString("")
	if h == "" {
		t.Error("empty string should produce non-empty hash")
	}
}
