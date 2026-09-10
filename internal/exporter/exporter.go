package exporter

import (
	"context"
	"fmt"
	"html/template"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"time"

	"github.com/gophernment/decky/internal/parser"
	"github.com/gophernment/decky/web"
)

// ChromeAvailableForTest exposes chromeAvailable to external test packages.
func ChromeAvailableForTest() bool { return chromeAvailable() }

// captureDeviceScale is the device pixel ratio each slide is rasterized at
// for a sharper PDF, without changing the CSS layout size (see capture.go's
// captureSlides doc comment for why this must be a rasterization scale, not
// a change to the slide's CSS box dimensions).
const captureDeviceScale = 2.0

// Export compiles the markdown presentation file into a single self-contained
// PDF, with every slide (including any asset/ images it references) captured
// as a full-resolution image via a locally installed headless Chrome.
func Export(markdownPath, outputPath string) error {
	pres, err := parser.ParseMarkdownFile(markdownPath)
	if err != nil {
		return err
	}
	if len(pres.Slides) == 0 {
		return fmt.Errorf("export: %s has no slides", markdownPath)
	}
	if !chromeAvailable() {
		return fmt.Errorf("export: no local Chrome/Chromium install found; PDF export requires one to be installed")
	}

	tmpHTMLPath, err := renderTempHTML(markdownPath, pres)
	if err != nil {
		return err
	}
	defer os.Remove(tmpHTMLPath)

	// Serve the deck's own directory (which now also contains tmpHTMLPath)
	// over HTTP instead of navigating to it via file://, so root-absolute
	// asset references (<img src="/asset/foo.png">, the same convention
	// `decky serve` uses) resolve to the deck's asset/ folder exactly as
	// they do under `decky serve`, rather than to the filesystem root.
	deckDir := filepath.Dir(markdownPath)
	assetServer := httptest.NewServer(http.FileServer(http.Dir(deckDir)))
	defer assetServer.Close()
	pageURL := assetServer.URL + "/" + filepath.Base(tmpHTMLPath)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	images, err := captureSlides(ctx, pageURL, len(pres.Slides), pres.SlideWidthPx, pres.SlideHeightPx, captureDeviceScale)
	if err != nil {
		return err
	}

	captureWidth := int(float64(pres.SlideWidthPx) * captureDeviceScale)
	captureHeight := int(float64(pres.SlideHeightPx) * captureDeviceScale)

	pdfBytes, err := buildPDF(images, captureWidth, captureHeight)
	if err != nil {
		return err
	}

	return os.WriteFile(outputPath, pdfBytes, 0o644)
}

// renderTempHTML renders the deck via the existing export.html template into
// a temp file placed next to markdownPath, so the file's relative and
// root-absolute asset/ references resolve once Export serves this directory
// over HTTP for the capture.
func renderTempHTML(markdownPath string, pres *parser.Presentation) (string, error) {
	cssBytes, err := web.Assets.ReadFile("static/css/styles.css")
	if err != nil {
		return "", err
	}
	jsBytes, err := web.Assets.ReadFile("static/js/app.js")
	if err != nil {
		return "", err
	}

	tmpl, err := template.New("export.html").Funcs(template.FuncMap{
		"safe":           func(content string) template.HTML { return template.HTML(content) },
		"sansFontFamily": func(custom string) template.CSS { return template.CSS(custom + ", " + parser.DefaultSansFallback) },
		"monoFontFamily": func(custom string) template.CSS { return template.CSS(custom + ", " + parser.DefaultMonoFallback) },
		"cssValue":       func(v string) template.CSS { return template.CSS(v) },
	}).ParseFS(web.Assets, "templates/export.html", "templates/_slide.html")
	if err != nil {
		return "", err
	}

	type ExportData struct {
		Title           string
		Fonts           parser.FontsConfig
		Slides          []parser.Slide
		SlideWidthPx    int
		SlideHeightPx   int
		CSS             template.CSS
		CustomCSS       template.CSS
		JS              template.JS
		ShowControls    bool
		ShowSlideNumber bool
	}

	data := ExportData{
		Title:           pres.Title,
		Fonts:           pres.Fonts,
		Slides:          pres.Slides,
		SlideWidthPx:    pres.SlideWidthPx,
		SlideHeightPx:   pres.SlideHeightPx,
		CSS:             template.CSS(cssBytes),
		CustomCSS:       template.CSS(pres.CustomCSS),
		JS:              template.JS(jsBytes),
		ShowControls:    pres.ShowControls,
		ShowSlideNumber: pres.ShowSlideNumber,
	}

	dir := filepath.Dir(markdownPath)
	tmpFile, err := os.CreateTemp(dir, ".decky-export-*.html")
	if err != nil {
		return "", err
	}
	defer tmpFile.Close()

	if err := tmpl.Execute(tmpFile, data); err != nil {
		os.Remove(tmpFile.Name())
		return "", err
	}

	absPath, err := filepath.Abs(tmpFile.Name())
	if err != nil {
		os.Remove(tmpFile.Name())
		return "", err
	}
	return absPath, nil
}
