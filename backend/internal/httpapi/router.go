// Package httpapi wires the REST API and the web app into one HTTP handler.
package httpapi

import (
	"context"
	"net/http"
	"strings"
)

// Pinger reports whether the database is reachable.
type Pinger interface {
	PingContext(ctx context.Context) error
}

// Deps are the dependencies of the HTTP handler.
type Deps struct {
	DB  Pinger
	Web http.Handler // serves the web app for every non-API path
}

// New returns the root handler: /api/* goes to the API, everything else to the web app.
func New(d Deps) http.Handler {
	api := http.NewServeMux()
	api.Handle("GET /api/health", health(d.DB))

	root := http.NewServeMux()
	root.Handle("/api/", jsonMuxErrors(api))
	root.Handle("/", d.Web)
	return logRequests(root)
}

func health(db Pinger) HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) error {
		if err := db.PingContext(r.Context()); err != nil {
			return NewError(http.StatusServiceUnavailable, "unavailable", "Database is not reachable")
		}
		WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
		return nil
	}
}

// jsonMuxErrors turns the plain-text 404 and 405 responses that http.ServeMux
// writes for unmatched requests into the API's JSON error format.
func jsonMuxErrors(mux *http.ServeMux) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, pattern := mux.Handler(r); pattern != "" {
			mux.ServeHTTP(w, r)
			return
		}
		mux.ServeHTTP(&muxErrorWriter{ResponseWriter: w}, r)
	})
}

type muxErrorWriter struct {
	http.ResponseWriter
	replaced bool
}

func (m *muxErrorWriter) WriteHeader(status int) {
	if !strings.HasPrefix(m.Header().Get("Content-Type"), "text/plain") {
		m.ResponseWriter.WriteHeader(status)
		return
	}
	m.replaced = true
	m.Header().Del("Content-Length")
	switch status {
	case http.StatusMethodNotAllowed:
		WriteError(m.ResponseWriter, NewError(status, "method_not_allowed", "Method not allowed"))
	case http.StatusNotFound:
		WriteError(m.ResponseWriter, NewError(status, "not_found", "Not found"))
	default:
		WriteError(m.ResponseWriter, NewError(status, "error", http.StatusText(status)))
	}
}

func (m *muxErrorWriter) Write(b []byte) (int, error) {
	if m.replaced {
		return len(b), nil
	}
	return m.ResponseWriter.Write(b)
}
