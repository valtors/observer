package proxy

import (
	"testing"
)

func TestLoadConfig_Defaults(t *testing.T) {
	t.Setenv("OBSERVER_TARGET", "echo hello")
	t.Setenv("OBSERVER_DB_PATH", "")
	t.Setenv("OBSERVER_LOG_LEVEL", "")
	t.Setenv("OBSERVER_MAX_TOOLS", "")
	t.Setenv("OBSERVER_FILTER", "")
	t.Setenv("OBSERVER_RAW_PAYLOAD", "")
	t.Setenv("OBSERVER_REDACT_PATTERNS", "")
	t.Setenv("OBSERVER_LISTEN_ADDR", "")

	c := LoadConfig()
	if c.Target != "echo hello" {
		t.Errorf("expected target 'echo hello', got %s", c.Target)
	}
	if c.LogLevel != "info" {
		t.Errorf("expected log level 'info', got %s", c.LogLevel)
	}
	if c.MaxTools != 0 {
		t.Errorf("expected max tools 0, got %d", c.MaxTools)
	}
	if c.RawPayload {
		t.Error("expected raw payload false")
	}
}

func TestLoadConfig_MaxTools(t *testing.T) {
	t.Setenv("OBSERVER_TARGET", "echo hello")
	t.Setenv("OBSERVER_MAX_TOOLS", "42")

	c := LoadConfig()
	if c.MaxTools != 42 {
		t.Errorf("expected max tools 42, got %d", c.MaxTools)
	}
}

func TestLoadConfig_Filter(t *testing.T) {
	t.Setenv("OBSERVER_TARGET", "echo hello")
	t.Setenv("OBSERVER_FILTER", "tool1, tool2 , tool3")

	c := LoadConfig()
	if len(c.Filter) != 3 {
		t.Fatalf("expected 3 filters, got %d", len(c.Filter))
	}
	if c.Filter[0] != "tool1" {
		t.Errorf("expected filter[0] 'tool1', got %s", c.Filter[0])
	}
	if c.Filter[1] != "tool2" {
		t.Errorf("expected filter[1] 'tool2', got %s", c.Filter[1])
	}
	if c.Filter[2] != "tool3" {
		t.Errorf("expected filter[2] 'tool3', got %s", c.Filter[2])
	}
}

func TestLoadConfig_RawPayload(t *testing.T) {
	t.Setenv("OBSERVER_TARGET", "echo hello")
	t.Setenv("OBSERVER_RAW_PAYLOAD", "true")

	c := LoadConfig()
	if !c.RawPayload {
		t.Error("expected raw payload true")
	}
}

func TestLoadConfig_RawPayload_One(t *testing.T) {
	t.Setenv("OBSERVER_TARGET", "echo hello")
	t.Setenv("OBSERVER_RAW_PAYLOAD", "1")

	c := LoadConfig()
	if !c.RawPayload {
		t.Error("expected raw payload true for '1'")
	}
}

func TestLoadConfig_RedactPatterns(t *testing.T) {
	t.Setenv("OBSERVER_TARGET", "echo hello")
	t.Setenv("OBSERVER_REDACT_PATTERNS", "password, token, secret")

	c := LoadConfig()
	if len(c.RedactPatterns) != 3 {
		t.Fatalf("expected 3 redact patterns, got %d", len(c.RedactPatterns))
	}
	if c.RedactPatterns[0] != "password" {
		t.Errorf("expected pattern[0] 'password', got %s", c.RedactPatterns[0])
	}
}

func TestLoadConfig_ListenAddr(t *testing.T) {
	t.Setenv("OBSERVER_TARGET", "echo hello")
	t.Setenv("OBSERVER_LISTEN_ADDR", ":8080")

	c := LoadConfig()
	if c.ListenAddr != ":8080" {
		t.Errorf("expected listen addr ':8080', got %s", c.ListenAddr)
	}
}

func TestLoadConfig_CustomLogLevel(t *testing.T) {
	t.Setenv("OBSERVER_TARGET", "echo hello")
	t.Setenv("OBSERVER_LOG_LEVEL", "debug")

	c := LoadConfig()
	if c.LogLevel != "debug" {
		t.Errorf("expected log level 'debug', got %s", c.LogLevel)
	}
}
