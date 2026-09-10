package exporter

import (
	"html/template"
	"os"

	"github.com/gophernment/decky/internal/parser"
	"github.com/gophernment/decky/web"
)

// ExportHTML compiles the markdown presentation file into a single
// self-contained HTML file — CSS and JS are inlined, so the output has no
// external dependency and can be opened directly (file://) or hosted
// anywhere as a static file, unlike `decky serve`'s /asset/ convention
// which needs a running server. Any asset/ images the deck references keep
// their original (relative) paths, so they must ship alongside this file.
//
// This is deliberately a separate code path from the PDF exporter — export
// stays PDF-only per its own design decision, this is a different command.
func ExportHTML(markdownPath, outputPath string) error {
	pres, err := parser.ParseMarkdownFile(markdownPath)
	if err != nil {
		return err
	}

	cssBytes, err := web.Assets.ReadFile("static/css/styles.css")
	if err != nil {
		return err
	}

	jsBytes, err := web.Assets.ReadFile("static/js/app.js")
	if err != nil {
		return err
	}

	tmpl, err := template.New("export.html").Funcs(template.FuncMap{
		"safe":           func(content string) template.HTML { return template.HTML(content) },
		"sansFontFamily": func(custom string) template.CSS { return template.CSS(custom + ", " + parser.DefaultSansFallback) },
		"monoFontFamily": func(custom string) template.CSS { return template.CSS(custom + ", " + parser.DefaultMonoFallback) },
		"cssValue":       func(v string) template.CSS { return template.CSS(v) },
	}).ParseFS(web.Assets, "templates/export.html", "templates/_slide.html")
	if err != nil {
		return err
	}

	type htmlExportData struct {
		Title           string
		Fonts           parser.FontsConfig
		Slides          []parser.Slide
		CSS             template.CSS
		CustomCSS       template.CSS
		JS              template.JS
		SlideWidthPx    int
		SlideHeightPx   int
		ShowControls    bool
		ShowSlideNumber bool
	}

	data := htmlExportData{
		Title:           pres.Title,
		Fonts:           pres.Fonts,
		Slides:          pres.Slides,
		CSS:             template.CSS(cssBytes),
		CustomCSS:       template.CSS(pres.CustomCSS),
		JS:              template.JS(jsBytes),
		SlideWidthPx:    pres.SlideWidthPx,
		SlideHeightPx:   pres.SlideHeightPx,
		ShowControls:    pres.ShowControls,
		ShowSlideNumber: pres.ShowSlideNumber,
	}

	f, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer f.Close()

	return tmpl.Execute(f, data)
}
