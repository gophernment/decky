package exporter_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gophernment/decky/internal/exporter"
)

func TestExportHTML_ProducesSelfContainedFile(t *testing.T) {
	dir := t.TempDir()
	mdPath := filepath.Join(dir, "deck.md")

	content := `---
title: Exported Test Deck
fonts:
  sans: 'Space Grotesk'
showControls: true
showSlideNumber: true
---
# Slide 1
Hello world.
---
fragments: true
---
# Slide 2
- Alpha
- Beta
`
	if err := os.WriteFile(mdPath, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write markdown: %v", err)
	}

	outPath := filepath.Join(dir, "out.html")
	if err := exporter.ExportHTML(mdPath, outPath); err != nil {
		t.Fatalf("ExportHTML failed: %v", err)
	}

	htmlBytes, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("failed to read output html: %v", err)
	}
	html := string(htmlBytes)

	if !strings.Contains(html, "<title>Exported Test Deck</title>") {
		t.Errorf("expected title in output, got: %s", html)
	}
	if !strings.Contains(html, "Hello world.") {
		t.Errorf("expected slide 1 content in output")
	}
	if !strings.Contains(html, `class="fragment" data-fragment-index="0"`) {
		t.Errorf("expected fragment classing to carry through to the export, got: %s", html)
	}
	// CSS and JS must be inlined (self-contained), not linked externally.
	if strings.Contains(html, `href="/static/css`) || strings.Contains(html, `src="/static/js`) {
		t.Errorf("expected inlined CSS/JS, found external /static/ reference")
	}
	if !strings.Contains(html, "<style>") || !strings.Contains(html, "<script>") {
		t.Errorf("expected inline <style> and <script> blocks in output")
	}
	// showControls/showSlideNumber true must render the nav buttons/number.
	if !strings.Contains(html, `id="btn-prev"`) || !strings.Contains(html, `id="slide-number"`) {
		t.Errorf("expected nav controls and slide number to render when enabled")
	}
	if !strings.Contains(html, `id="btn-fullscreen"`) {
		t.Errorf("expected fullscreen button to always render")
	}
}

func TestExportHTML_ControlsHiddenByDefault(t *testing.T) {
	dir := t.TempDir()
	mdPath := filepath.Join(dir, "deck.md")
	if err := os.WriteFile(mdPath, []byte("# Only slide\n"), 0o644); err != nil {
		t.Fatalf("failed to write markdown: %v", err)
	}

	outPath := filepath.Join(dir, "out.html")
	if err := exporter.ExportHTML(mdPath, outPath); err != nil {
		t.Fatalf("ExportHTML failed: %v", err)
	}

	html, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("failed to read output html: %v", err)
	}
	if strings.Contains(string(html), `id="btn-prev"`) || strings.Contains(string(html), `id="slide-number"`) {
		t.Errorf("expected nav controls/slide number hidden by default, got: %s", html)
	}
}

func TestExportHTML_HeadingColors(t *testing.T) {
	dir := t.TempDir()
	mdPath := filepath.Join(dir, "deck.md")

	content := `---
title: Heading Colors Export Test
headingColors:
  h1: "linear-gradient(90deg, #f472b6, #60a5fa)"
  h3: "#a3e635"
---
# Slide 1
### A level-3 heading
`
	if err := os.WriteFile(mdPath, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write markdown: %v", err)
	}

	outPath := filepath.Join(dir, "out.html")
	if err := exporter.ExportHTML(mdPath, outPath); err != nil {
		t.Fatalf("ExportHTML failed: %v", err)
	}

	htmlBytes, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("failed to read output html: %v", err)
	}
	html := string(htmlBytes)

	styleIdx := strings.Index(html, "#slide-0 h1 { background: linear-gradient(90deg, #f472b6, #60a5fa);")
	if styleIdx == -1 {
		t.Fatalf("expected scoped gradient CSS rule for slide 0 h1 in output, got: %s", html)
	}
	divIdx := strings.Index(html, `id="slide-0"`)
	if divIdx == -1 || styleIdx > divIdx {
		t.Errorf("expected the <style> block to appear before the slide div, styleIdx=%d divIdx=%d", styleIdx, divIdx)
	}
	if !strings.Contains(html, "#slide-0 h3 { color: #a3e635; -webkit-text-fill-color: #a3e635; }") {
		t.Errorf("expected scoped solid-color CSS rule for slide 0 h3, got: %s", html)
	}
}
