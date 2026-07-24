package proxy

import (
	"fmt"
	"os"
	"strings"
)

type Config struct {
	Target         string
	DBPath         string
	LogLevel       string
	MaxTools       int
	Filter         []string
	RawPayload     bool
	RedactPatterns []string
	ListenAddr     string
}

func LoadConfig() *Config {
	c := &Config{
		Target:   os.Getenv("OBSERVER_TARGET"),
		DBPath:   os.Getenv("OBSERVER_DB_PATH"),
		LogLevel: os.Getenv("OBSERVER_LOG_LEVEL"),
		MaxTools: 0,
	}

	if v := os.Getenv("OBSERVER_MAX_TOOLS"); v != "" {
		fmt.Sscanf(v, "%d", &c.MaxTools)
	}

	if v := os.Getenv("OBSERVER_FILTER"); v != "" {
		c.Filter = strings.Split(v, ",")
		for i, f := range c.Filter {
			c.Filter[i] = strings.TrimSpace(f)
		}
	}

	if v := os.Getenv("OBSERVER_RAW_PAYLOAD"); v == "1" || v == "true" {
		c.RawPayload = true
	}

	if v := os.Getenv("OBSERVER_REDACT_PATTERNS"); v != "" {
		c.RedactPatterns = strings.Split(v, ",")
		for i, p := range c.RedactPatterns {
			c.RedactPatterns[i] = strings.TrimSpace(p)
		}
	}

	if v := os.Getenv("OBSERVER_LISTEN_ADDR"); v != "" {
		c.ListenAddr = v
	}

	if c.LogLevel == "" {
		c.LogLevel = "info"
	}

	if c.Target == "" {
		fmt.Fprintln(os.Stderr, "OBSERVER_TARGET is required")
		fmt.Fprintln(os.Stderr, "Example: OBSERVER_TARGET=\"npx -y @modelcontextprotocol/server-filesystem /tmp\" observer")
		os.Exit(1)
	}

	return c
}
