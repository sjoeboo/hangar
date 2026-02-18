package mcppool

import (
	"bufio"
	"net"
	"strings"
	"testing"
)

func TestScannerHandlesLargeMessages(t *testing.T) {
	// Default bufio.Scanner fails on messages > 64KB
	// MCP responses from tools like context7, firecrawl regularly exceed this
	largeMessage := strings.Repeat("x", 100*1024) // 100KB

	// This simulates what broadcastResponses does with our fix
	scanner := bufio.NewScanner(strings.NewReader(largeMessage + "\n"))
	scanner.Buffer(make([]byte, 64*1024), 10*1024*1024) // Our fix: 10MB max

	if !scanner.Scan() {
		t.Fatalf("Scanner should handle 100KB message, got error: %v", scanner.Err())
	}
	if len(scanner.Text()) != 100*1024 {
		t.Errorf("Expected 100KB message, got %d bytes", len(scanner.Text()))
	}
}

func TestDefaultScannerFailsOnLargeMessages(t *testing.T) {
	// Proves the bug: default scanner cannot handle >64KB
	largeMessage := strings.Repeat("x", 100*1024)

	scanner := bufio.NewScanner(strings.NewReader(largeMessage + "\n"))
	// No Buffer() call = default 64KB limit

	if scanner.Scan() {
		t.Fatal("Default scanner should fail on 100KB message (this proves the bug exists)")
	}
	if scanner.Err() == nil {
		t.Fatal("Expected bufio.ErrTooLong error")
	}
}

func TestBroadcastResponsesClosesClientsOnFailure(t *testing.T) {
	// When broadcastResponses exits (MCP died), all client connections
	// should be closed so reconnecting proxies know to retry
	proxy := &SocketProxy{
		name:       "test",
		clients:    make(map[string]net.Conn),
		requestMap: make(map[interface{}]string),
		Status:     StatusRunning,
	}

	// Create a pipe to simulate a client connection
	server, client := net.Pipe()
	proxy.clientsMu.Lock()
	proxy.clients["test-client"] = server
	proxy.clientsMu.Unlock()

	// Simulate what happens after broadcastResponses exits
	proxy.closeAllClientsOnFailure()

	// Client should be closed
	buf := make([]byte, 1)
	_, err := client.Read(buf)
	if err == nil {
		t.Error("Expected client connection to be closed")
	}

	// Clients map should be empty
	proxy.clientsMu.RLock()
	count := len(proxy.clients)
	proxy.clientsMu.RUnlock()
	if count != 0 {
		t.Errorf("Expected 0 clients after failure, got %d", count)
	}
}
