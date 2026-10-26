// Package httpapi wires the REST API and the web app into one HTTP handler.
package httpapi

import (
	"context"
	"net/http"
	"strings"

	"boardly/internal/apptokens"
	"boardly/internal/auth"
	"boardly/internal/dashboards"
	"boardly/internal/favourites"
	"boardly/internal/images"
	"boardly/internal/ratelimit"
	"boardly/internal/users"
)

// Pinger reports whether the database is reachable.
type Pinger interface {
	PingContext(ctx context.Context) error
}

// Deps are the dependencies of the HTTP handler.
type Deps struct {
	DB  Pinger
	Web http.Handler // serves the web app for every non-API path

	Auth       *auth.Service
	Users      *users.Store
	Favourites *favourites.Store
	Dashboards *dashboards.Store
	Images     *images.Store
	Favicons   *images.Favicons
	// Previews takes preview images that connected apps send.
	Previews  *images.Previews
	AppTokens *apptokens.Store
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
	a := &api{auth: d.Auth, users: d.Users, favourites: d.Favourites, dashboards: d.Dashboards, images: d.Images, favicons: d.Favicons, previews: d.Previews, refreshLimiter: ratelimit.New(refreshesPerHour, refreshBurst, 10_000), appTokens: d.AppTokens, origin: d.Origin, secureCookies: d.SecureCookies, trustProxy: d.TrustProxyHeaders}

	mux := http.NewServeMux()
	mux.Handle("GET /api/health", health(d.DB))
	mux.Handle("POST /api/auth/magic-link", HandlerFunc(a.requestLink))
	mux.Handle("POST /api/auth/magic-link/verify", HandlerFunc(a.verifyLink))
	mux.Handle("POST /api/auth/magic-link/code", HandlerFunc(a.redeemCode))
	mux.Handle("POST /api/auth/logout", HandlerFunc(a.logout))
	mux.Handle("GET /api/me", a.requireUser(a.getMe))
	mux.Handle("PATCH /api/me", a.requireUser(a.patchMe))
	mux.Handle("POST /api/auth/passkey/login/options", HandlerFunc(a.passkeyLoginOptions))
	mux.Handle("POST /api/auth/passkey/login/verify", HandlerFunc(a.passkeyLoginVerify))
	mux.Handle("GET /api/passkeys", a.requireUser(a.listPasskeys))
	mux.Handle("POST /api/passkeys/registration/options", a.requireUser(a.passkeyRegistrationOptions))
	mux.Handle("POST /api/passkeys/registration/verify", a.requireUser(a.passkeyRegistrationVerify))
	mux.Handle("DELETE /api/passkeys/{id}", a.requireUser(a.deletePasskey))
	mux.Handle("GET /api/favourites", a.requireUser(a.getFavourites))
	mux.Handle("POST /api/favourites", a.requireUser(a.createFavourite))
	mux.Handle("PATCH /api/favourites/{id}", a.requireUser(a.updateFavourite))
	mux.Handle("POST /api/favourites/bulk", a.requireUser(a.bulkFavourites))
	mux.Handle("POST /api/favourites/{id}/move", a.requireUser(a.moveFavourite))
	mux.Handle("DELETE /api/favourites/{id}", a.requireUser(a.deleteFavourite))
	mux.Handle("POST /api/favourites/import", a.requireUser(a.importFavourites))
	mux.Handle("GET /api/tags", a.requireUser(a.suggestTags))
	mux.Handle("GET /api/dashboards", a.requireUser(a.listDashboards))
	mux.Handle("POST /api/dashboards", a.requireUser(a.createDashboard))
	mux.Handle("PATCH /api/dashboards/{id}", a.requireUser(a.updateDashboard))
	mux.Handle("DELETE /api/dashboards/{id}", a.requireUser(a.deleteDashboard))
	mux.Handle("PUT /api/dashboards/{id}/layout", a.requireUser(a.saveLayout))
	mux.Handle("GET /api/u/{userSlug}/{dashboardSlug}", a.optionalUser(a.viewDashboard))
	mux.Handle("GET /api/images/{id}", HandlerFunc(a.getImage))
	mux.Handle("POST /api/images", a.requireUser(a.uploadImage))
	mux.Handle("POST /api/favicons", a.requireUser(a.getFavicon))
	mux.Handle("POST /api/favicons/refresh", a.requireUser(a.refreshFavicon))
	mux.Handle("GET /api/app-tokens", a.requireUser(a.listAppTokens))
	mux.Handle("POST /api/app-tokens", a.requireUser(a.createAppToken))
	mux.Handle("POST /api/app-tokens/{id}/revoke", a.requireUser(a.revokeAppToken))
	mux.Handle("DELETE /api/app-tokens/{id}", a.requireUser(a.deleteAppToken))

	root := http.NewServeMux()
	root.Handle("/api/", jsonMuxErrors(mux, d.Origin))
	root.Handle(karakeepPrefix+"/", a.appAPI(apptokens.AppKarakeep, a.karakeepAPI()))
	root.Handle(linkwardenPrefix+"/", a.appAPI(apptokens.AppLinkwarden, a.linkwardenAPI()))
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
