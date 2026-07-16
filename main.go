package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
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

	if config.ListenAddr != "" {
		runSSE(ctx, p, config.ListenAddr)
		return
	}

	if err := p.Run(ctx); err != nil {
		log.Fatalf("proxy error: %v", err)
	}
}

func runSSE(ctx context.Context, p *proxy.Proxy, addr string) {
	sse := proxy.NewSSEServer(p)
	mux := http.NewServeMux()
	mux.HandleFunc("/sse", sse.HandleSSE)
	mux.HandleFunc("/message", sse.HandleMessage)

	srv := &http.Server{Addr: addr, Handler: mux}

	go func() {
		<-ctx.Done()
		srv.Shutdown(context.Background())
	}()

	log.Printf("observer SSE on %s (/sse, /message)", addr)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("SSE server error: %v", err)
	}
}

func printHelp() {
	fmt.Println(`observer - transparent MCP proxy

it watches what your agent does. every tool call, every argument, every response.
logged to sqlite. you can search it later. you can replay it later.

usage:
  observer                    start the proxy (stdio)
  OBSERVER_LISTEN_ADDR=:8080  start with SSE transport (http)
  observer --version           print version
  observer --help              this

environment:
  OBSERVER_TARGET             command to run the upstream MCP server
  OBSERVER_DB_PATH            sqlite database path (default: ~/.observer/trace.db)
  OBSERVER_LOG_LEVEL          debug, info, warn, error (default: info)
  OBSERVER_MAX_TOOLS          max tools to expose to client (0 = all)
  OBSERVER_FILTER             comma-separated tool names to hide from client

trace tools (injected into your agent's tool list):
  trace.history   recent tool calls
  trace.stats     usage statistics
  trace.replay    replay a previous tool call
  trace.search    search through call history

example:
  OBSERVER_TARGET="npx -y @modelcontextprotocol/server-filesystem /tmp" observer`)
}
