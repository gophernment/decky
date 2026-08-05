package main

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func init() {
	startServer = func(markdownFile, port string, stdout io.Writer) error {
		return nil
	}
	startExport = func(markdownFile, outputFile string, stdout io.Writer) error {
		return nil
	}
	startExportHTML = func(markdownFile, outputFile string, stdout io.Writer) error {
		return nil
	}
}

func runCLI(args ...string) (string, error) {
	fullArgs := append([]string{"gophern"}, args...)
	var stdout, stderr bytes.Buffer
	err := run(fullArgs, &stdout, &stderr)
	output := stdout.String() + stderr.String()
	return output, err
}

func TestCLIUsage(t *testing.T) {
	output, err := runCLI()
	if err == nil {
		t.Fatal("expected error exit code when running with no arguments, got nil")
	}
	if !strings.Contains(output, "Usage: gophern <command>") {
		t.Errorf("expected usage output, got: %s", output)
	}
}

func TestCLIServeNoFile(t *testing.T) {
	output, err := runCLI("serve")
	if err == nil {
		t.Fatal("expected error exit code when serving without file, got nil")
	}
	if !strings.Contains(output, "Error: Markdown file path required for serve command") {
		t.Errorf("expected error message for missing markdown file, got: %s", output)
	}
}

func TestCLIServeSuccess(t *testing.T) {
	output, err := runCLI("serve", "test.md")
	if err != nil {
		t.Fatalf("expected no error, got: %v (output: %s)", err, output)
	}
	if !strings.Contains(output, "Serving test.md on port 8080...") {
		t.Errorf("expected serving message, got: %s", output)
	}
}

func TestCLIServeCustomPort(t *testing.T) {
	output, err := runCLI("serve", "-port", "9090", "test.md")
	if err != nil {
		t.Fatalf("expected no error, got: %v (output: %s)", err, output)
	}
	if !strings.Contains(output, "Serving test.md on port 9090...") {
		t.Errorf("expected serving message with custom port, got: %s", output)
	}
}

func TestCLIServePortFromEnv(t *testing.T) {
	t.Setenv("PORT", "3000")
	output, err := runCLI("serve", "test.md")
	if err != nil {
		t.Fatalf("expected no error, got: %v (output: %s)", err, output)
	}
	if !strings.Contains(output, "Serving test.md on port 3000...") {
		t.Errorf("expected serving message using PORT env var, got: %s", output)
	}
}

func TestCLIServeFlagOverridesEnv(t *testing.T) {
	t.Setenv("PORT", "3000")
	output, err := runCLI("serve", "-port", "9090", "test.md")
	if err != nil {
		t.Fatalf("expected no error, got: %v (output: %s)", err, output)
	}
	if !strings.Contains(output, "Serving test.md on port 9090...") {
		t.Errorf("expected -port flag to override PORT env var, got: %s", output)
	}
}

func TestCLIExportNoFile(t *testing.T) {
	output, err := runCLI("export")
	if err == nil {
		t.Fatal("expected error exit code when exporting without file, got nil")
	}
	if !strings.Contains(output, "Error: Markdown file path required for export command") {
		t.Errorf("expected error message for missing markdown file, got: %s", output)
	}
}

func TestCLIExportSuccess(t *testing.T) {
	output, err := runCLI("export", "test.md")
	if err != nil {
		t.Fatalf("expected no error, got: %v (output: %s)", err, output)
	}
	if !strings.Contains(output, "Exporting test.md to presentation.pdf...") {
		t.Errorf("expected export message, got: %s", output)
	}
}

func TestCLIExportCustomOutput(t *testing.T) {
	output, err := runCLI("export", "-o", "out.pdf", "test.md")
	if err != nil {
		t.Fatalf("expected no error, got: %v (output: %s)", err, output)
	}
	if !strings.Contains(output, "Exporting test.md to out.pdf...") {
		t.Errorf("expected export message with custom output path, got: %s", output)
	}
}

func TestCLIHTMLNoFile(t *testing.T) {
	output, err := runCLI("html")
	if err == nil {
		t.Fatal("expected error exit code when exporting without file, got nil")
	}
	if !strings.Contains(output, "Error: Markdown file path required for html command") {
		t.Errorf("expected error message for missing markdown file, got: %s", output)
	}
}

func TestCLIHTMLSuccess(t *testing.T) {
	output, err := runCLI("html", "test.md")
	if err != nil {
		t.Fatalf("expected no error, got: %v (output: %s)", err, output)
	}
	if !strings.Contains(output, "Exporting test.md to presentation.html...") {
		t.Errorf("expected export message, got: %s", output)
	}
}

func TestCLIHTMLCustomOutput(t *testing.T) {
	output, err := runCLI("html", "-o", "out.html", "test.md")
	if err != nil {
		t.Fatalf("expected no error, got: %v (output: %s)", err, output)
	}
	if !strings.Contains(output, "Exporting test.md to out.html...") {
		t.Errorf("expected export message with custom output path, got: %s", output)
	}
}

func TestCLIUsageCommand(t *testing.T) {
	dir := t.TempDir()
	outPath := filepath.Join(dir, "USAGE.md")

	output, err := runCLI("usage", "-o", outPath)
	if err != nil {
		t.Fatalf("expected no error, got: %v (output: %s)", err, output)
	}
	if !strings.Contains(output, "Wrote "+outPath) {
		t.Errorf("expected confirmation message, got: %s", output)
	}

	written, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("expected file to be written: %v", err)
	}
	if !strings.Contains(string(written), "# Gophern Usage Guide") {
		t.Errorf("expected written file to contain the usage guide's title, got %d bytes starting with: %.80s", len(written), written)
	}
}

func TestCLIUsageCommandDefaultOutput(t *testing.T) {
	dir := t.TempDir()
	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(oldwd)

	output, err := runCLI("usage")
	if err != nil {
		t.Fatalf("expected no error, got: %v (output: %s)", err, output)
	}
	if !strings.Contains(output, "Wrote USAGE.md") {
		t.Errorf("expected default-output confirmation message, got: %s", output)
	}
	if _, err := os.Stat(filepath.Join(dir, "USAGE.md")); err != nil {
		t.Errorf("expected USAGE.md to be written to cwd: %v", err)
	}
}
