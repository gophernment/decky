package server

import (
	"strings"
	"testing"
)

func TestStartupMessage_ShowsBothExportHints(t *testing.T) {
	msg := startupMessage("http://localhost:3325/", "http://localhost:3325/presenter", "test.md")

	if !strings.Contains(msg, "decky export -o output.pdf test.md") {
		t.Errorf("expected PDF export hint, got: %s", msg)
	}
	if !strings.Contains(msg, "decky html -o output.html test.md") {
		t.Errorf("expected HTML export hint, got: %s", msg)
	}
	if strings.Contains(msg, "decky export -o output.html") {
		t.Errorf("expected the old incorrect hint (export producing .html) to be gone, got: %s", msg)
	}
}
