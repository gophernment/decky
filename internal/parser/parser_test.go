package parser

import (
	"os"
	"strings"
	"testing"
)

func TestParseMarkdown(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "slides-*.md")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())

	content := `---
title: Spec Test
author: Gopher
theme: slate
---
# Slide 1 Content

---
layout: cover
background: blue
---
# Slide 2 Content
`
	if _, err := tmpFile.Write([]byte(content)); err != nil {
		t.Fatal(err)
	}
	tmpFile.Close()

	pres, err := ParseMarkdownFile(tmpFile.Name())
	if err != nil {
		t.Fatalf("ParseMarkdownFile failed: %v", err)
	}

	if pres.Title != "Spec Test" {
		t.Errorf("Expected Title 'Spec Test', got '%s'", pres.Title)
	}
	if pres.Author != "Gopher" {
		t.Errorf("Expected Author 'Gopher', got '%s'", pres.Author)
	}
	if pres.Theme != "slate" {
		t.Errorf("Expected Theme 'slate', got '%s'", pres.Theme)
	}
	if len(pres.Slides) != 2 {
		t.Fatalf("Expected 2 slides, got %d", len(pres.Slides))
	}
	if pres.Slides[0].Index != 0 || pres.Slides[1].Index != 1 {
		t.Errorf("Incorrect slide indices")
	}
	if pres.Slides[1].Layout != "cover" {
		t.Errorf("Expected Layout 'cover', got '%s'", pres.Slides[1].Layout)
	}
	if pres.Slides[1].Background != "blue" {
		t.Errorf("Expected Background 'blue', got '%s'", pres.Slides[1].Background)
	}
}

func TestParseMarkdownNoGlobalFrontmatter(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "slides-*.md")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())

	content := `# Slide 1 Content

---
layout: cover
background: red
---
# Slide 2 Content
`
	if _, err := tmpFile.Write([]byte(content)); err != nil {
		t.Fatal(err)
	}
	tmpFile.Close()

	pres, err := ParseMarkdownFile(tmpFile.Name())
	if err != nil {
		t.Fatalf("ParseMarkdownFile failed: %v", err)
	}

	if pres.Title != "Presentation" { // default
		t.Errorf("Expected Title 'Presentation', got '%s'", pres.Title)
	}
	if len(pres.Slides) != 2 {
		t.Fatalf("Expected 2 slides, got %d", len(pres.Slides))
	}
	if pres.Slides[0].RawMarkdown != "# Slide 1 Content" {
		t.Errorf("Expected first slide content '# Slide 1 Content', got %q", pres.Slides[0].RawMarkdown)
	}
	if pres.Slides[1].Layout != "cover" || pres.Slides[1].Background != "red" {
		t.Errorf("Incorrect slide 2 attributes: layout=%s, bg=%s", pres.Slides[1].Layout, pres.Slides[1].Background)
	}
	if pres.Slides[1].RawMarkdown != "# Slide 2 Content" {
		t.Errorf("Expected slide 2 content '# Slide 2 Content', got %q", pres.Slides[1].RawMarkdown)
	}
}

func TestSpeakerNotes(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "slides-*.md")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())

	content := `---
title: Notes Test
---
# Slide 1

Some content here.

<!--
These are speaker notes for slide 1.
Multiple lines allowed.
-->

---
# Slide 2
No notes here.
`
	if _, err := tmpFile.Write([]byte(content)); err != nil {
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

	s1 := pres.Slides[0]
	expectedNotes := "These are speaker notes for slide 1.\nMultiple lines allowed."
	if s1.SpeakerNotes != expectedNotes {
		t.Errorf("Expected speaker notes %q, got %q", expectedNotes, s1.SpeakerNotes)
	}
	expectedMarkdown := "# Slide 1\n\nSome content here."
	if s1.RawMarkdown != expectedMarkdown {
		t.Errorf("Expected clean markdown %q, got %q", expectedMarkdown, s1.RawMarkdown)
	}

	s2 := pres.Slides[1]
	if s2.SpeakerNotes != "" {
		t.Errorf("Expected empty speaker notes for slide 2, got %q", s2.SpeakerNotes)
	}
}

func TestNonExistentFile(t *testing.T) {
	_, err := ParseMarkdownFile("does_not_exist.md")
	if err == nil {
		t.Error("Expected error for non-existent file, got nil")
	}
}

func TestParserFixes(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "slides-fixes-*.md")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())

	content := `---
title: Global Settings
layout: first-layout
background: first-bg
color: first-color
---
# First Slide

Note: test this colon

---
# Second Slide
` + "```go\n" + `
func main() {
	// Some code
	// ---
}
` + "```\n" + `
---
layout: third-layout
---
# Third Slide
`

	if _, err := tmpFile.Write([]byte(content)); err != nil {
		t.Fatal(err)
	}
	tmpFile.Close()

	pres, err := ParseMarkdownFile(tmpFile.Name())
	if err != nil {
		t.Fatalf("ParseMarkdownFile failed: %v", err)
	}

	if len(pres.Slides) != 3 {
		t.Fatalf("Expected 3 slides, got %d", len(pres.Slides))
	}

	// 1. Slide 0 global frontmatter inheritance
	s0 := pres.Slides[0]
	if s0.Layout != "first-layout" {
		t.Errorf("Expected s0 Layout 'first-layout', got '%s'", s0.Layout)
	}
	if s0.Background != "first-bg" {
		t.Errorf("Expected s0 Background 'first-bg', got '%s'", s0.Background)
	}
	if s0.Color != "first-color" {
		t.Errorf("Expected s0 Color 'first-color', got '%s'", s0.Color)
	}

	// 2. Slide content containing colons not parsed as frontmatter
	if !strings.Contains(s0.RawMarkdown, "Note: test this colon") {
		t.Errorf("Expected s0 RawMarkdown to contain 'Note: test this colon', got %q", s0.RawMarkdown)
	}

	// 3. Fenced code block with '---' inside it
	s1 := pres.Slides[1]
	if !strings.Contains(s1.RawMarkdown, "---") {
		t.Errorf("Expected s1 RawMarkdown to preserve '---' inside code block, got %q", s1.RawMarkdown)
	}

	// 4. HTMLContent conversion check
	if !strings.Contains(s0.HTMLContent, "<h1>First Slide</h1>") {
		t.Errorf("Expected s0 HTMLContent to contain '<h1>First Slide</h1>', got %q", s0.HTMLContent)
	}
}

func TestTask2ReviewIssues(t *testing.T) {
	// 1. A slide frontmatter with comments, multi-line string, and lists parses correctly.
	t.Run("frontmatter comments, multi-line, lists", func(t *testing.T) {
		tmpFile, err := os.CreateTemp("", "slides-yaml-*.md")
		if err != nil {
			t.Fatal(err)
		}
		defer os.Remove(tmpFile.Name())

		content := `# Slide 1
---
layout: slide-with-yaml-features
# This is a comment in frontmatter
comments:
  - comment 1
  - comment 2
notes: |
  This is a multi-line
  string
---
# Slide 2 Content
`
		if _, err := tmpFile.Write([]byte(content)); err != nil {
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

		s0 := pres.Slides[0]
		if s0.RawMarkdown != "# Slide 1" {
			t.Errorf("Expected s0 RawMarkdown '# Slide 1', got %q", s0.RawMarkdown)
		}

		s1 := pres.Slides[1]
		if s1.Layout != "slide-with-yaml-features" {
			t.Errorf("Expected s1 Layout 'slide-with-yaml-features', got %q", s1.Layout)
		}
		if s1.RawMarkdown != "# Slide 2 Content" {
			t.Errorf("Expected s1 RawMarkdown '# Slide 2 Content', got %q", s1.RawMarkdown)
		}
	})

	// 2. A document ending with a frontmatter-only slide (no content and no trailing separator)
	// correctly creates the last slide with that frontmatter and empty content.
	t.Run("ending with frontmatter-only slide", func(t *testing.T) {
		tmpFile, err := os.CreateTemp("", "slides-end-*.md")
		if err != nil {
			t.Fatal(err)
		}
		defer os.Remove(tmpFile.Name())

		content := `# Slide 1 Content
---
layout: end-slide
background: black`
		if _, err := tmpFile.Write([]byte(content)); err != nil {
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

		s0 := pres.Slides[0]
		if s0.RawMarkdown != "# Slide 1 Content" {
			t.Errorf("Expected s0 RawMarkdown '# Slide 1 Content', got %q", s0.RawMarkdown)
		}

		s1 := pres.Slides[1]
		if s1.Layout != "end-slide" {
			t.Errorf("Expected s1 Layout 'end-slide', got %q", s1.Layout)
		}
		if s1.Background != "black" {
			t.Errorf("Expected s1 Background 'black', got %q", s1.Background)
		}
		if s1.RawMarkdown != "" {
			t.Errorf("Expected s1 RawMarkdown to be empty, got %q", s1.RawMarkdown)
		}
	})

	// 3. Empty/blank slides are not skipped and are indexed correctly.
	t.Run("empty blank slides indexed correctly", func(t *testing.T) {
		tmpFile, err := os.CreateTemp("", "slides-empty-*.md")
		if err != nil {
			t.Fatal(err)
		}
		defer os.Remove(tmpFile.Name())

		content := `# Slide 1
---
---
# Slide 3`
		if _, err := tmpFile.Write([]byte(content)); err != nil {
			t.Fatal(err)
		}
		tmpFile.Close()

		pres, err := ParseMarkdownFile(tmpFile.Name())
		if err != nil {
			t.Fatalf("ParseMarkdownFile failed: %v", err)
		}

		if len(pres.Slides) != 3 {
			t.Fatalf("Expected 3 slides, got %d", len(pres.Slides))
		}

		s0 := pres.Slides[0]
		if s0.Index != 0 || s0.RawMarkdown != "# Slide 1" {
			t.Errorf("Expected slide 0 with content '# Slide 1', got index=%d, content=%q", s0.Index, s0.RawMarkdown)
		}

		s1 := pres.Slides[1]
		if s1.Index != 1 || s1.RawMarkdown != "" {
			t.Errorf("Expected slide 1 to be blank, got index=%d, content=%q", s1.Index, s1.RawMarkdown)
		}

		s2 := pres.Slides[2]
		if s2.Index != 2 || s2.RawMarkdown != "# Slide 3" {
			t.Errorf("Expected slide 2 with content '# Slide 3', got index=%d, content=%q", s2.Index, s2.RawMarkdown)
		}
	})

	// 4. Fenced code blocks starting with ~~~ containing --- are not split.
	t.Run("tilde fenced code block does not split", func(t *testing.T) {
		tmpFile, err := os.CreateTemp("", "slides-tilde-*.md")
		if err != nil {
			t.Fatal(err)
		}
		defer os.Remove(tmpFile.Name())

		content := `# Slide with tildes
~~~go
func hello() {
    // ---
}
~~~`
		if _, err := tmpFile.Write([]byte(content)); err != nil {
			t.Fatal(err)
		}
		tmpFile.Close()

		pres, err := ParseMarkdownFile(tmpFile.Name())
		if err != nil {
			t.Fatalf("ParseMarkdownFile failed: %v", err)
		}

		if len(pres.Slides) != 1 {
			t.Fatalf("Expected 1 slide, got %d", len(pres.Slides))
		}

		s0 := pres.Slides[0]
		if !strings.Contains(s0.RawMarkdown, "~~~go") || !strings.Contains(s0.RawMarkdown, "---") || !strings.Contains(s0.RawMarkdown, "~~~") {
			t.Errorf("Expected slide 0 RawMarkdown to contain the tilde fenced code block with '---', got %q", s0.RawMarkdown)
		}
	})

	// 5. A markdown file starting with --- but containing no global frontmatter
	// correctly parses Slide 0 without losing it.
	t.Run("starts with --- but no global frontmatter", func(t *testing.T) {
		tmpFile, err := os.CreateTemp("", "slides-no-global-*.md")
		if err != nil {
			t.Fatal(err)
		}
		defer os.Remove(tmpFile.Name())

		content := `---
# Slide 0 Content
This has no global frontmatter, but starts with ---
---
# Slide 1 Content
`
		if _, err := tmpFile.Write([]byte(content)); err != nil {
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

		s0 := pres.Slides[0]
		expectedS0Content := "# Slide 0 Content\nThis has no global frontmatter, but starts with ---"
		if s0.RawMarkdown != expectedS0Content {
			t.Errorf("Expected s0 RawMarkdown %q, got %q", expectedS0Content, s0.RawMarkdown)
		}

		s1 := pres.Slides[1]
		expectedS1Content := "# Slide 1 Content"
		if s1.RawMarkdown != expectedS1Content {
			t.Errorf("Expected s1 RawMarkdown %q, got %q", expectedS1Content, s1.RawMarkdown)
		}

		// Title should be default since there was no global frontmatter
		if pres.Title != "Presentation" {
			t.Errorf("Expected Title 'Presentation', got %q", pres.Title)
		}
	})

	// 6. An HTML comment placed inside a fenced code block is not extracted as speaker notes
	// and remains in the code block.
	t.Run("HTML comment inside fenced code block not extracted", func(t *testing.T) {
		tmpFile, err := os.CreateTemp("", "slides-comment-code-*.md")
		if err != nil {
			t.Fatal(err)
		}
		defer os.Remove(tmpFile.Name())

		content := `# Slide with Code Block
` + "```html\n" + `
<!-- comment inside code block -->
` + "```\n"
		if _, err := tmpFile.Write([]byte(content)); err != nil {
			t.Fatal(err)
		}
		tmpFile.Close()

		pres, err := ParseMarkdownFile(tmpFile.Name())
		if err != nil {
			t.Fatalf("ParseMarkdownFile failed: %v", err)
		}

		if len(pres.Slides) != 1 {
			t.Fatalf("Expected 1 slide, got %d", len(pres.Slides))
		}

		s0 := pres.Slides[0]
		if s0.SpeakerNotes != "" {
			t.Errorf("Expected no speaker notes, got %q", s0.SpeakerNotes)
		}

		if !strings.Contains(s0.RawMarkdown, "<!-- comment inside code block -->") {
			t.Errorf("Expected RawMarkdown to contain '<!-- comment inside code block -->', got %q", s0.RawMarkdown)
		}
	})

	// 7. A frontmatter block starting with a comment on the first line is correctly parsed
	t.Run("frontmatter starts with comment", func(t *testing.T) {
		tmpFile, err := os.CreateTemp("", "slides-comment-first-*.md")
		if err != nil {
			t.Fatal(err)
		}
		defer os.Remove(tmpFile.Name())

		content := `# Slide 1
---
# This is a comment on the first line
layout: center
background: red
---
# Slide 2 Content
`
		if _, err := tmpFile.Write([]byte(content)); err != nil {
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

		s1 := pres.Slides[1]
		if s1.Layout != "center" {
			t.Errorf("Expected s1 Layout 'center', got %q", s1.Layout)
		}
		if s1.Background != "red" {
			t.Errorf("Expected s1 Background 'red', got %q", s1.Background)
		}
	})

	// 8. Raw HTML tags are rendered correctly (unsafe HTML enabled in goldmark)
	t.Run("goldmark unsafe HTML rendering", func(t *testing.T) {
		tmpFile, err := os.CreateTemp("", "slides-unsafe-html-*.md")
		if err != nil {
			t.Fatal(err)
		}
		defer os.Remove(tmpFile.Name())

		content := `<div class="custom-class">Hello HTML</div>`
		if _, err := tmpFile.Write([]byte(content)); err != nil {
			t.Fatal(err)
		}
		tmpFile.Close()

		pres, err := ParseMarkdownFile(tmpFile.Name())
		if err != nil {
			t.Fatalf("ParseMarkdownFile failed: %v", err)
		}

		s0 := pres.Slides[0]
		expectedHTML := `<div class="custom-class">Hello HTML</div>`
		if !strings.Contains(s0.HTMLContent, expectedHTML) {
			t.Errorf("Expected HTMLContent to contain %q, got %q", expectedHTML, s0.HTMLContent)
		}
	})
}

func TestSplitLayoutRegions(t *testing.T) {
	t.Run("split-h with two regions and ratio", func(t *testing.T) {
		tmpFile, err := os.CreateTemp("", "slides-split-h-*.md")
		if err != nil {
			t.Fatal(err)
		}
		defer os.Remove(tmpFile.Name())

		content := `---
layout: "split-h"
ratio: "70/30"
---
# Split Slide

::left::
Left content.
- point 1
- point 2

::right::
` + "```go\nfunc main() {}\n```\n"

		if _, err := tmpFile.WriteString(content); err != nil {
			t.Fatal(err)
		}
		tmpFile.Close()

		pres, err := ParseMarkdownFile(tmpFile.Name())
		if err != nil {
			t.Fatalf("ParseMarkdownFile failed: %v", err)
		}
		if len(pres.Slides) != 1 {
			t.Fatalf("Expected 1 slide, got %d", len(pres.Slides))
		}

		s := pres.Slides[0]
		if !strings.Contains(s.HTMLContent, "<h1>Split Slide</h1>") {
			t.Errorf("Expected header HTMLContent to contain the h1, got %q", s.HTMLContent)
		}
		if len(s.Regions) != 2 {
			t.Fatalf("Expected 2 regions, got %d: %v", len(s.Regions), s.Regions)
		}
		if !strings.Contains(s.Regions["left"], "<li>point 1</li>") {
			t.Errorf("Expected left region to contain rendered list, got %q", s.Regions["left"])
		}
		if !strings.Contains(s.Regions["right"], "<span") {
			t.Errorf("Expected right region code block to be syntax-highlighted (contain <span>), got %q", s.Regions["right"])
		}
		if !strings.Contains(s.Regions["right"], ">func<") {
			t.Errorf("Expected right region to contain highlighted 'func' token, got %q", s.Regions["right"])
		}
		if !strings.Contains(s.Regions["right"], ">main<") {
			t.Errorf("Expected right region to contain highlighted 'main' token, got %q", s.Regions["right"])
		}
		if s.ColsCSS != "70fr 30fr" {
			t.Errorf("Expected ColsCSS '70fr 30fr', got %q", s.ColsCSS)
		}
		if s.RowsCSS != "" {
			t.Errorf("Expected empty RowsCSS for split-h, got %q", s.RowsCSS)
		}
	})

	t.Run("no markers renders exactly like before", func(t *testing.T) {
		tmpFile, err := os.CreateTemp("", "slides-nomarkers-*.md")
		if err != nil {
			t.Fatal(err)
		}
		defer os.Remove(tmpFile.Name())

		content := `# Plain Slide

Just a paragraph, no regions here.
`
		if _, err := tmpFile.WriteString(content); err != nil {
			t.Fatal(err)
		}
		tmpFile.Close()

		pres, err := ParseMarkdownFile(tmpFile.Name())
		if err != nil {
			t.Fatalf("ParseMarkdownFile failed: %v", err)
		}
		s := pres.Slides[0]
		if len(s.Regions) != 0 {
			t.Errorf("Expected no regions, got %v", s.Regions)
		}
		if !strings.Contains(s.HTMLContent, "<h1>Plain Slide</h1>") || !strings.Contains(s.HTMLContent, "Just a paragraph") {
			t.Errorf("Expected full content in HTMLContent, got %q", s.HTMLContent)
		}
		if s.ColsCSS != "" || s.RowsCSS != "" {
			t.Errorf("Expected empty ColsCSS/RowsCSS for non-split layout, got %q / %q", s.ColsCSS, s.RowsCSS)
		}
	})

	t.Run("duplicate region markers append", func(t *testing.T) {
		tmpFile, err := os.CreateTemp("", "slides-dup-*.md")
		if err != nil {
			t.Fatal(err)
		}
		defer os.Remove(tmpFile.Name())

		content := `---
layout: "split-h"
---
::left::
First part.

::right::
Right content.

::left::
Second part.
`
		if _, err := tmpFile.WriteString(content); err != nil {
			t.Fatal(err)
		}
		tmpFile.Close()

		pres, err := ParseMarkdownFile(tmpFile.Name())
		if err != nil {
			t.Fatalf("ParseMarkdownFile failed: %v", err)
		}
		s := pres.Slides[0]
		if !strings.Contains(s.Regions["left"], "First part") || !strings.Contains(s.Regions["left"], "Second part") {
			t.Errorf("Expected left region to contain both parts, got %q", s.Regions["left"])
		}
	})

	t.Run("marker inside fenced code block is not a region boundary", func(t *testing.T) {
		tmpFile, err := os.CreateTemp("", "slides-fenced-marker-*.md")
		if err != nil {
			t.Fatal(err)
		}
		defer os.Remove(tmpFile.Name())

		content := `---
layout: "split-h"
---
::left::
` + "```text\n::right::\n```\n" + `
::right::
Actual right content.
`
		if _, err := tmpFile.WriteString(content); err != nil {
			t.Fatal(err)
		}
		tmpFile.Close()

		pres, err := ParseMarkdownFile(tmpFile.Name())
		if err != nil {
			t.Fatalf("ParseMarkdownFile failed: %v", err)
		}
		s := pres.Slides[0]
		if len(s.Regions) != 2 {
			t.Fatalf("Expected 2 regions, got %d: %v", len(s.Regions), s.Regions)
		}
		if !strings.Contains(s.Regions["left"], "::right::") {
			t.Errorf("Expected left region to contain the literal marker text from the code block, got %q", s.Regions["left"])
		}
		if !strings.Contains(s.Regions["right"], "Actual right content") {
			t.Errorf("Expected right region content, got %q", s.Regions["right"])
		}
	})

	t.Run("invalid ratio falls back to equal split", func(t *testing.T) {
		tmpFile, err := os.CreateTemp("", "slides-badratio-*.md")
		if err != nil {
			t.Fatal(err)
		}
		defer os.Remove(tmpFile.Name())

		content := `---
layout: "split-h"
ratio: "70/20/10"
---
::left::
L

::right::
R
`
		if _, err := tmpFile.WriteString(content); err != nil {
			t.Fatal(err)
		}
		tmpFile.Close()

		pres, err := ParseMarkdownFile(tmpFile.Name())
		if err != nil {
			t.Fatalf("ParseMarkdownFile failed: %v", err)
		}
		s := pres.Slides[0]
		if s.ColsCSS != "" {
			t.Errorf("Expected empty ColsCSS for mismatched ratio part count, got %q", s.ColsCSS)
		}
	})

	t.Run("grid-4 uses independent cols and rows", func(t *testing.T) {
		tmpFile, err := os.CreateTemp("", "slides-grid4-*.md")
		if err != nil {
			t.Fatal(err)
		}
		defer os.Remove(tmpFile.Name())

		content := `---
layout: "grid-4"
cols: "60/40"
rows: "70/30"
---
::tl::
A

::tr::
B

::bl::
C

::br::
D
`
		if _, err := tmpFile.WriteString(content); err != nil {
			t.Fatal(err)
		}
		tmpFile.Close()

		pres, err := ParseMarkdownFile(tmpFile.Name())
		if err != nil {
			t.Fatalf("ParseMarkdownFile failed: %v", err)
		}
		s := pres.Slides[0]
		if len(s.Regions) != 4 {
			t.Fatalf("Expected 4 regions, got %d: %v", len(s.Regions), s.Regions)
		}
		if s.ColsCSS != "60fr 40fr" {
			t.Errorf("Expected ColsCSS '60fr 40fr', got %q", s.ColsCSS)
		}
		if s.RowsCSS != "70fr 30fr" {
			t.Errorf("Expected RowsCSS '70fr 30fr', got %q", s.RowsCSS)
		}
	})

	t.Run("non-numeric ratio falls back to equal split", func(t *testing.T) {
		tmpFile, err := os.CreateTemp("", "slides-nonnumeric-*.md")
		if err != nil {
			t.Fatal(err)
		}
		defer os.Remove(tmpFile.Name())

		content := `---
layout: "split-h"
ratio: "70/abc"
---
::left::
L

::right::
R
`
		if _, err := tmpFile.WriteString(content); err != nil {
			t.Fatal(err)
		}
		tmpFile.Close()

		pres, err := ParseMarkdownFile(tmpFile.Name())
		if err != nil {
			t.Fatalf("ParseMarkdownFile failed: %v", err)
		}
		s := pres.Slides[0]
		if s.ColsCSS != "" {
			t.Errorf("Expected empty ColsCSS for non-numeric ratio part, got %q", s.ColsCSS)
		}
	})

	t.Run("grid-4 with only cols set leaves rows empty", func(t *testing.T) {
		tmpFile, err := os.CreateTemp("", "slides-grid4-colsonly-*.md")
		if err != nil {
			t.Fatal(err)
		}
		defer os.Remove(tmpFile.Name())

		content := `---
layout: "grid-4"
cols: "60/40"
---
::tl::
A

::tr::
B

::bl::
C

::br::
D
`
		if _, err := tmpFile.WriteString(content); err != nil {
			t.Fatal(err)
		}
		tmpFile.Close()

		pres, err := ParseMarkdownFile(tmpFile.Name())
		if err != nil {
			t.Fatalf("ParseMarkdownFile failed: %v", err)
		}
		s := pres.Slides[0]
		if s.ColsCSS != "60fr 40fr" {
			t.Errorf("Expected ColsCSS '60fr 40fr', got %q", s.ColsCSS)
		}
		if s.RowsCSS != "" {
			t.Errorf("Expected empty RowsCSS when rows is unset, got %q", s.RowsCSS)
		}
	})
}

func TestFontFields(t *testing.T) {
	t.Run("global fonts.sans and fonts.mono from top-level frontmatter", func(t *testing.T) {
		tmpFile, err := os.CreateTemp("", "slides-font-global-*.md")
		if err != nil {
			t.Fatal(err)
		}
		defer os.Remove(tmpFile.Name())

		content := `---
title: Font Test
fonts:
  sans: 'Space Grotesk'
  mono: 'JetBrains Mono'
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
		if pres.Fonts.Sans != "Space Grotesk" {
			t.Errorf("Expected Fonts.Sans 'Space Grotesk', got %q", pres.Fonts.Sans)
		}
		if pres.Fonts.Mono != "JetBrains Mono" {
			t.Errorf("Expected Fonts.Mono 'JetBrains Mono', got %q", pres.Fonts.Mono)
		}
	})

	t.Run("no global fonts leaves Fonts empty", func(t *testing.T) {
		tmpFile, err := os.CreateTemp("", "slides-font-empty-*.md")
		if err != nil {
			t.Fatal(err)
		}
		defer os.Remove(tmpFile.Name())

		content := `# Slide 1`
		if _, err := tmpFile.WriteString(content); err != nil {
			t.Fatal(err)
		}
		tmpFile.Close()

		pres, err := ParseMarkdownFile(tmpFile.Name())
		if err != nil {
			t.Fatalf("ParseMarkdownFile failed: %v", err)
		}
		if pres.Fonts.Sans != "" || pres.Fonts.Mono != "" {
			t.Errorf("Expected empty Fonts, got %+v", pres.Fonts)
		}
	})

	t.Run("per-slide headerFont overrides only that slide", func(t *testing.T) {
		tmpFile, err := os.CreateTemp("", "slides-font-header-*.md")
		if err != nil {
			t.Fatal(err)
		}
		defer os.Remove(tmpFile.Name())

		content := `---
title: Font Test
fonts:
  sans: 'Space Grotesk'
---
# Slide 1 (uses global font only)

---
headerFont: "Poppins, sans-serif"
---
# Slide 2 (custom header font)
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
		if pres.Slides[0].HeaderFont != "" {
			t.Errorf("Expected slide 0 HeaderFont empty, got %q", pres.Slides[0].HeaderFont)
		}
		if pres.Slides[1].HeaderFont != "Poppins, sans-serif" {
			t.Errorf("Expected slide 1 HeaderFont 'Poppins, sans-serif', got %q", pres.Slides[1].HeaderFont)
		}
	})
}

func TestFragments(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "slides-fragments-*.md")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())

	content := `# Slide 1

- Alpha
- Beta

---
fragments: true
---

# Slide 2

- Alpha
- Beta
- Gamma
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

	if pres.Slides[0].Fragments {
		t.Errorf("Expected slide 0 Fragments false, got true")
	}
	if strings.Contains(pres.Slides[0].HTMLContent, `class="fragment"`) {
		t.Errorf("Expected slide 0 HTML to have no fragment classing, got %q", pres.Slides[0].HTMLContent)
	}

	if !pres.Slides[1].Fragments {
		t.Errorf("Expected slide 1 Fragments true, got false")
	}
	want := `<li class="fragment" data-fragment-index="0">Alpha</li>` + "\n" +
		`<li class="fragment" data-fragment-index="1">Beta</li>` + "\n" +
		`<li class="fragment" data-fragment-index="2">Gamma</li>`
	if !strings.Contains(pres.Slides[1].HTMLContent, want) {
		t.Errorf("Expected slide 1 HTML to contain numbered fragment <li> tags, got %q", pres.Slides[1].HTMLContent)
	}
}

func TestFragmentsWithoutList(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "slides-fragments-noli-*.md")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())

	content := `---
fragments: true
---

# Log lines

` + "```" + `sh
one
` + "```" + `

or

` + "```" + `sh
two
` + "```"
	if _, err := tmpFile.WriteString(content); err != nil {
		t.Fatal(err)
	}
	tmpFile.Close()

	pres, err := ParseMarkdownFile(tmpFile.Name())
	if err != nil {
		t.Fatalf("ParseMarkdownFile failed: %v", err)
	}
	if len(pres.Slides) != 1 {
		t.Fatalf("Expected 1 slide, got %d", len(pres.Slides))
	}

	html := pres.Slides[0].HTMLContent
	if !strings.Contains(html, `data-fragment-index="0"`) {
		t.Errorf("Expected first fragment (pre) at index 0, got %q", html)
	}
	if !strings.Contains(html, `<p class="fragment" data-fragment-index="1">or</p>`) {
		t.Errorf("Expected the 'or' paragraph fragmented at index 1, got %q", html)
	}
	if !strings.Contains(html, `data-fragment-index="2"`) {
		t.Errorf("Expected second pre fragment at index 2, got %q", html)
	}
	// The <pre> tags carry chroma's inline style attribute; the class must
	// be appended to it, not replace it.
	if !strings.Contains(html, `style="background-color:#fff;" class="fragment" data-fragment-index="0">`) {
		t.Errorf("Expected fragment class appended after chroma's style attribute, got %q", html)
	}
}

func TestControlsVisibilityFields(t *testing.T) {
	t.Run("default false when unset", func(t *testing.T) {
		tmpFile, err := os.CreateTemp("", "slides-controls-default-*.md")
		if err != nil {
			t.Fatal(err)
		}
		defer os.Remove(tmpFile.Name())

		content := `# Slide 1`
		if _, err := tmpFile.WriteString(content); err != nil {
			t.Fatal(err)
		}
		tmpFile.Close()

		pres, err := ParseMarkdownFile(tmpFile.Name())
		if err != nil {
			t.Fatalf("ParseMarkdownFile failed: %v", err)
		}
		if pres.ShowControls {
			t.Errorf("Expected ShowControls false by default, got true")
		}
		if pres.ShowSlideNumber {
			t.Errorf("Expected ShowSlideNumber false by default, got true")
		}
	})

	t.Run("enabled via global frontmatter", func(t *testing.T) {
		tmpFile, err := os.CreateTemp("", "slides-controls-enabled-*.md")
		if err != nil {
			t.Fatal(err)
		}
		defer os.Remove(tmpFile.Name())

		content := `---
showControls: true
showSlideNumber: true
---
# Slide 1
`
		if _, err := tmpFile.WriteString(content); err != nil {
			t.Fatal(err)
		}
		tmpFile.Close()

		pres, err := ParseMarkdownFile(tmpFile.Name())
		if err != nil {
			t.Fatalf("ParseMarkdownFile failed: %v", err)
		}
		if !pres.ShowControls {
			t.Errorf("Expected ShowControls true, got false")
		}
		if !pres.ShowSlideNumber {
			t.Errorf("Expected ShowSlideNumber true, got false")
		}
	})
}

// parseFromString writes content to a temp .md file and parses it.
func parseFromString(t *testing.T, content string) (*Presentation, error) {
	t.Helper()
	f, err := os.CreateTemp(t.TempDir(), "deck_*.md")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	if _, err := f.WriteString(content); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}
	f.Close()
	return ParseMarkdownFile(f.Name())
}

func TestSlideDimensions_Default16x9(t *testing.T) {
	pres, err := parseFromString(t, "# Slide 1")
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if pres.SlideWidthPx != 960 || pres.SlideHeightPx != 540 {
		t.Errorf("expected 960x540, got %dx%d", pres.SlideWidthPx, pres.SlideHeightPx)
	}
}

func TestSlideDimensions_4x3(t *testing.T) {
	pres, err := parseFromString(t, "---\naspectRatio: \"4:3\"\n---\n# Slide 1")
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if pres.SlideWidthPx != 960 || pres.SlideHeightPx != 720 {
		t.Errorf("expected 960x720, got %dx%d", pres.SlideWidthPx, pres.SlideHeightPx)
	}
}

func TestSlideDimensions_MalformedFallsBack(t *testing.T) {
	pres, err := parseFromString(t, "---\naspectRatio: \"garbage\"\n---\n# Slide 1")
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if pres.SlideWidthPx != 960 || pres.SlideHeightPx != 540 {
		t.Errorf("expected fallback 960x540, got %dx%d", pres.SlideWidthPx, pres.SlideHeightPx)
	}
}

func TestSlideDimensions_InfinityFallsBack(t *testing.T) {
	testCases := []struct {
		name        string
		aspectRatio string
	}{
		{"Inf in numerator", "Inf:9"},
		{"+Inf in numerator", "+Inf:9"},
		{"Infinity in numerator", "Infinity:9"},
		{"infinity lowercase in numerator", "infinity:9"},
		{"Inf in denominator", "16:Inf"},
		{"+Inf in denominator", "16:+Inf"},
		{"Infinity in denominator", "16:Infinity"},
		{"infinity lowercase in denominator", "16:infinity"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			pres, err := parseFromString(t, "---\naspectRatio: \""+tc.aspectRatio+"\"\n---\n# Slide 1")
			if err != nil {
				t.Fatalf("parse failed: %v", err)
			}
			if pres.SlideWidthPx != 960 || pres.SlideHeightPx != 540 {
				t.Errorf("expected fallback 960x540 for %q, got %dx%d", tc.aspectRatio, pres.SlideWidthPx, pres.SlideHeightPx)
			}
		})
	}
}

func TestGoogleFontsURL(t *testing.T) {
	t.Run("builds URL from global sans and mono fonts", func(t *testing.T) {
		tmpFile, err := os.CreateTemp("", "slides-gfonts-*.md")
		if err != nil {
			t.Fatal(err)
		}
		defer os.Remove(tmpFile.Name())

		content := `---
title: Font Test
fonts:
  sans: 'Space Grotesk'
  mono: 'JetBrains Mono'
---
# Slide 1
`
		if _, err := tmpFile.WriteString(content); err != nil {
			t.Fatal(err)
		}
		tmpFile.Close()

		pres, err := ParseMarkdownFile(tmpFile.Name())
		if err != nil {
			t.Fatalf("ParseMarkdownFile failed: %v", err)
		}
		if !strings.Contains(pres.GoogleFontsURL, "family=Space+Grotesk") {
			t.Errorf("expected GoogleFontsURL to contain 'family=Space+Grotesk', got %q", pres.GoogleFontsURL)
		}
		if !strings.Contains(pres.GoogleFontsURL, "family=JetBrains+Mono") {
			t.Errorf("expected GoogleFontsURL to contain 'family=JetBrains+Mono', got %q", pres.GoogleFontsURL)
		}
		if !strings.HasPrefix(pres.GoogleFontsURL, "https://fonts.googleapis.com/css2?") {
			t.Errorf("expected GoogleFontsURL to be a Google Fonts CSS2 URL, got %q", pres.GoogleFontsURL)
		}
	})

	t.Run("headerFont with fallback keyword only pulls the primary family name", func(t *testing.T) {
		tmpFile, err := os.CreateTemp("", "slides-gfonts-header-*.md")
		if err != nil {
			t.Fatal(err)
		}
		defer os.Remove(tmpFile.Name())

		content := `# Slide 1

---
headerFont: "Poppins, sans-serif"
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
		if !strings.Contains(pres.GoogleFontsURL, "family=Poppins") {
			t.Errorf("expected GoogleFontsURL to contain 'family=Poppins', got %q", pres.GoogleFontsURL)
		}
		if strings.Contains(pres.GoogleFontsURL, "sans-serif") {
			t.Errorf("expected 'sans-serif' fallback keyword to be excluded, got %q", pres.GoogleFontsURL)
		}
	})

	t.Run("multiple real fonts in one field are all fetched (e.g. latin font + Thai fallback)", func(t *testing.T) {
		tmpFile, err := os.CreateTemp("", "slides-gfonts-multi-*.md")
		if err != nil {
			t.Fatal(err)
		}
		defer os.Remove(tmpFile.Name())

		content := `---
fonts:
  sans: "Poppins, 'Noto Sans Thai'"
---
# Slide 1
`
		if _, err := tmpFile.WriteString(content); err != nil {
			t.Fatal(err)
		}
		tmpFile.Close()

		pres, err := ParseMarkdownFile(tmpFile.Name())
		if err != nil {
			t.Fatalf("ParseMarkdownFile failed: %v", err)
		}
		if !strings.Contains(pres.GoogleFontsURL, "family=Poppins") {
			t.Errorf("expected GoogleFontsURL to contain 'family=Poppins', got %q", pres.GoogleFontsURL)
		}
		if !strings.Contains(pres.GoogleFontsURL, "family=Noto+Sans+Thai") {
			t.Errorf("expected GoogleFontsURL to also contain 'family=Noto+Sans+Thai', got %q", pres.GoogleFontsURL)
		}
	})

	t.Run("generic CSS keyword after two real fonts is still excluded", func(t *testing.T) {
		tmpFile, err := os.CreateTemp("", "slides-gfonts-multi-generic-*.md")
		if err != nil {
			t.Fatal(err)
		}
		defer os.Remove(tmpFile.Name())

		content := `---
fonts:
  sans: "Poppins, 'Noto Sans Thai', sans-serif"
---
# Slide 1
`
		if _, err := tmpFile.WriteString(content); err != nil {
			t.Fatal(err)
		}
		tmpFile.Close()

		pres, err := ParseMarkdownFile(tmpFile.Name())
		if err != nil {
			t.Fatalf("ParseMarkdownFile failed: %v", err)
		}
		if !strings.Contains(pres.GoogleFontsURL, "family=Poppins") || !strings.Contains(pres.GoogleFontsURL, "family=Noto+Sans+Thai") {
			t.Errorf("expected both real fonts fetched, got %q", pres.GoogleFontsURL)
		}
		if strings.Contains(pres.GoogleFontsURL, "sans-serif") {
			t.Errorf("expected generic keyword 'sans-serif' excluded even after real fonts, got %q", pres.GoogleFontsURL)
		}
	})

	t.Run("duplicate font names across fields are deduplicated", func(t *testing.T) {
		tmpFile, err := os.CreateTemp("", "slides-gfonts-dup-*.md")
		if err != nil {
			t.Fatal(err)
		}
		defer os.Remove(tmpFile.Name())

		content := `---
fonts:
  sans: 'Poppins'
---
# Slide 1

---
headerFont: 'Poppins'
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
		count := strings.Count(pres.GoogleFontsURL, "family=Poppins")
		if count != 1 {
			t.Errorf("expected 'family=Poppins' to appear once (deduplicated), got %d times in %q", count, pres.GoogleFontsURL)
		}
	})

	t.Run("no custom fonts leaves GoogleFontsURL empty", func(t *testing.T) {
		tmpFile, err := os.CreateTemp("", "slides-gfonts-empty-*.md")
		if err != nil {
			t.Fatal(err)
		}
		defer os.Remove(tmpFile.Name())

		content := `# Slide 1`
		if _, err := tmpFile.WriteString(content); err != nil {
			t.Fatal(err)
		}
		tmpFile.Close()

		pres, err := ParseMarkdownFile(tmpFile.Name())
		if err != nil {
			t.Fatalf("ParseMarkdownFile failed: %v", err)
		}
		if pres.GoogleFontsURL != "" {
			t.Errorf("expected empty GoogleFontsURL, got %q", pres.GoogleFontsURL)
		}
	})
}

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
