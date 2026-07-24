package main

import (
	"os/exec"
	"strings"
	"testing"
)

func TestVersion(t *testing.T) {
	if version != "0.1.0" {
		t.Errorf("expected version 0.1.0, got %s", version)
	}
}

func TestHelpFlag(t *testing.T) {
	cmd := exec.Command("go", "run", ".", "--help")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("go run --help failed: %v", err)
	}
	s := string(out)
	if !strings.Contains(s, "observer") {
		t.Error("help output should contain 'observer'")
	}
	if !strings.Contains(s, "OBSERVER_TARGET") {
		t.Error("help output should mention OBSERVER_TARGET")
	}
}

func TestVersionFlag(t *testing.T) {
	cmd := exec.Command("go", "run", ".", "--version")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("go run --version failed: %v", err)
	}
	s := strings.TrimSpace(string(out))
	if s != version {
		t.Errorf("expected %s, got %s", version, s)
	}
}
