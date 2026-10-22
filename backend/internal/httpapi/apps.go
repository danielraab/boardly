package httpapi

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"famodash/internal/apptokens"
)

type appTokenKey struct{}

// appToken returns the app token that appAPI put into the context.
func appToken(ctx context.Context) (apptokens.Token, bool) {
	t, ok := ctx.Value(appTokenKey{}).(apptokens.Token)
	return t, ok
}

// appAPI serves the endpoints of a connected app. Requests are authenticated
// only by an "Authorization: Bearer" app token for that app; the session
// cookie is ignored and the web app's cross-site check doesn't apply, so any
// origin may call them (CORS without credentials). A valid token is put into
// the context; requests without one are passed on too, because some of the
// app's endpoints are public, and the others reject them.
func (a *api) appAPI(app apptokens.App, mux *http.ServeMux) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("Access-Control-Allow-Origin", "*")
		if r.Method == http.MethodOptions {
			h.Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE")
			// Clients let users add their own headers (e.g. for a reverse
			// proxy), so allow whatever the preflight asks for.
			h.Set("Access-Control-Allow-Headers", r.Header.Get("Access-Control-Request-Headers"))
			h.Set("Access-Control-Max-Age", "86400")
			w.WriteHeader(http.StatusNoContent)
			return
		}
		HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
			scheme, secret, _ := strings.Cut(r.Header.Get("Authorization"), " ")
			secret = strings.TrimSpace(secret)
			if strings.EqualFold(scheme, "Bearer") && secret != "" {
				token, err := a.appTokens.Authenticate(r.Context(), secret, app)
				switch {
				case err == nil:
					r = r.WithContext(context.WithValue(r.Context(), appTokenKey{}, token))
				case !errors.Is(err, apptokens.ErrInvalidToken):
					return err
				}
			}
			mux.ServeHTTP(&muxErrorWriter{ResponseWriter: w}, r)
			return nil
		}).ServeHTTP(w, r)
	})
}
