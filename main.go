package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/valtors/observer/internal/proxy"
	"github.com/valtors/observer/internal/store"
)

const version = "0.1.0"

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "--help", "-h":
			printHelp()
			return
		case "--version", "-v":
			fmt.Println(version)
			return
		}
	}

	config := proxy.LoadConfig()

	db, err := store.Open(config.DBPath)
	if err != nil {
		log.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()

	if err := store.Migrate(db); err != nil {
		log.Fatalf("failed to migrate database: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		cancel()
	}()

	p, err := proxy.New(config, db)
	if err != nil {
		log.Fatalf("failed to create proxy: %v", err)
	}

	if err := p.Run(ctx); err != nil {
		log.Fatalf("proxy error: %v", err)
	}
}

func printHelp() {
	fmt.Println(`observer - transparent MCP proxy for agent observability

Usage:
  observer                    Start the proxy server (stdio transport)
  observer --version           Print version
  observer --help              Show this help

Configuration (environment variables):
  OBSERVER_TARGET             Command to run the upstream MCP server
                              Example: "npx -y @modelcontextprotocol/server-filesystem /tmp"
  OBSERVER_DB_PATH            SQLite database path (default: ~/.observer/trace.db)
  OBSERVER_LOG_LEVEL          Log level: debug, info, warn, error (default: info)
  OBSERVER_MAX_TOOLS          Max tools to expose to client (0 = all, default: 0)
  OBSERVER_FILTER             Comma-separated tool names to hide from client

How it works:
  Observer sits between your MCP client (Claude, Cline, Goose, etc.)
  and the actual MCP server. It logs every tool call to SQLite and
  exposes trace history through additional MCP tools:
    - trace.history   List recent tool calls
    - trace.stats     Get usage statistics
    - trace.replay    Replay a previous tool call
    - trace.search    Search through tool call history

Example:
  OBSERVER_TARGET="npx -y @modelcontextprotocol/server-filesystem /tmp" observer`)
}
