package server

import (
	"bufio"
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/celerix-dev/celerix-store/pkg/engine"
)

func TestRouter_TCP_Commands(t *testing.T) {
	store := engine.NewMemStore(nil, nil)
	router := NewRouter(store)

	// Start listener on a random port
	// We'll let Router.Listen use ":0" to get a random port
	go router.Listen("0")

	// Wait a bit for listener to be set
	var port string
	for i := 0; i < 10; i++ {
		time.Sleep(50 * time.Millisecond)
		router.mu.Lock()
		if router.listener != nil {
			port = fmt.Sprintf("%d", router.listener.Addr().(*net.TCPAddr).Port)
			router.mu.Unlock()
			break
		}
		router.mu.Unlock()
	}

	if port == "" {
		t.Fatalf("Server did not start in time")
	}

	defer router.Stop()

	// Client
	conn, err := net.Dial("tcp", "127.0.0.1:"+port)
	if err != nil {
		t.Fatalf("Failed to dial: %v", err)
	}
	defer conn.Close()

	reader := bufio.NewReader(conn)

	// Test PING
	fmt.Fprintf(conn, "PING\n")
	line, _ := reader.ReadString('\n')
	if line != "PONG\n" {
		t.Errorf("Expected PONG, got %q", line)
	}

	// Test SET
	fmt.Fprintf(conn, "SET p1 a1 k1 {\"name\": \"test\"}\n")
	line, _ = reader.ReadString('\n')
	if line != "OK\n" {
		t.Errorf("Expected OK, got %q", line)
	}

	// Test GET
	fmt.Fprintf(conn, "GET p1 a1 k1\n")
	line, _ = reader.ReadString('\n')
	if line != "OK {\"name\":\"test\"}\n" {
		t.Errorf("Expected OK {\"name\":\"test\"}, got %q", line)
	}

	// Test DEL
	fmt.Fprintf(conn, "DEL p1 a1 k1\n")
	line, _ = reader.ReadString('\n')
	if line != "OK\n" {
		t.Errorf("Expected OK, got %q", line)
	}

	// Test GET after DEL
	fmt.Fprintf(conn, "GET p1 a1 k1\n")
	line, _ = reader.ReadString('\n')
	if len(line) < 3 || line[:3] != "ERR" {
		t.Errorf("Expected ERR, got %q", line)
	}
}

func TestRouter_ConcurrentConnections(t *testing.T) {
	store := engine.NewMemStore(nil, nil)
	router := NewRouter(store)

	go router.Listen("0")
	var port string
	for i := 0; i < 10; i++ {
		time.Sleep(50 * time.Millisecond)
		router.mu.Lock()
		if router.listener != nil {
			port = fmt.Sprintf("%d", router.listener.Addr().(*net.TCPAddr).Port)
			router.mu.Unlock()
			break
		}
		router.mu.Unlock()
	}
	if port == "" {
		t.Fatalf("Server did not start in time")
	}
	defer router.Stop()

	// Try to open 101 connections
	conns := make([]net.Conn, 0)
	for i := 0; i < 110; i++ {
		conn, err := net.DialTimeout("tcp", "127.0.0.1:"+port, 100*time.Millisecond)
		if err == nil {
			conns = append(conns, conn)
		}
	}

	for _, c := range conns {
		c.Close()
	}
}

func TestRouter_MalformedCommands(t *testing.T) {
	store := engine.NewMemStore(nil, nil)
	router := NewRouter(store)

	go router.Listen("0")
	var port string
	for i := 0; i < 10; i++ {
		time.Sleep(50 * time.Millisecond)
		router.mu.Lock()
		if router.listener != nil {
			port = fmt.Sprintf("%d", router.listener.Addr().(*net.TCPAddr).Port)
			router.mu.Unlock()
			break
		}
		router.mu.Unlock()
	}
	if port == "" {
		t.Fatalf("Server did not start in time")
	}
	defer router.Stop()

	conn, _ := net.Dial("tcp", "127.0.0.1:"+port)
	defer conn.Close()
	reader := bufio.NewReader(conn)

	// Case 1: Incomplete command (less than 5 parts for SET)
	fmt.Fprintf(conn, "SET p1 a1 k1\n")

	// Case 2: Malformed JSON in SET (enough parts, but invalid JSON)
	fmt.Fprintf(conn, "SET p1 a1 k1 {invalid}\n")

	// Flush with a valid command and check response
	fmt.Fprintf(conn, "PING\n")

	// We read until we find PONG. We might get "ERR invalid json value" first.
	foundPong := false
	for i := 0; i < 3; i++ {
		line, err := reader.ReadString('\n')
		if err != nil {
			break
		}
		if line == "PONG\n" {
			foundPong = true
			break
		}
	}
	if !foundPong {
		t.Error("Did not receive PONG")
	}
}

func TestRouter_DumpAndGlobal(t *testing.T) {
	store := engine.NewMemStore(nil, nil)
	store.Set("p1", "a1", "k1", "v1")
	router := NewRouter(store)

	go router.Listen("0")
	var port string
	for i := 0; i < 10; i++ {
		time.Sleep(50 * time.Millisecond)
		router.mu.Lock()
		if router.listener != nil {
			port = fmt.Sprintf("%d", router.listener.Addr().(*net.TCPAddr).Port)
			router.mu.Unlock()
			break
		}
		router.mu.Unlock()
	}
	if port == "" {
		t.Fatalf("Server did not start in time")
	}
	defer router.Stop()

	conn, err := net.Dial("tcp", "127.0.0.1:"+port)
	if err != nil {
		t.Fatalf("Failed to dial: %v", err)
	}
	defer conn.Close()
	reader := bufio.NewReader(conn)

	// Test LIST_PERSONAS
	fmt.Fprintf(conn, "LIST_PERSONAS\n")
	line, err := reader.ReadString('\n')
	if err != nil {
		t.Fatalf("Read error: %v", err)
	}
	if line != "OK [\"p1\"]\n" {
		t.Errorf("Expected OK [\"p1\"], got %q", line)
	}

	// Test LIST_APPS
	fmt.Fprintf(conn, "LIST_APPS p1\n")
	line, _ = reader.ReadString('\n')
	if line != "OK [\"a1\"]\n" {
		t.Errorf("Expected OK [\"a1\"], got %q", line)
	}

	// Test DUMP
	fmt.Fprintf(conn, "DUMP p1 a1\n")
	line, _ = reader.ReadString('\n')
	if line != "OK {\"k1\":\"v1\"}\n" {
		t.Errorf("Expected OK {\"k1\":\"v1\"}, got %q", line)
	}

	// Test GET_GLOBAL
	fmt.Fprintf(conn, "GET_GLOBAL a1 k1\n")
	line, _ = reader.ReadString('\n')
	if line != "OK {\"persona\":\"p1\",\"value\":\"v1\"}\n" {
		t.Errorf("Expected global JSON, got %q", line)
	}
}
