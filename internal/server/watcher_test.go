package server

import (
	"bufio"
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

func TestWatchFile(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "test_watch_file_*.md")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	_, err = tmpFile.WriteString("# Slide 1\nInitial content\n")
	if err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}
	tmpFile.Close()

	srv := NewServer(tmpFile.Name())
	// Start the file watcher
	go srv.watchFile()

	testServer := httptest.NewServer(srv.Router())
	defer testServer.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", testServer.URL+"/events", nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer resp.Body.Close()

	reader := bufio.NewReader(resp.Body)

	// Consume initial handshake/slide events
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			t.Fatalf("failed reading stream: %v", err)
		}
		if line == "\n" {
			break
		}
	}

	// Now modify the file
	time.Sleep(100 * time.Millisecond)     // wait a bit to avoid mod time conflict
	now := time.Now().Add(5 * time.Second) // future mod time to ensure watcher detects change
	if err := os.Chtimes(tmpFile.Name(), now, now); err != nil {
		t.Fatalf("failed to touch file: %v", err)
	}

	// Expect data: {"reload":true}
	reloadReceived := false
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			break
		}
		if strings.Contains(line, `{"reload":true}`) {
			reloadReceived = true
			break
		}
	}

	if !reloadReceived {
		t.Errorf("expected reload event from watcher, but did not receive it")
	}
}
