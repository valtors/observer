package main

import (
	"testing"
)

func TestVersion(t *testing.T) {
	if version != "0.1.0" {
		t.Errorf("expected 0.1.0, got %s", version)
	}
}

func TestPrintHelp(t *testing.T) {
	printHelp()
}
