package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"math"
	"mime"
	"net/http"
	"strconv"
	"time"

	"boardly/internal/apptokens"
	"boardly/internal/auth"
	"boardly/internal/dashboards"
	"boardly/internal/favourites"
	"boardly/internal/images"
	"boardly/internal/ratelimit"
	"boardly/internal/users"
)

// SessionCookie is the name of the session cookie.
const SessionCookie = "boardly_session"

// LoginRequestCookie binds sign-in links to the browser that requested them.
// It is only sent to the magic-link endpoints.
const (
	LoginRequestCookie = "boardly_login_request"
	loginRequestPath   = "/api/auth/magic-link"
)

const maxBodyBytes = 64 << 10

type api struct {
	auth       *auth.Service
	users      *users.Store
	favourites *favourites.Store
	dashboards *dashboards.Store
	images     *images.Store
	favicons   *images.Favicons
	previews   *images.Previews
	// refreshLimiter limits favicon refreshes per user.
	refreshLimiter *ratelimit.Limiter
	appTokens      *apptokens.Store
	origin         string
	secureCookies  bool
	trustProxy     bool
}

func (a *api) requestLink(w http.ResponseWriter, r *http.Request) error {
	var body struct {
		Email  string `json:"email"`
		Locale string `json:"locale"`
	}
	if err := decodeJSON(r, &body); err != nil {
		return err
	}
	email, err := auth.NormalizeEmail(body.Email)
	if err != nil {
		return NewError(http.StatusBadRequest, "invalid_email", "Enter a valid email address")
	}

	browserSecret, err := a.auth.RequestLink(r.Context(), email, body.Locale, ratelimit.ClientIP(r, a.trustProxy), cookieValue(r, LoginRequestCookie))
	var limited *auth.RateLimitedError
	if errors.As(err, &limited) {
		w.Header().Set("Retry-After", strconv.Itoa(retryAfterSeconds(limited.RetryAfter)))
		return NewError(http.StatusTooManyRequests, "rate_limited", "Too many sign-in requests, try again later")
	}
	if err != nil {
		return err
	}
	http.SetCookie(w, a.loginRequestCookie(browserSecret, int(auth.LinkTTL.Seconds())))
	WriteJSON(w, http.StatusAccepted, map[string]string{"status": "accepted"})
	return nil
}

func (a *api) verifyLink(w http.ResponseWriter, r *http.Request) error {
	var body struct {
		Token   string `json:"token"`
		Confirm bool   `json:"confirm"`
	}
	if err := decodeJSON(r, &body); err != nil {
		return err
	}
	user, sess, err := a.auth.Redeem(r.Context(), body.Token, r.UserAgent(), cookieValue(r, LoginRequestCookie), body.Confirm)
	if errors.Is(err, auth.ErrInvalidToken) {
		return NewError(http.StatusBadRequest, "invalid_token", "The sign-in link is invalid or has expired")
	}
	if errors.Is(err, auth.ErrConfirmationRequired) {
		return NewError(http.StatusForbidden, "confirmation_required", "Confirm the sign-in in this browser")
	}
	if err != nil {
		return err
	}
	a.setSessionCookie(w, sess)
	http.SetCookie(w, a.loginRequestCookie("", -1))
	WriteJSON(w, http.StatusOK, me{User: user, PasskeyEnrollmentUntil: sess.EnrollmentUntil})
	return nil
}

func (a *api) redeemCode(w http.ResponseWriter, r *http.Request) error {
	var body struct {
		Email string `json:"email"`
		Code  string `json:"code"`
	}
	if err := decodeJSON(r, &body); err != nil {
		return err
	}
	invalid := NewError(http.StatusBadRequest, "invalid_code", "The code is wrong or has expired")
	email, err := auth.NormalizeEmail(body.Email)
	if err != nil {
		return invalid
	}
	user, sess, err := a.auth.RedeemCode(r.Context(), email, body.Code, r.UserAgent(), cookieValue(r, LoginRequestCookie))
	if errors.Is(err, auth.ErrInvalidCode) {
		return invalid
	}
	if err != nil {
		return err
	}
	a.setSessionCookie(w, sess)
	http.SetCookie(w, a.loginRequestCookie("", -1))
	WriteJSON(w, http.StatusOK, me{User: user, PasskeyEnrollmentUntil: sess.EnrollmentUntil})
	return nil
}

func (a *api) logout(w http.ResponseWriter, r *http.Request) error {
	if c, err := r.Cookie(SessionCookie); err == nil {
		if err := a.auth.DeleteSession(r.Context(), c.Value); err != nil {
			return err
		}
	}
	http.SetCookie(w, a.cookie("", -1))
	w.WriteHeader(http.StatusNoContent)
	return nil
}

// me is the response of GET /api/me: the user plus the session's passkey
// enrollment window (nil when none is open).
type me struct {
	users.User
	PasskeyEnrollmentUntil *time.Time `json:"passkeyEnrollmentUntil"`
}

func (a *api) getMe(w http.ResponseWriter, r *http.Request) error {
	user, _ := CurrentUser(r.Context())
	a.writeMe(w, r, user)
	return nil
}

func (a *api) writeMe(w http.ResponseWriter, r *http.Request, user users.User) {
	sess, _ := CurrentSession(r.Context())
	body := me{User: user}
	if sess.EnrollmentOpen(a.auth.Now()) {
		body.PasskeyEnrollmentUntil = sess.EnrollmentUntil
	}
	WriteJSON(w, http.StatusOK, body)
}

func (a *api) patchMe(w http.ResponseWriter, r *http.Request) error {
	var body map[string]json.RawMessage
	if err := decodeJSON(r, &body); err != nil {
		return err
	}
	var patch users.Patch
	if raw, ok := body["name"]; ok {
		var name *string
		if err := json.Unmarshal(raw, &name); err != nil {
			return NewError(http.StatusBadRequest, "invalid_name", "The name must be a string or null")
		}
		patch.Name = &name
	}
	if raw, ok := body["slug"]; ok {
		var slug string
		if err := json.Unmarshal(raw, &slug); err != nil {
			return errInvalidSlug
		}
		patch.Slug = &slug
	}
	if patch.Name == nil && patch.Slug == nil {
		return NewError(http.StatusBadRequest, "invalid_request", "Nothing to update")
	}

	current, _ := CurrentUser(r.Context())
	user, err := a.users.Update(r.Context(), current.ID, patch)
	switch {
	case errors.Is(err, users.ErrInvalidName):
		return NewError(http.StatusBadRequest, "invalid_name", "The name can be at most 100 characters long")
	case errors.Is(err, users.ErrInvalidSlug):
		return errInvalidSlug
	case errors.Is(err, users.ErrSlugReserved):
		return NewError(http.StatusBadRequest, "slug_reserved", "This address is reserved")
	case errors.Is(err, users.ErrSlugTaken):
		return errSlugTaken
	case errors.Is(err, users.ErrNotFound):
		return errUnauthenticated
	case err != nil:
		return err
	}
	a.writeMe(w, r, user)
	return nil
}

var (
	errInvalidSlug = NewError(http.StatusBadRequest, "invalid_slug", "A slug has 1 to 32 characters from a-z, 0-9 and \"-\"")
	errSlugTaken   = NewError(http.StatusConflict, "slug_taken", "This slug is already in use")
)

var errUnauthenticated = NewError(http.StatusUnauthorized, "unauthenticated", "Sign in to continue")

type userKey struct{}

type sessionKey struct{}

// CurrentUser returns the signed-in user that requireUser put into the context.
func CurrentUser(ctx context.Context) (users.User, bool) {
	u, ok := ctx.Value(userKey{}).(users.User)
	return u, ok
}

// CurrentSession returns the session that requireUser put into the context.
func CurrentSession(ctx context.Context) (auth.Session, bool) {
	s, ok := ctx.Value(sessionKey{}).(auth.Session)
	return s, ok
}

// requireUser loads the session and its user, renews the cookie when the session
// was extended, and responds 401 when there is no valid session.
func (a *api) requireUser(next HandlerFunc) HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) error {
		ctx, err := a.withSession(w, r)
		if err != nil {
			return err
		}
		return next(w, r.WithContext(ctx))
	}
}

// optionalUser works like requireUser but continues without a user instead of
// responding 401, so CurrentUser reports false for signed-out requests.
func (a *api) optionalUser(next HandlerFunc) HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) error {
		ctx, err := a.withSession(w, r)
		if errors.Is(err, errUnauthenticated) {
			return next(w, r)
		}
		if err != nil {
			return err
		}
		return next(w, r.WithContext(ctx))
	}
}

// withSession returns the request context with the session and its user, and
// renews the cookie when the session was extended. Without a valid session it
// returns errUnauthenticated.
func (a *api) withSession(w http.ResponseWriter, r *http.Request) (context.Context, error) {
	c, err := r.Cookie(SessionCookie)
	if err != nil {
		return nil, errUnauthenticated
	}
	sess, err := a.auth.LookupSession(r.Context(), c.Value)
	if errors.Is(err, auth.ErrNoSession) {
		return nil, errUnauthenticated
	}
	if err != nil {
		return nil, err
	}
	user, err := a.users.Get(r.Context(), sess.UserID)
	if errors.Is(err, users.ErrNotFound) {
		return nil, errUnauthenticated
	}
	if err != nil {
		return nil, err
	}
	if sess.Renewed {
		a.setSessionCookie(w, sess)
	}
	ctx := context.WithValue(r.Context(), userKey{}, user)
	return context.WithValue(ctx, sessionKey{}, sess), nil
}

func (a *api) setSessionCookie(w http.ResponseWriter, sess auth.Session) {
	http.SetCookie(w, a.cookie(sess.Secret, int(sess.ExpiresAt.Sub(a.auth.Now()).Seconds())))
}

func (a *api) loginRequestCookie(value string, maxAge int) *http.Cookie {
	c := a.cookie(value, maxAge)
	c.Name = LoginRequestCookie
	c.Path = loginRequestPath
	return c
}

// cookieValue returns the named cookie's value, or "" if the request has none.
func cookieValue(r *http.Request, name string) string {
	if c, err := r.Cookie(name); err == nil {
		return c.Value
	}
	return ""
}

func (a *api) cookie(value string, maxAge int) *http.Cookie {
	return &http.Cookie{
		Name:     SessionCookie,
		Value:    value,
		Path:     "/",
		MaxAge:   maxAge,
		HttpOnly: true,
		Secure:   a.secureCookies,
		SameSite: http.SameSiteLaxMode,
	}
}

// checkCSRF rejects state-changing requests that don't send a JSON content
// type or come from another origin.
func checkCSRF(r *http.Request, origin string) *Error {
	switch r.Method {
	case http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
	default:
		return nil
	}
	forbidden := NewError(http.StatusForbidden, "forbidden_origin", "Cross-site request rejected")
	// Neither type can be sent cross-site without a CORS preflight, unlike
	// form types. Image uploads send the raw file as octet-stream.
	mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || (mediaType != "application/json" && mediaType != "application/octet-stream") {
		return forbidden
	}
	if o := r.Header.Get("Origin"); o != "" && o != origin {
		return forbidden
	}
	return nil
}

func decodeJSON(r *http.Request, v any) error {
	r.Body = http.MaxBytesReader(nil, r.Body, maxBodyBytes)
	if err := json.NewDecoder(r.Body).Decode(v); err != nil {
		return NewError(http.StatusBadRequest, "invalid_request", "The request body is not valid JSON")
	}
	return nil
}

func retryAfterSeconds(d time.Duration) int {
	return max(1, int(math.Ceil(d.Seconds())))
}
