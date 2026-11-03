package httpapi

import "net/http"

// routeMux is an http.ServeMux that remembers the patterns registered on it,
// so the tests can check them against the OpenAPI documents.
type routeMux struct {
	*http.ServeMux
	patterns []string
}

func newRouteMux() *routeMux {
	return &routeMux{ServeMux: http.NewServeMux()}
}

func (m *routeMux) Handle(pattern string, h http.Handler) {
	m.patterns = append(m.patterns, pattern)
	m.ServeMux.Handle(pattern, h)
}

func (m *routeMux) HandleFunc(pattern string, h func(http.ResponseWriter, *http.Request)) {
	m.Handle(pattern, http.HandlerFunc(h))
}
