package parser_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gophernment/decky/internal/parser"
)

func TestFragmentsAppliedToRegions(t *testing.T) {
	md := `---
title: Test
---
layout: "split-h"
ratio: "50/50"
fragments: true
---
# Split Slide with Fragments

::left::

### Left
- Alpha
- Beta

::right::

### Right
- Gamma
- Delta
`
	dir := t.TempDir()
	mdPath := filepath.Join(dir, "deck.md")
	if err := os.WriteFile(mdPath, []byte(md), 0o644); err != nil {
		t.Fatal(err)
	}

	pres, err := parser.ParseMarkdownFile(mdPath)
	if err != nil {
		t.Fatal(err)
	}

	slide := pres.Slides[0]
	if !slide.Fragments {
		t.Fatal("expected slide.Fragments to be true")
	}
	if len(slide.Regions) != 2 {
		t.Fatalf("expected 2 regions, got %d", len(slide.Regions))
	}
	for name, html := range slide.Regions {
		if !strings.Contains(html, `class="fragment"`) {
			t.Errorf("region %q missing fragment class, html: %s", name, html)
		}
		// Ensure "fragments: true" text doesn't leak into rendered HTML
		if strings.Contains(html, "fragments: true") {
			t.Errorf("region %q contains 'fragments: true' text leak, html: %s", name, html)
		}
	}
}
