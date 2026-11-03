package httpapi

import (
	"embed"
	"net/http"
)

// openAPIDocs are the OpenAPI documents of Boardly's API and of the connected
// apps' APIs. A test checks that they list exactly the registered routes.
//
//go:embed openapi/*.yaml
var openAPIDocs embed.FS

// openAPIDoc serves the embedded OpenAPI document name, without
// authentication.
func openAPIDoc(name string) http.Handler {
	doc, err := openAPIDocs.ReadFile("openapi/" + name)
	if err != nil {
		panic(err)
	}
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		h := w.Header()
		h.Set("Content-Type", "application/yaml")
		h.Set("Cache-Control", "no-cache")
		_, _ = w.Write(doc)
	})
}
