# Heading Colors & Gradients Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Let users set a solid color or CSS gradient per heading level (`h1`-`h4`) via a new `headingColors` frontmatter key, with a deck-wide default (global) and per-slide override (local).

**Architecture:** Add `HeadingColors map[string]string` to both `Presentation` (global) and `Slide` (local) in `internal/parser/parser.go`. After parsing, merge global+local per level and precompute a scoped CSS string (`Slide.HeadingColorCSS`) using the slide's existing `#slide-N` DOM id. The shared `web/templates/_slide.html` partial emits this as an inline `<style>` block, so `serve`/`export`/`html` all pick it up with zero duplication. Add base `.slide h3`/`.slide h4` CSS rules since none exist today.

**Tech Stack:** Go 1.x, `gopkg.in/yaml.v3`, `html/template`, existing `goldmark`-based markdown pipeline. No new dependencies.

## Global Constraints

- No CSS value validation — `headingColors` values are raw passthrough, matching the existing trust model of `color`/`background` (spec: "Gradient detection" section).
- Recognized levels are exactly `h1`, `h2`, `h3`, `h4`; unrecognized keys are silently ignored, no error.
- Gradient detection: a value is a gradient iff it contains the substring `gradient(`, case-insensitive.
- Merge is per-key: a level set in a slide's local `headingColors` overrides only that level; every other level falls back to the global (Slide 0) value.
- No changes needed to `internal/server/server.go`, `internal/exporter/html.go`, or `internal/exporter/exporter.go` — the CSS is fully precomputed in the parser and interpolated as a plain value in the shared template, no new template func required.

Spec: `docs/superpowers/specs/2026-08-05-heading-colors-design.md`

---

### Task 1: Data model — `headingColors` frontmatter parsing

**Files:**
- Modify: `internal/parser/parser.go:55-83` (`Presentation` struct), `:92-117` (`Slide` struct), `:144-167` (global frontmatter parsing), `:432-457` (`coreFrontmatterKeys`)
- Test: `internal/parser/parser_test.go`

**Interfaces:**
- Produces: `Presentation.HeadingColors map[string]string` (deck-wide default, populated from the file's global frontmatter block), `Slide.HeadingColors map[string]string` (per-slide override, populated from that slide's own frontmatter block). Both use yaml tag `headingColors`. Task 2 consumes both.

- [ ] **Step 1: Write the failing test**

Add to `internal/parser/parser_test.go`:

```go
func TestHeadingColorsParsing(t *testing.T) {
	t.Run("global headingColors is parsed onto the presentation", func(t *testing.T) {
		tmpFile, err := os.CreateTemp("", "slides-heading-colors-global-*.md")
		if err != nil {
			t.Fatal(err)
		}
		defer os.Remove(tmpFile.Name())

		content := `---
title: Heading Color Test
headingColors:
  h1: "#22d3ee"
  h2: "linear-gradient(90deg, #f472b6, #60a5fa)"
---
# Slide 1

---
# Slide 2
`
		if _, err := tmpFile.WriteString(content); err != nil {
			t.Fatal(err)
		}
		tmpFile.Close()

		pres, err := ParseMarkdownFile(tmpFile.Name())
		if err != nil {
			t.Fatalf("ParseMarkdownFile failed: %v", err)
		}
		if pres.HeadingColors["h1"] != "#22d3ee" {
			t.Errorf("Expected pres.HeadingColors[h1] = #22d3ee, got %q", pres.HeadingColors["h1"])
		}
		if pres.HeadingColors["h2"] != "linear-gradient(90deg, #f472b6, #60a5fa)" {
			t.Errorf("Expected pres.HeadingColors[h2] to be the gradient, got %q", pres.HeadingColors["h2"])
		}
		if len(pres.Slides) != 2 {
			t.Fatalf("Expected 2 slides, got %d", len(pres.Slides))
		}
	})

	t.Run("local headingColors is parsed onto only that slide", func(t *testing.T) {
		tmpFile, err := os.CreateTemp("", "slides-heading-colors-local-*.md")
		if err != nil {
			t.Fatal(err)
		}
		defer os.Remove(tmpFile.Name())

		content := `---
title: Heading Color Test
headingColors:
  h1: "#22d3ee"
  h2: "#a3e635"
---
# Slide 1

---
headingColors:
  h1: "#facc15"
---
# Slide 2
`
		if _, err := tmpFile.WriteString(content); err != nil {
			t.Fatal(err)
		}
		tmpFile.Close()

		pres, err := ParseMarkdownFile(tmpFile.Name())
		if err != nil {
			t.Fatalf("ParseMarkdownFile failed: %v", err)
		}
		if len(pres.Slides) != 2 {
			t.Fatalf("Expected 2 slides, got %d", len(pres.Slides))
		}
		if len(pres.Slides[0].HeadingColors) != 0 {
			t.Errorf("Expected slide 0 to have no local headingColors, got %+v", pres.Slides[0].HeadingColors)
		}
		if pres.Slides[1].HeadingColors["h1"] != "#facc15" {
			t.Errorf("Expected slide 1 local HeadingColors[h1] = #facc15, got %q", pres.Slides[1].HeadingColors["h1"])
		}
		if _, ok := pres.Slides[1].HeadingColors["h2"]; ok {
			t.Errorf("Expected slide 1 local HeadingColors to not set h2, got %q", pres.Slides[1].HeadingColors["h2"])
		}
	})
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/parser/... -run TestHeadingColorsParsing -v`
Expected: FAIL — compile error, `pres.HeadingColors` / `slide.HeadingColors` undefined (fields don't exist yet).

- [ ] **Step 3: Add the struct fields, global-config threading, and allowlist entry**

In `internal/parser/parser.go`, add to the `Presentation` struct (after the `Fonts` field, around line 60):

```go
	Fonts       FontsConfig `yaml:"fonts"`

	// HeadingColors sets deck-wide default colors/gradients per heading
	// level (h1-h4). A slide's own headingColors overrides individual
	// levels; levels it doesn't set fall back to this deck-wide default.
	// See Slide.HeadingColors and mergeHeadingColors (Task 2).
	HeadingColors map[string]string `yaml:"headingColors"`

	Slides      []Slide
```

Add to the `Slide` struct (after the `HeaderFont` field, around line 111):

```go
	HeaderFont string `yaml:"headerFont"`

	// HeadingColors overrides the deck-wide default (Presentation.HeadingColors)
	// per heading level, for this slide only. See mergeHeadingColors (Task 2).
	HeadingColors map[string]string `yaml:"headingColors"`
```

In `ParseMarkdownFile`, inside the global-config block (around line 159-164, right after the `Fonts.Mono` copy), add:

```go
					if globalConfig.Fonts.Mono != "" {
						pres.Fonts.Mono = globalConfig.Fonts.Mono
					}
					if len(globalConfig.HeadingColors) > 0 {
						pres.HeadingColors = globalConfig.HeadingColors
					}
```

Add `"headingColors": true` to `coreFrontmatterKeys` (around line 452, next to `"headerFont"`):

```go
	"headerFont":      true,
	"headingColors":   true,
	"fragments":       true,
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/parser/... -run TestHeadingColorsParsing -v`
Expected: PASS

- [ ] **Step 5: Run the full parser test suite to check for regressions**

Run: `go test ./internal/parser/...`
Expected: PASS (all existing tests still pass — struct additions are additive, no existing field renamed or removed)

- [ ] **Step 6: Commit**

```bash
git add internal/parser/parser.go internal/parser/parser_test.go
git commit -m "feat: parse headingColors frontmatter (global + per-slide)"
```

---

### Task 2: Merge + gradient detection + scoped CSS generation

**Files:**
- Modify: `internal/parser/parser.go` (imports, `Slide` struct, new functions, `ParseMarkdownFile` loop)
- Test: `internal/parser/parser_test.go`

**Interfaces:**
- Consumes: `Presentation.HeadingColors map[string]string`, `Slide.HeadingColors map[string]string` (Task 1).
- Produces: `Slide.HeadingColorCSS template.CSS` — precomputed, scoped CSS text (empty string if the slide has no effective heading colors). Task 3 consumes this field directly in the shared template.
- Produces (unexported, package-internal): `mergeHeadingColors(global, local map[string]string) map[string]string`, `isGradientValue(value string) bool`, `buildHeadingColorCSS(slideIndex int, merged map[string]string) template.CSS`.

- [ ] **Step 1: Write the failing tests**

Add to `internal/parser/parser_test.go`:

```go
func TestMergeHeadingColors(t *testing.T) {
	global := map[string]string{"h1": "#111111", "h2": "#222222"}
	local := map[string]string{"h1": "#facc15"}

	merged := mergeHeadingColors(global, local)

	if merged["h1"] != "#facc15" {
		t.Errorf("Expected local h1 to win, got %q", merged["h1"])
	}
	if merged["h2"] != "#222222" {
		t.Errorf("Expected h2 to fall back to global, got %q", merged["h2"])
	}
	if _, ok := merged["h3"]; ok {
		t.Errorf("Expected h3 to be absent when unset in both maps, got %q", merged["h3"])
	}
}

func TestBuildHeadingColorCSS(t *testing.T) {
	t.Run("empty merged map returns empty CSS", func(t *testing.T) {
		if css := buildHeadingColorCSS(3, map[string]string{}); css != "" {
			t.Errorf("Expected empty CSS, got %q", css)
		}
	})

	t.Run("solid color emits a color rule scoped to the slide id", func(t *testing.T) {
		css := string(buildHeadingColorCSS(3, map[string]string{"h2": "#22d3ee"}))
		want := "#slide-3 h2 { color: #22d3ee; }"
		if !strings.Contains(css, want) {
			t.Errorf("Expected CSS to contain %q, got %q", want, css)
		}
	})

	t.Run("gradient value emits background-clip:text rules", func(t *testing.T) {
		css := string(buildHeadingColorCSS(0, map[string]string{"h1": "linear-gradient(90deg, #f472b6, #60a5fa)"}))
		for _, want := range []string{
			"#slide-0 h1 { background: linear-gradient(90deg, #f472b6, #60a5fa);",
			"-webkit-background-clip: text",
			"background-clip: text",
			"color: transparent",
			"-webkit-text-fill-color: transparent",
		} {
			if !strings.Contains(css, want) {
				t.Errorf("Expected CSS to contain %q, got %q", want, css)
			}
		}
	})

	t.Run("gradient detection is case-insensitive", func(t *testing.T) {
		css := string(buildHeadingColorCSS(0, map[string]string{"h3": "RADIAL-GRADIENT(circle, red, blue)"}))
		if !strings.Contains(css, "background: RADIAL-GRADIENT(circle, red, blue);") {
			t.Errorf("Expected gradient technique for uppercase gradient value, got %q", css)
		}
	})

	t.Run("rules are emitted in h1,h2,h3,h4 order regardless of map iteration order", func(t *testing.T) {
		css := string(buildHeadingColorCSS(0, map[string]string{"h4": "#fbbf24", "h1": "#f472b6", "h3": "#a3e635"}))
		i1 := strings.Index(css, "#slide-0 h1")
		i3 := strings.Index(css, "#slide-0 h3")
		i4 := strings.Index(css, "#slide-0 h4")
		if !(i1 < i3 && i3 < i4) {
			t.Errorf("Expected h1 < h3 < h4 order in output, got %q", css)
		}
	})
}

func TestHeadingColorsMergeIntoSlideCSS(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "slides-heading-colors-merge-*.md")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())

	content := `---
title: Heading Color Merge Test
headingColors:
  h1: "#22d3ee"
  h2: "#a3e635"
---
# Slide 1

---
headingColors:
  h1: "linear-gradient(90deg, #f472b6, #60a5fa)"
---
# Slide 2
`
	if _, err := tmpFile.WriteString(content); err != nil {
		t.Fatal(err)
	}
	tmpFile.Close()

	pres, err := ParseMarkdownFile(tmpFile.Name())
	if err != nil {
		t.Fatalf("ParseMarkdownFile failed: %v", err)
	}
	if len(pres.Slides) != 2 {
		t.Fatalf("Expected 2 slides, got %d", len(pres.Slides))
	}

	css0 := string(pres.Slides[0].HeadingColorCSS)
	if !strings.Contains(css0, "#slide-0 h1 { color: #22d3ee; }") {
		t.Errorf("Expected slide 0 h1 solid color rule, got %q", css0)
	}
	if !strings.Contains(css0, "#slide-0 h2 { color: #a3e635; }") {
		t.Errorf("Expected slide 0 h2 solid color rule, got %q", css0)
	}

	css1 := string(pres.Slides[1].HeadingColorCSS)
	if !strings.Contains(css1, "#slide-1 h1 { background: linear-gradient(90deg, #f472b6, #60a5fa);") {
		t.Errorf("Expected slide 1 h1 gradient rule (local override), got %q", css1)
	}
	if !strings.Contains(css1, "#slide-1 h2 { color: #a3e635; }") {
		t.Errorf("Expected slide 1 h2 to fall back to the global value, got %q", css1)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/parser/... -run 'TestMergeHeadingColors|TestBuildHeadingColorCSS|TestHeadingColorsMergeIntoSlideCSS' -v`
Expected: FAIL — compile error, `mergeHeadingColors`/`buildHeadingColorCSS`/`Slide.HeadingColorCSS` undefined.

- [ ] **Step 3: Implement the merge, detection, and CSS builder**

Add `"fmt"` and `"html/template"` to the import block at the top of `internal/parser/parser.go` (alongside the existing `"bytes"`, `"math"`, etc.):

```go
import (
	"bytes"
	"fmt"
	"html/template"
	"math"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"

	chromahtml "github.com/alecthomas/chroma/v2/formatters/html"
	...
```

Add to the `Slide` struct, right after `HeadingColors`:

```go
	// HeadingColors overrides the deck-wide default (Presentation.HeadingColors)
	// per heading level, for this slide only. See mergeHeadingColors.
	HeadingColors map[string]string `yaml:"headingColors"`

	// HeadingColorCSS is computed (not user-set) from the merge of
	// Presentation.HeadingColors and this slide's own HeadingColors. It is
	// a scoped CSS rule set (one rule per heading level with a color or
	// gradient set), emitted as an inline <style> block by
	// web/templates/_slide.html. Empty if no heading colors apply.
	HeadingColorCSS template.CSS
```

Add these functions near `buildGoogleFontsURL` (after it, before `splitBySeparator`, around line 320):

```go
// headingLevels are the heading levels headingColors recognizes, in the
// order their CSS rules are emitted.
var headingLevels = []string{"h1", "h2", "h3", "h4"}

// mergeHeadingColors merges a slide's local headingColors over the deck's
// global headingColors, per level: a level set locally wins; otherwise the
// global value (if any) is used. Levels absent from both maps are omitted
// from the result.
func mergeHeadingColors(global, local map[string]string) map[string]string {
	merged := make(map[string]string, len(headingLevels))
	for _, level := range headingLevels {
		if v, ok := local[level]; ok && v != "" {
			merged[level] = v
			continue
		}
		if v, ok := global[level]; ok && v != "" {
			merged[level] = v
		}
	}
	return merged
}

// isGradientValue reports whether a headingColors value is a CSS gradient
// (linear-gradient, radial-gradient, conic-gradient, or their repeating-*
// variants) rather than a solid color.
func isGradientValue(value string) bool {
	return strings.Contains(strings.ToLower(value), "gradient(")
}

// buildHeadingColorCSS generates a scoped CSS rule set for a slide's merged
// headingColors, one rule per level, scoped to that slide's DOM id
// (#slide-N, matching web/templates/_slide.html's `id="slide-{{ .Index }}"`).
// Solid colors set `color`; gradients use the background-clip:text
// technique so the gradient paints the glyph shapes instead of a solid
// background box. Returns "" if merged is empty.
func buildHeadingColorCSS(slideIndex int, merged map[string]string) template.CSS {
	if len(merged) == 0 {
		return ""
	}
	var b strings.Builder
	for _, level := range headingLevels {
		value, ok := merged[level]
		if !ok {
			continue
		}
		fmt.Fprintf(&b, "#slide-%d %s { ", slideIndex, level)
		if isGradientValue(value) {
			fmt.Fprintf(&b, "background: %s; -webkit-background-clip: text; background-clip: text; color: transparent; -webkit-text-fill-color: transparent;", value)
		} else {
			fmt.Fprintf(&b, "color: %s;", value)
		}
		b.WriteString(" }\n")
	}
	return template.CSS(b.String())
}
```

In `ParseMarkdownFile`'s slide loop, right before `pres.Slides = append(pres.Slides, slide)` (around line 265), add:

```go
		merged := mergeHeadingColors(pres.HeadingColors, slide.HeadingColors)
		slide.HeadingColorCSS = buildHeadingColorCSS(slide.Index, merged)

		pres.Slides = append(pres.Slides, slide)
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/parser/... -run 'TestMergeHeadingColors|TestBuildHeadingColorCSS|TestHeadingColorsMergeIntoSlideCSS' -v`
Expected: PASS

- [ ] **Step 5: Run the full parser test suite to check for regressions**

Run: `go test ./internal/parser/...`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add internal/parser/parser.go internal/parser/parser_test.go
git commit -m "feat: merge headingColors and precompute scoped CSS per slide"
```

---

### Task 3: Template wiring + base h3/h4 CSS + end-to-end render test

**Files:**
- Modify: `web/templates/_slide.html:5-6`, `web/static/css/styles.css` (after the `.slide h1` block, around line 95)
- Test: `internal/exporter/html_test.go`

**Interfaces:**
- Consumes: `Slide.HeadingColorCSS template.CSS` (Task 2), `Slide.Index int` (existing).
- Produces: nothing new consumed by later tasks — this is the user-visible rendering surface.

- [ ] **Step 1: Write the failing test**

Add to `internal/exporter/html_test.go`:

```go
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
	if !strings.Contains(html, "#slide-0 h3 { color: #a3e635; }") {
		t.Errorf("expected scoped solid-color CSS rule for slide 0 h3, got: %s", html)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/exporter/... -run TestExportHTML_HeadingColors -v`
Expected: FAIL — output contains no `<style>#slide-0 h1 { background: linear-gradient...` because `_slide.html` doesn't emit `HeadingColorCSS` yet.

- [ ] **Step 3: Wire the template**

In `web/templates/_slide.html`, change:

```html
{{ define "slideBody" }}
<div id="slide-{{ .Index }}" class="slide {{ if .Layout }}{{ .Layout }}{{ else }}default{{ end }}"
```

to:

```html
{{ define "slideBody" }}
{{ if .HeadingColorCSS }}<style>{{ .HeadingColorCSS }}</style>{{ end }}
<div id="slide-{{ .Index }}" class="slide {{ if .Layout }}{{ .Layout }}{{ else }}default{{ end }}"
```

- [ ] **Step 4: Add base h3/h4 CSS rules**

In `web/static/css/styles.css`, right after the `.slide h1 { ... }` block (ends around line 95, right before the `/* Slide Transition States */` comment), add:

```css
.slide h3 {
  font-size: 1.75rem;
  font-weight: 700;
  margin-top: 0;
  margin-bottom: 1rem;
  color: var(--slide-text-color, var(--text-primary));
  font-family: var(--font-heading, var(--font-sans));
}

.slide h4 {
  font-size: 1.35rem;
  font-weight: 600;
  margin-top: 0;
  margin-bottom: 0.75rem;
  color: var(--slide-text-color, var(--text-primary));
  font-family: var(--font-heading, var(--font-sans));
}
```

- [ ] **Step 5: Run test to verify it passes**

Run: `go test ./internal/exporter/... -run TestExportHTML_HeadingColors -v`
Expected: PASS

- [ ] **Step 6: Run the full test suite to check for regressions**

Run: `go build ./... && go test ./...`
Expected: PASS — building confirms the template/CSS edits didn't break anything (Go embeds these via `embed.FS` at build time), and the full suite (parser, exporter, server) passes.

- [ ] **Step 7: Commit**

```bash
git add web/templates/_slide.html web/static/css/styles.css internal/exporter/html_test.go
git commit -m "feat: render headingColors as scoped CSS on every slide renderer"
```

---

### Task 4: Document `headingColors` in USAGE.md

**Files:**
- Modify: `USAGE.md` (insert a new numbered section before the closing slide, after "20. Fragments" which ends around line 581)

**Interfaces:**
- Consumes: nothing (documentation only).
- Produces: nothing (terminal task).

- [ ] **Step 1: Insert the new section**

In `USAGE.md`, right after the fragments-explanation HTML comment that closes section 20 (ends at what is currently line 581, `-->`) and before the blank line + `---` that starts the closing cover slide's frontmatter, insert:

```markdown

---
headingColors:
  h1: "linear-gradient(90deg, #f472b6, #60a5fa)"
  h2: "#22d3ee"
  h3: "#a3e635"
  h4: "#fbbf24"
background: "#1e293b"
color: "#f8fafc"
---

# 21. Heading Colors & Gradients
## A solid cyan H2
### A solid green H3
#### A solid amber H4

Give each heading level (`h1`-`h4`) its own color or gradient with
`headingColors` frontmatter — set deck-wide defaults on Slide 0, override
per level on any later slide (levels you don't list there keep the deck
default).

```markdown
---
headingColors:
  h1: "linear-gradient(90deg, #f472b6, #60a5fa)"
  h2: "#22d3ee"
---
```

<!--
Any value containing "gradient(" (linear-gradient, radial-gradient,
conic-gradient, case-insensitive) renders via the background-clip:text
technique; anything else is treated as a solid CSS color. Merge is
per-level: a level set in a slide's own headingColors overrides just that
level, every other level still falls back to whatever was set globally on
Slide 0.
-->
```

- [ ] **Step 2: Verify the deck still parses and serves**

Run: `go run . serve USAGE.md` (or `go build -o /tmp/gophern . && /tmp/gophern serve USAGE.md`), open the local URL in a browser, navigate to the new "21. Heading Colors & Gradients" slide.
Expected: H1 renders as a pink-to-blue gradient, H2 cyan, H3 green, H4 amber — no console errors, no broken layout on adjacent slides (20 and the closing slide still render correctly).

- [ ] **Step 3: Commit**

```bash
git add USAGE.md
git commit -m "docs: add headingColors usage example"
```
