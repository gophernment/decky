package web_test

import (
	"html/template"
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
