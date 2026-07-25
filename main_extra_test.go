package main

import (
	"context"
	"database/sql"
	"net"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/valtors/observer/internal/proxy"
)

func TestRunSSE(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("sql.Open: %v", err)
	}
	defer db.Close()

	p, err := proxy.New(&proxy.Config{Target: "echo"}, db)
	if err != nil {
		t.Fatalf("proxy.New: %v", err)
	}

	// Find a free port
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("net.Listen: %v", err)
	}
	addr := ln.Addr().String()
	ln.Close()

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		runSSE(ctx, p, addr)
		close(done)
	}()

	time.Sleep(100 * time.Millisecond)
	cancel()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Error("runSSE did not shut down in time")
	}
}
