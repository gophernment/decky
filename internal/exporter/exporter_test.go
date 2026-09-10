package exporter_test

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/gophernment/decky/internal/exporter"
)

func TestExport_ProducesPDFWithOnePagePerSlide(t *testing.T) {
	if !exporter.ChromeAvailableForTest() {
		t.Skip("no local Chrome/Chromium found, skipping PDF export test")
	}

	dir := t.TempDir()
	mdPath := filepath.Join(dir, "deck.md")
	assetDir := filepath.Join(dir, "asset")
	if err := os.Mkdir(assetDir, 0o755); err != nil {
		t.Fatalf("failed to create asset dir: %v", err)
	}
	// 1x1 red pixel PNG
	pngBytes := []byte{
		0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a, 0x00, 0x00, 0x00, 0x0d,
		0x49, 0x48, 0x44, 0x52, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01,
		0x08, 0x02, 0x00, 0x00, 0x00, 0x90, 0x77, 0x53, 0xde, 0x00, 0x00, 0x00,
		0x0c, 0x49, 0x44, 0x41, 0x54, 0x08, 0xd7, 0x63, 0xf8, 0xcf, 0xc0, 0x00,
		0x00, 0x00, 0x03, 0x00, 0x01, 0x18, 0xdd, 0x8d, 0xb0, 0x00, 0x00, 0x00,
		0x00, 0x49, 0x45, 0x4e, 0x44, 0xae, 0x42, 0x60, 0x82,
	}
	if err := os.WriteFile(filepath.Join(assetDir, "pic.png"), pngBytes, 0o644); err != nil {
		t.Fatalf("failed to write test asset: %v", err)
	}

	content := `---
title: Exported Test Deck
author: Gopher
theme: slate
---
# Slide 1
![pic](asset/pic.png)
---
# Slide 2
Here is a code block:
` + "```go\npackage main\n```" + `
---
# Slide 3
Final slide.
`
	if err := os.WriteFile(mdPath, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write markdown: %v", err)
	}

	outPath := filepath.Join(dir, "out.pdf")
	if err := exporter.Export(mdPath, outPath, io.Discard); err != nil {
		t.Fatalf("export failed: %v", err)
	}

	pdfBytes, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("failed to read output pdf: %v", err)
	}
	if !bytes.HasPrefix(pdfBytes, []byte("%PDF-")) {
		t.Fatalf("output is not a PDF file")
	}
	if len(pdfBytes) < 1024 {
		t.Errorf("output pdf suspiciously small (%d bytes), image content likely missing", len(pdfBytes))
	}

	// No leftover temp HTML file should remain next to the source markdown.
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("failed to read dir: %v", err)
	}
	for _, e := range entries {
		if filepath.Ext(e.Name()) == ".html" {
			t.Errorf("leftover temp HTML file not cleaned up: %s", e.Name())
		}
	}
}
