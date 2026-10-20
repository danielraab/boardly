// Package httpapi wires the REST API and the web app into one HTTP handler.
package httpapi

import (
	"context"
	"net/http"
	"strings"

	"famodash/internal/auth"
	"famodash/internal/users"
)

// Pinger reports whether the database is reachable.
type Pinger interface {
	PingContext(ctx context.Context) error
}

// Deps are the dependencies of the HTTP handler.
type Deps struct {
	DB  Pinger
	Web http.Handler // serves the web app for every non-API path

	Auth  *auth.Service
	Users *users.Store
	// Origin is the web app's origin (scheme://host[:port]); state-changing
	// requests with another Origin header are rejected.
	Origin string
	// SecureCookies sets the Secure attribute on the session cookie.
	SecureCookies bool
	// TrustProxyHeaders takes the client IP from X-Forwarded-For.
	TrustProxyHeaders bool
}

// New returns the root handler: /api/* goes to the API, everything else to the web app.
func New(d Deps) http.Handler {
	a := &api{auth: d.Auth, users: d.Users, origin: d.Origin, secureCookies: d.SecureCookies, trustProxy: d.TrustProxyHeaders}

	mux := http.NewServeMux()
	mux.Handle("GET /api/health", health(d.DB))
	mux.Handle("POST /api/auth/magic-link", HandlerFunc(a.requestLink))
	mux.Handle("POST /api/auth/magic-link/verify", HandlerFunc(a.verifyLink))
	mux.Handle("POST /api/auth/logout", HandlerFunc(a.logout))
	mux.Handle("GET /api/me", a.requireUser(a.getMe))
	mux.Handle("PATCH /api/me", a.requireUser(a.patchMe))
	mux.Handle("POST /api/auth/passkey/login/options", HandlerFunc(a.passkeyLoginOptions))
	mux.Handle("POST /api/auth/passkey/login/verify", HandlerFunc(a.passkeyLoginVerify))
	mux.Handle("GET /api/passkeys", a.requireUser(a.listPasskeys))
	mux.Handle("POST /api/passkeys/registration/options", a.requireUser(a.passkeyRegistrationOptions))
	mux.Handle("POST /api/passkeys/registration/verify", a.requireUser(a.passkeyRegistrationVerify))
	mux.Handle("DELETE /api/passkeys/{id}", a.requireUser(a.deletePasskey))

	root := http.NewServeMux()
	root.Handle("/api/", jsonMuxErrors(mux, d.Origin))
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
// writes for unmatched requests into the API's JSON error format. Matched
// requests pass the cross-site request check first.
func jsonMuxErrors(mux *http.ServeMux, origin string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, pattern := mux.Handler(r); pattern != "" {
			if err := checkCSRF(r, origin); err != nil {
				WriteError(w, err)
				return
			}
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
