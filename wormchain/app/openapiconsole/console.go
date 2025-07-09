package openapiconsole

import (
	"embed"
	"html/template"
	"net/http"
)

// index.tpl is the template file for the OpenAPI console
//
//go:embed index.tpl
var index embed.FS

// Handler returns an http handler that servers OpenAPI console for an OpenAPI spec at specURL.
func Handler(title, specURL string) http.HandlerFunc {
	t, _ := template.ParseFS(index, "index.tpl")

	return func(w http.ResponseWriter, _ *http.Request) {
		t.Execute(w, struct { //nolint:errcheck
			Title string
			URL   string
		}{
			title,
			specURL,
		})
	}
}
