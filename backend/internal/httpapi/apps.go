package httpapi

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"famodash/internal/apptokens"
	"famodash/internal/users"
)

// errInvalidAppToken is the response to every rejected app request, whatever
// was wrong with the token.
var errInvalidAppToken = NewError(http.StatusUnauthorized, "unauthenticated", "Missing or invalid API key")

type appTokenKey struct{}

// appToken returns the app token that appAPI put into the context.
func appToken(ctx context.Context) (apptokens.Token, bool) {
	t, ok := ctx.Value(appTokenKey{}).(apptokens.Token)
	return t, ok
}

// appAPI serves the endpoints of a connected app. Requests are authenticated
// only by an "Authorization: Bearer" app token for that app; the session
// cookie is ignored and the web app's cross-site check doesn't apply, so any
// origin may call them (CORS without credentials).
func (a *api) appAPI(app apptokens.App, mux *http.ServeMux) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("Access-Control-Allow-Origin", "*")
		if r.Method == http.MethodOptions {
			h.Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE")
			h.Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
			h.Set("Access-Control-Max-Age", "86400")
			w.WriteHeader(http.StatusNoContent)
			return
		}
		HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
			scheme, secret, _ := strings.Cut(r.Header.Get("Authorization"), " ")
			if !strings.EqualFold(scheme, "Bearer") {
				secret = ""
			}
			token, err := a.appTokens.Authenticate(r.Context(), strings.TrimSpace(secret), app)
			if errors.Is(err, apptokens.ErrInvalidToken) {
				h.Set("WWW-Authenticate", "Bearer")
				return errInvalidAppToken
			}
			if err != nil {
				return err
			}
			mux.ServeHTTP(&muxErrorWriter{ResponseWriter: w}, r.WithContext(context.WithValue(r.Context(), appTokenKey{}, token)))
			return nil
		}).ServeHTTP(w, r)
	})
}

// karakeepMe is GET /api/karakeep/api/v1/users/me: the token owner, which
// clients use to check the connection.
func (a *api) karakeepMe(w http.ResponseWriter, r *http.Request) error {
	token, _ := appToken(r.Context())
	user, err := a.users.Get(r.Context(), token.UserID)
	if errors.Is(err, users.ErrNotFound) {
		return errInvalidAppToken
	}
	if err != nil {
		return err
	}
	WriteJSON(w, http.StatusOK, struct {
		ID    string  `json:"id"`
		Name  *string `json:"name"`
		Email string  `json:"email"`
	}{user.ID, user.Name, user.Email})
	return nil
}
