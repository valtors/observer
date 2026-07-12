package main

import (
	"bufio"
	"fmt"
	"os"
)

func main() {
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		if contains(line, "initialize") {
			fmt.Println(`{"jsonrpc":"2.0","id":1,"result":{"protocolVersion":"2024-11-05","capabilities":{"tools":{}},"serverInfo":{"name":"echo-server","version":"1.0"}}}`)
		} else if contains(line, "tools/list") {
			fmt.Println(`{"jsonrpc":"2.0","id":2,"result":{"tools":[{"name":"echo","description":"Echo back the input message","inputSchema":{"type":"object","properties":{"message":{"type":"string","description":"Message to echo"}},"required":["message"]}}]}}`)
		} else if contains(line, "tools/call") {
			fmt.Println(`{"jsonrpc":"2.0","id":3,"result":{"content":[{"type":"text","text":"echo: hello world"}]}}`)
		} else {
			fmt.Println(`{"jsonrpc":"2.0","id":99,"error":{"code":-32601,"message":"Method not found"}}`)
		}
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || indexOf(s, sub) >= 0)
}

func indexOf(s, sub string) int {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
