package web_test

import (
	"html/template"
	"strings"
	"testing"

	"github.com/gophernment/decky/web"
)

func TestAssetsExist(t *testing.T) {
	files := []string{
		"templates/presentation.html",
		"templates/presenter.html",
		"templates/export.html",
		"static/css/styles.css",
		"static/js/app.js",
	}

	for _, file := range files {
		t.Run(file, func(t *testing.T) {
			_, err := web.Assets.ReadFile(file)
			if err != nil {
				t.Fatalf("failed to read embedded file %s: %v", file, err)
			}
		})
	}
}

// A code block inside the flex-column .slide must never be shrunk below its
// natural height (it would show ~1 line and force the presenter to scroll).
func TestPreDoesNotShrinkInSlide(t *testing.T) {
	data, err := web.Assets.ReadFile("static/css/styles.css")
	if err != nil {
		t.Fatalf("failed to read styles.css: %v", err)
	}
	css := string(data)

	start := strings.Index(css, "\npre {")
	if start < 0 {
		t.Fatal("styles.css has no top-level `pre {` rule")
	}
	end := strings.Index(css[start:], "}")
	if end < 0 {
		t.Fatal("unterminated `pre {` rule")
	}
	if !strings.Contains(css[start:start+end], "flex-shrink: 0") {
		t.Error("`pre` rule must set flex-shrink: 0 so code blocks aren't squeezed inside the flex-column .slide")
	}
}

func TestTemplateCompiles(t *testing.T) {
	templates := []string{"templates/presentation.html", "templates/export.html"}
	for _, path := range templates {
		htmlData, err := web.Assets.ReadFile(path)
		if err != nil {
			t.Fatalf("failed to read template %s: %v", path, err)
		}

		tmpl, err := template.New(path).Funcs(template.FuncMap{
			"safe":           func(s string) template.HTML { return template.HTML(s) },
			"sansFontFamily": func(s string) template.CSS { return template.CSS(s) },
			"monoFontFamily": func(s string) template.CSS { return template.CSS(s) },
		}).Parse(string(htmlData))
		if err != nil {
			t.Fatalf("failed to parse template %s: %v", path, err)
		}

		if tmpl == nil {
			t.Fatal("expected non-nil template")
		}
	}
}
