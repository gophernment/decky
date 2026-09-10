package parser

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
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/renderer/html"
	"github.com/yuin/goldmark/util"
	"gopkg.in/yaml.v3"
)

// DefaultSansFallback and DefaultMonoFallback are appended after any
// user-supplied font (fonts.sans, fonts.mono, headerFont) so an unavailable
// web font degrades gracefully to the built-in stack instead of the
// browser's default font. Keep these in sync with the --font-sans/
// --font-mono defaults in web/static/css/styles.css's :root block.
const (
	DefaultSansFallback = `'Inter', -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, Helvetica, Arial, sans-serif`
	DefaultMonoFallback = `'Fira Code', Consolas, Monaco, 'Courier New', monospace`
)

// baseSlideWidthPx is the reference width (in px) that slide height is
// derived from for any configured aspect ratio. 960 matches the
// long-standing default 16:9 slide size (960x540).
const baseSlideWidthPx = 960

// computeSlideDimensions parses an "W:H" aspect ratio string (e.g. "16:9",
// "4:3") into pixel dimensions at a fixed base width. Malformed or empty
// input falls back to the 16:9 default (960x540).
func computeSlideDimensions(aspectRatio string) (width, height int) {
	parts := strings.SplitN(aspectRatio, ":", 2)
	if len(parts) == 2 {
		w, errW := strconv.ParseFloat(strings.TrimSpace(parts[0]), 64)
		h, errH := strconv.ParseFloat(strings.TrimSpace(parts[1]), 64)
		if errW == nil && errH == nil && w > 0 && h > 0 && !math.IsInf(w, 0) && !math.IsInf(h, 0) {
			return baseSlideWidthPx, int(math.Round(baseSlideWidthPx * h / w))
		}
	}
	return baseSlideWidthPx, 540
}

// Presentation represents the parsed presentation document.
type Presentation struct {
	Title       string      `yaml:"title"`
	Author      string      `yaml:"author"`
	Theme       string      `yaml:"theme"`
	AspectRatio string      `yaml:"aspectRatio"`
	Fonts       FontsConfig `yaml:"fonts"`

	// HeadingColors sets deck-wide default colors/gradients per heading
	// level (h1-h4). A slide's own headingColors overrides individual
	// levels; levels it doesn't set fall back to this deck-wide default.
	// See Slide.HeadingColors and mergeHeadingColors (Task 2).
	HeadingColors map[string]string `yaml:"headingColors"`

	// CustomCSS is a raw CSS string set deck-wide via the global front
	// matter's `css:` key (typically a YAML block scalar). Every render
	// path (serve, presenter, export, html) emits it in a <style> block
	// placed after the built-in stylesheet so it wins the cascade,
	// letting a deck author override any built-in rule without forking
	// the tool. It is trusted verbatim — it is the author's own file.
	CustomCSS string `yaml:"css"`

	Slides []Slide

	// ShowControls and ShowSlideNumber toggle the prev/next nav buttons and
	// the slide-number indicator ("1 / 66") in the live presentation view.
	// Both default to false (hidden) and must be explicitly enabled in the
	// deck's frontmatter.
	ShowControls    bool `yaml:"showControls"`
	ShowSlideNumber bool `yaml:"showSlideNumber"`

	// GoogleFontsURL is computed (not user-set) from Fonts.Sans, Fonts.Mono,
	// and every slide's HeaderFont. It is a Google Fonts CSS2 stylesheet URL
	// that live views (presentation/presenter) link to, so custom web fonts
	// actually load instead of silently falling back. Left empty when no
	// custom font is set. Never used by export (self-contained output has
	// no network dependency).
	GoogleFontsURL string

	// SlideWidthPx and SlideHeightPx are computed from AspectRatio at a
	// fixed base width of 960px. Used by PDF capture/assembly (Task 4) and
	// template rendering (Task 2) to size slides correctly.
	SlideWidthPx  int
	SlideHeightPx int
}

// FontsConfig holds the deck-wide font family overrides.
type FontsConfig struct {
	Sans string `yaml:"sans"`
	Mono string `yaml:"mono"`
}

// Slide represents a single slide in the presentation.
type Slide struct {
	Index       int
	RawMarkdown string
	HTMLContent string
	Layout      string `yaml:"layout"`
	Background  string `yaml:"background"`
	Color       string `yaml:"color"`

	// Align overrides the slide's content alignment ("left", "center", or
	// "right"), emitted as an `align-<value>` class by web/templates/_slide.html.
	// Any other value is dropped during parsing. Its main use is opting a
	// `cover` slide out of its default centered layout. Slide 0 inherits the
	// global block's value, like Layout/Background/Color.
	Align            string `yaml:"align"`
	SpeakerNotes     string
	SpeakerNotesHTML string

	Ratio string `yaml:"ratio"`
	Cols  string `yaml:"cols"`
	Rows  string `yaml:"rows"`

	Regions map[string]string
	ColsCSS string
	RowsCSS string

	HeaderFont string `yaml:"headerFont"`

	// HeadingColors overrides the deck-wide default (Presentation.HeadingColors)
	// per heading level, for this slide only. See mergeHeadingColors (Task 2).
	HeadingColors map[string]string `yaml:"headingColors"`

	// HeadingColorCSS is computed (not user-set) from the merge of
	// Presentation.HeadingColors and this slide's own HeadingColors. It is
	// a scoped CSS rule set (one rule per heading level with a color or
	// gradient set), emitted as an inline <style> block by
	// web/templates/_slide.html. Empty if no heading colors apply.
	HeadingColorCSS template.CSS

	// Fragments, when true, numbers every <li> rendered from this slide's
	// markdown (in document order) and marks it with class="fragment" so
	// the live view reveals list items one at a time on next/prev nav
	// instead of showing the whole list at once (see applyFragmentClasses
	// and web/static/js/app.js's fragment stepping logic).
	Fragments bool `yaml:"fragments"`
}

// ParseMarkdownFile parses a Markdown file into a Presentation struct.
func ParseMarkdownFile(path string) (*Presentation, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	content := string(data)
	blocks := splitBySeparator(content)

	pres := &Presentation{
		Title:       "Presentation",
		AspectRatio: "16:9",
		Theme:       "default",
	}

	var slide0Layout, slide0Background, slide0Color, slide0Align string
	var slide0Ratio, slide0Cols, slide0Rows string
	var slide0HeaderFont string
	var slide0Fragments bool
	var remainingBlocks []string
	if len(blocks) > 0 {
		// Check if first block is empty (meaning the file started with ---)
		if strings.TrimSpace(blocks[0]) == "" && len(blocks) > 1 {
			if _, ok := parseYAMLMap(blocks[1]); ok {
				// Parse blocks[1] as global frontmatter
				var globalConfig Presentation
				if err := yaml.Unmarshal([]byte(blocks[1]), &globalConfig); err == nil {
					if globalConfig.Title != "" {
						pres.Title = globalConfig.Title
					}
					if globalConfig.Author != "" {
						pres.Author = globalConfig.Author
					}
					if globalConfig.Theme != "" {
						pres.Theme = globalConfig.Theme
					}
					if globalConfig.AspectRatio != "" {
						pres.AspectRatio = globalConfig.AspectRatio
					}
					if globalConfig.Fonts.Sans != "" {
						pres.Fonts.Sans = globalConfig.Fonts.Sans
					}
					if globalConfig.Fonts.Mono != "" {
						pres.Fonts.Mono = globalConfig.Fonts.Mono
					}
					if len(globalConfig.HeadingColors) > 0 {
						pres.HeadingColors = globalConfig.HeadingColors
					}
					if globalConfig.CustomCSS != "" {
						pres.CustomCSS = globalConfig.CustomCSS
					}
					pres.ShowControls = globalConfig.ShowControls
					pres.ShowSlideNumber = globalConfig.ShowSlideNumber
				}

				var slide0Config Slide
				if err := yaml.Unmarshal([]byte(blocks[1]), &slide0Config); err == nil {
					slide0Layout = slide0Config.Layout
					slide0Background = slide0Config.Background
					slide0Color = slide0Config.Color
					slide0Align = slide0Config.Align
					slide0Ratio = slide0Config.Ratio
					slide0Cols = slide0Config.Cols
					slide0Rows = slide0Config.Rows
					slide0HeaderFont = slide0Config.HeaderFont
					slide0Fragments = slide0Config.Fragments
				}

				remainingBlocks = blocks[2:]
			} else {
				remainingBlocks = blocks[1:]
			}
		} else {
			remainingBlocks = blocks
		}
	}

	slideIdx := 0
	for i := 0; i < len(remainingBlocks); {
		block := remainingBlocks[i]

		var slide Slide
		slide.Index = slideIdx
		if slideIdx == 0 {
			slide.Layout = slide0Layout
			slide.Background = slide0Background
			slide.Color = slide0Color
			slide.Align = slide0Align
			slide.Ratio = slide0Ratio
			slide.Cols = slide0Cols
			slide.Rows = slide0Rows
			slide.HeaderFont = slide0HeaderFont
			slide.Fragments = slide0Fragments
		}

		// Check if the current block is a YAML frontmatter block
		if _, ok := parseYAMLMap(block); ok {
			// Parse block as frontmatter
			_ = yaml.Unmarshal([]byte(block), &slide)

			if i < len(remainingBlocks)-1 {
				contentBlock := remainingBlocks[i+1]
				notes, cleanContent := extractSpeakerNotes(contentBlock)
				slide.RawMarkdown = cleanContent
				slide.SpeakerNotes = notes
				i += 2
			} else {
				slide.RawMarkdown = ""
				slide.SpeakerNotes = ""
				i++
			}
		} else {
			notes, cleanContent := extractSpeakerNotes(block)
			slide.RawMarkdown = cleanContent
			slide.SpeakerNotes = notes
			i++
		}

		notesHTML, err := RenderMarkdownToHTML(slide.SpeakerNotes)
		if err != nil {
			return nil, err
		}
		slide.SpeakerNotesHTML = notesHTML

		header, regions := splitRegions(slide.RawMarkdown)

		htmlContent, err := RenderMarkdownToHTML(header)
		if err != nil {
			return nil, err
		}
		if slide.Fragments {
			htmlContent = applyFragmentClasses(htmlContent)
		}
		slide.HTMLContent = htmlContent

		if len(regions) > 0 {
			renderedRegions := make(map[string]string, len(regions))
			for name, regionMarkdown := range regions {
				regionHTML, err := RenderMarkdownToHTML(regionMarkdown)
				if err != nil {
					return nil, err
				}
				if slide.Fragments {
					regionHTML = applyFragmentClasses(regionHTML)
				}
				renderedRegions[name] = regionHTML
			}
			slide.Regions = renderedRegions
		}

		switch slide.Layout {
		case "split-h":
			slide.ColsCSS = ratioToGridTemplate(slide.Ratio, 2)
		case "split-v":
			slide.RowsCSS = ratioToGridTemplate(slide.Ratio, 2)
		case "split-3":
			slide.ColsCSS = ratioToGridTemplate(slide.Ratio, 3)
		case "grid-4":
			slide.ColsCSS = ratioToGridTemplate(slide.Cols, 2)
			slide.RowsCSS = ratioToGridTemplate(slide.Rows, 2)
		}

		slide.Align = normalizeAlign(slide.Align)

		merged := mergeHeadingColors(pres.HeadingColors, slide.HeadingColors)
		slide.HeadingColorCSS = buildHeadingColorCSS(slide.Index, merged)

		pres.Slides = append(pres.Slides, slide)
		slideIdx++
	}

	pres.GoogleFontsURL = buildGoogleFontsURL(pres)
	pres.SlideWidthPx, pres.SlideHeightPx = computeSlideDimensions(pres.AspectRatio)

	return pres, nil
}

// genericCSSFontFamilies are CSS generic font-family keywords, not real font
// names — never worth requesting from Google Fonts.
var genericCSSFontFamilies = map[string]bool{
	"serif": true, "sans-serif": true, "monospace": true, "cursive": true,
	"fantasy": true, "system-ui": true, "ui-serif": true, "ui-sans-serif": true,
	"ui-monospace": true, "ui-rounded": true, "math": true, "emoji": true, "fangsong": true,
}

// buildGoogleFontsURL constructs a Google Fonts CSS2 stylesheet URL from
// every distinct custom font family referenced in the presentation
// (Fonts.Sans, Fonts.Mono, and each slide's HeaderFont). Each field may hold
// a full CSS font stack (e.g. "Poppins, 'Noto Sans Thai', sans-serif") —
// every real font name in the list is fetched; generic CSS keywords like
// "sans-serif" are not real fonts and are dropped. Returns "" if no custom
// font is set anywhere in the presentation.
func buildGoogleFontsURL(pres *Presentation) string {
	seen := make(map[string]bool)
	var families []string

	add := func(value string) {
		for _, part := range strings.Split(value, ",") {
			name := strings.Trim(strings.TrimSpace(part), `'"`)
			if name == "" || genericCSSFontFamilies[strings.ToLower(name)] || seen[name] {
				continue
			}
			seen[name] = true
			families = append(families, name)
		}
	}

	add(pres.Fonts.Sans)
	add(pres.Fonts.Mono)
	for _, s := range pres.Slides {
		add(s.HeaderFont)
	}

	if len(families) == 0 {
		return ""
	}

	params := make([]string, len(families))
	for i, f := range families {
		params[i] = "family=" + url.QueryEscape(f)
	}
	return "https://fonts.googleapis.com/css2?" + strings.Join(params, "&") + "&display=swap"
}

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
// Solid colors set both `color` and `-webkit-text-fill-color`: layouts like
// `.slide.cover h1` apply their own gradient-text effect via
// `-webkit-text-fill-color: transparent`, which wins over a plain `color`
// declaration in WebKit/Blink browsers, so the fill color must be set
// explicitly to override it. Gradients use the background-clip:text
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
			fmt.Fprintf(&b, "color: %s; -webkit-text-fill-color: %s;", value, value)
		}
		b.WriteString(" }\n")
	}
	return template.CSS(b.String())
}

// normalizeAlign lower-cases and validates a slide's `align` value, returning
// "" (no alignment override) for anything that isn't "left", "center", or
// "right" so web/templates/_slide.html never emits a bogus align-* class.
func normalizeAlign(align string) string {
	switch strings.ToLower(strings.TrimSpace(align)) {
	case "left":
		return "left"
	case "center":
		return "center"
	case "right":
		return "right"
	default:
		return ""
	}
}

// splitBySeparator splits the markdown content by lines that are exactly "---".
func splitBySeparator(content string) []string {
	content = strings.ReplaceAll(content, "\r\n", "\n")
	lines := strings.Split(content, "\n")

	var blocks []string
	var currentBlock []string
	inCodeBlock := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "```") || strings.HasPrefix(trimmed, "~~~") {
			inCodeBlock = !inCodeBlock
		}

		if trimmed == "---" && !inCodeBlock {
			blocks = append(blocks, strings.Join(currentBlock, "\n"))
			currentBlock = []string{}
		} else {
			currentBlock = append(currentBlock, line)
		}
	}
	blocks = append(blocks, strings.Join(currentBlock, "\n"))
	return blocks
}

var regionMarkerRegex = regexp.MustCompile(`^::([a-zA-Z0-9_-]+)::$`)

// splitRegions splits a slide's markdown into a header portion (everything
// before the first ::name:: marker) and a map of named regions. Marker
// lines inside fenced code blocks (``` or ~~~) are treated as plain text,
// not markers, matching splitBySeparator's fence-tracking behavior.
// Duplicate markers for the same name have their content appended.
func splitRegions(markdown string) (header string, regions map[string]string) {
	lines := strings.Split(markdown, "\n")

	var headerLines []string
	regions = make(map[string]string)
	var currentName string
	var currentLines []string
	inCodeBlock := false
	inRegion := false

	flush := func() {
		if !inRegion {
			return
		}
		text := strings.TrimSpace(strings.Join(currentLines, "\n"))
		if existing, ok := regions[currentName]; ok && existing != "" {
			if text != "" {
				regions[currentName] = existing + "\n\n" + text
			}
		} else {
			regions[currentName] = text
		}
	}

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "```") || strings.HasPrefix(trimmed, "~~~") {
			inCodeBlock = !inCodeBlock
		}

		if !inCodeBlock {
			if m := regionMarkerRegex.FindStringSubmatch(trimmed); m != nil {
				flush()
				currentName = m[1]
				currentLines = nil
				inRegion = true
				continue
			}
		}

		if inRegion {
			currentLines = append(currentLines, line)
		} else {
			headerLines = append(headerLines, line)
		}
	}
	flush()

	header = strings.TrimSpace(strings.Join(headerLines, "\n"))
	return header, regions
}

// ratioToGridTemplate converts a "/"-separated ratio string like "70/30"
// into a CSS grid-template value like "70fr 30fr". It returns "" (signaling
// "use the CSS default equal split") if ratio is empty, the part count
// doesn't match wantParts, or any part isn't a positive number.
func ratioToGridTemplate(ratio string, wantParts int) string {
	if ratio == "" {
		return ""
	}
	parts := strings.Split(ratio, "/")
	if len(parts) != wantParts {
		return ""
	}

	values := make([]string, len(parts))
	for i, p := range parts {
		trimmed := strings.TrimSpace(p)
		n, err := strconv.ParseFloat(trimmed, 64)
		if err != nil || n <= 0 {
			return ""
		}
		values[i] = trimmed + "fr"
	}
	return strings.Join(values, " ")
}

var coreFrontmatterKeys = map[string]bool{
	"title":           true,
	"author":          true,
	"theme":           true,
	"aspectRatio":     true,
	"css":             true,
	"layout":          true,
	"background":      true,
	"color":           true,
	"align":           true,
	"class":           true,
	"transition":      true,
	"disabled":        true,
	"clicks":          true,
	"preload":         true,
	"src":             true,
	"name":            true,
	"route":           true,
	"drawings":        true,
	"ratio":           true,
	"cols":            true,
	"rows":            true,
	"fonts":           true,
	"headerFont":      true,
	"headingColors":   true,
	"fragments":       true,
	"showControls":    true,
	"showSlideNumber": true,
}

// parseYAMLMap checks if a block is a valid non-empty YAML map.
func parseYAMLMap(block string) (map[string]interface{}, bool) {
	block = strings.TrimSpace(block)
	if block == "" {
		return nil, false
	}

	var m map[string]interface{}
	err := yaml.Unmarshal([]byte(block), &m)
	if err != nil {
		return nil, false
	}
	if len(m) == 0 {
		return nil, false
	}

	// Validate map keys consist of standard YAML identifiers
	var validate func(interface{}) bool
	validate = func(val interface{}) bool {
		switch v := val.(type) {
		case map[string]interface{}:
			for k, child := range v {
				if k == "" {
					return false
				}
				for _, r := range k {
					if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_') {
						return false
					}
				}
				if !validate(child) {
					return false
				}
			}
		case []interface{}:
			for _, item := range v {
				if !validate(item) {
					return false
				}
			}
		}
		return true
	}

	if !validate(m) {
		return nil, false
	}

	// Ensure the YAML map contains at least one core slide or presentation configuration key.
	// This prevents markdown paragraphs/lists containing colons from being misclassified as frontmatter
	// while allowing frontmatter blocks to contain YAML comments on any line (including the first).
	hasCoreKey := false
	for k := range m {
		if coreFrontmatterKeys[k] {
			hasCoreKey = true
			break
		}
	}
	if !hasCoreKey {
		return nil, false
	}

	// Verify every line of the block is YAML structure to avoid false positives with markdown
	lines := strings.Split(block, "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		// Must be a comment, or start with spaces/tabs (indented YAML), or be a key-value line
		if strings.HasPrefix(trimmed, "#") {
			continue
		}
		if strings.HasPrefix(line, " ") || strings.HasPrefix(line, "\t") {
			continue
		}
		// Otherwise, it must be a top-level key-value line
		colonIdx := strings.Index(trimmed, ":")
		if colonIdx == -1 || colonIdx == 0 {
			return nil, false
		}
		key := strings.TrimSpace(trimmed[:colonIdx])
		// Key must be valid YAML key (alphanumeric/hyphen/underscore)
		for _, r := range key {
			if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_') {
				return nil, false
			}
		}
	}

	return m, true
}

// extractSpeakerNotes searches for HTML comment blocks (<!-- ... -->) at the bottom of the slide markdown
// and returns the extracted speaker notes and the clean markdown.
func extractSpeakerNotes(markdown string) (string, string) {
	startIdx := strings.LastIndex(markdown, "<!--")
	if startIdx == -1 {
		return "", strings.TrimSpace(markdown)
	}

	// Scan candidate "-->" occurrences left to right and accept the first
	// one after which only whitespace remains. A single Index lookup isn't
	// enough: speaker notes commonly contain ASCII-art arrows like "--->",
	// which itself contains "-->" as a substring, so the real closing tag
	// can be preceded by false matches inside the notes body.
	searchFrom := startIdx
	for {
		relIdx := strings.Index(markdown[searchFrom:], "-->")
		if relIdx == -1 {
			return "", strings.TrimSpace(markdown)
		}
		endIdx := searchFrom + relIdx

		afterComment := markdown[endIdx+3:]
		if strings.TrimSpace(afterComment) == "" {
			notes := markdown[startIdx+4 : endIdx]
			cleanMarkdown := markdown[:startIdx] + markdown[endIdx+3:]
			return strings.TrimSpace(notes), strings.TrimSpace(cleanMarkdown)
		}
		searchFrom = endIdx + 3
	}
}

var listItemOpenTagRegex = regexp.MustCompile(`<li>`)
var fragmentBlockOpenTagRegex = regexp.MustCompile(`<pre[^>]*>|<p>`)

// applyFragmentClasses numbers, in document order starting at 0, the units a
// slide reveals one at a time when it opts into fragments: true, and marks
// each with class="fragment" plus a data-fragment-index that
// web/static/js/app.js uses for next/prev stepping.
//
// A slide with any <li> fragments only its list items (goldmark never emits
// attributes on a bare <li>, tight or loose list, nesting included, so
// matching the literal opening tag is safe without a full HTML parse) —
// this is the common case (a bullet list revealed line by line) and leaves
// any plain paragraph/code block on the same slide, such as a static
// reference link, untouched.
//
// A slide with no <li> at all instead fragments its top-level <pre> and <p>
// blocks (e.g. a sequence of log-line examples), matching how the original
// reveal.js deck this tool replaced used fragments on non-list content.
// Fenced code blocks go through ChromaRenderer, which emits an inline
// style on <pre> (`<pre style="...">`) instead of a bare tag — the class is
// appended inside that existing tag rather than assuming no attributes.
func applyFragmentClasses(htmlContent string) string {
	if strings.Contains(htmlContent, "<li>") {
		idx := 0
		return listItemOpenTagRegex.ReplaceAllStringFunc(htmlContent, func(string) string {
			tag := `<li class="fragment" data-fragment-index="` + strconv.Itoa(idx) + `">`
			idx++
			return tag
		})
	}

	idx := 0
	return fragmentBlockOpenTagRegex.ReplaceAllStringFunc(htmlContent, func(match string) string {
		attrs := ` class="fragment" data-fragment-index="` + strconv.Itoa(idx) + `">`
		idx++
		return match[:len(match)-1] + attrs
	})
}

var markdownRenderer = goldmark.New(
	goldmark.WithExtensions(
		extension.Table,
	),
	goldmark.WithRendererOptions(
		html.WithUnsafe(),
		renderer.WithNodeRenderers(
			util.Prioritized(&ChromaRenderer{}, 100),
		),
	),
)

// RenderMarkdownToHTML renders markdown content into HTML with code syntax highlighting.
func RenderMarkdownToHTML(markdownInput string) (string, error) {
	var buf bytes.Buffer
	if err := markdownRenderer.Convert([]byte(markdownInput), &buf); err != nil {
		return "", err
	}
	return buf.String(), nil
}

var chromaFormatter = chromahtml.New(
	chromahtml.WithClasses(false),
)

type ChromaRenderer struct{}

func (r *ChromaRenderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	reg.Register(ast.KindFencedCodeBlock, r.renderFencedCodeBlock)
	reg.Register(ast.KindCodeBlock, r.renderCodeBlock)
}

func (r *ChromaRenderer) renderFencedCodeBlock(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if !entering {
		return ast.WalkContinue, nil
	}
	n := node.(*ast.FencedCodeBlock)
	lang := string(n.Language(source))

	var codeBuf bytes.Buffer
	l := n.Lines().Len()
	for i := 0; i < l; i++ {
		line := n.Lines().At(i)
		codeBuf.Write(line.Value(source))
	}

	return r.highlight(w, codeBuf.String(), lang)
}

func (r *ChromaRenderer) renderCodeBlock(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if !entering {
		return ast.WalkContinue, nil
	}
	n := node.(*ast.CodeBlock)

	var codeBuf bytes.Buffer
	l := n.Lines().Len()
	for i := 0; i < l; i++ {
		line := n.Lines().At(i)
		codeBuf.Write(line.Value(source))
	}

	return r.highlight(w, codeBuf.String(), "")
}

func (r *ChromaRenderer) highlight(w util.BufWriter, code string, lang string) (ast.WalkStatus, error) {
	lexer := lexers.Get(lang)
	if lexer == nil {
		lexer = lexers.Fallback
	}

	style := styles.Get("github")
	if style == nil {
		style = styles.Fallback
	}

	iterator, err := lexer.Tokenise(nil, code)
	if err != nil {
		return ast.WalkStop, err
	}

	err = chromaFormatter.Format(w, style, iterator)
	if err != nil {
		return ast.WalkStop, err
	}

	return ast.WalkSkipChildren, nil
}
