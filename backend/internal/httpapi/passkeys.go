package httpapi

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"famodash/internal/auth"
)

// WebAuthnCookie carries the challenge of a running passkey ceremony.
const (
	WebAuthnCookie = "famodash_webauthn"
	webAuthnPath   = "/api"
)

func (a *api) webAuthnCookie(value string, maxAge int) *http.Cookie {
	c := a.cookie(value, maxAge)
	c.Name = WebAuthnCookie
	c.Path = webAuthnPath
	c.SameSite = http.SameSiteStrictMode
	return c
}

func (a *api) setChallengeCookie(w http.ResponseWriter, secret string) {
	http.SetCookie(w, a.webAuthnCookie(secret, int(auth.ChallengeTTL.Seconds())))
}

func (a *api) clearChallengeCookie(w http.ResponseWriter) {
	http.SetCookie(w, a.webAuthnCookie("", -1))
}

var (
	errPasskeyInvalid = NewError(http.StatusUnauthorized, "passkey_invalid", "The passkey could not be verified")
	errPasskeyUnknown = NewError(http.StatusUnauthorized, "passkey_unknown", "This passkey is no longer registered")
)

func readBody(r *http.Request) ([]byte, error) {
	body, err := io.ReadAll(http.MaxBytesReader(nil, r.Body, maxBodyBytes))
	if err != nil {
		return nil, NewError(http.StatusBadRequest, "invalid_request", "The request body is too large")
	}
	return body, nil
}

func (a *api) passkeyLoginOptions(w http.ResponseWriter, r *http.Request) error {
	opts, secret, err := a.auth.PasskeyLoginOptions(r.Context())
	if err != nil {
		return err
	}
	a.setChallengeCookie(w, secret)
	WriteJSON(w, http.StatusOK, opts)
	return nil
}

func (a *api) passkeyLoginVerify(w http.ResponseWriter, r *http.Request) error {
	body, err := readBody(r)
	if err != nil {
		return err
	}
	a.clearChallengeCookie(w)
	user, sess, err := a.auth.PasskeyLogin(r.Context(), cookieValue(r, WebAuthnCookie), body, r.UserAgent())
	if errors.Is(err, auth.ErrPasskeyUnknown) {
		return errPasskeyUnknown
	}
	if errors.Is(err, auth.ErrPasskeyInvalid) {
		return errPasskeyInvalid
	}
	if err != nil {
		return err
	}
	a.setSessionCookie(w, sess)
	WriteJSON(w, http.StatusOK, me{User: user})
	return nil
}

func (a *api) listPasskeys(w http.ResponseWriter, r *http.Request) error {
	user, _ := CurrentUser(r.Context())
	list, err := a.auth.ListPasskeys(r.Context(), user.ID)
	if err != nil {
		return err
	}
	WriteJSON(w, http.StatusOK, list)
	return nil
}

func (a *api) passkeyRegistrationOptions(w http.ResponseWriter, r *http.Request) error {
	user, _ := CurrentUser(r.Context())
	sess, _ := CurrentSession(r.Context())
	opts, secret, err := a.auth.PasskeyRegistrationOptions(r.Context(), user, sess)
	if errors.Is(err, auth.ErrEnrollmentClosed) {
		return NewError(http.StatusForbidden, "enrollment_window_closed", "Sign in with an email link to add a passkey")
	}
	if err != nil {
		return err
	}
	a.setChallengeCookie(w, secret)
	WriteJSON(w, http.StatusOK, opts)
	return nil
}

func (a *api) passkeyRegistrationVerify(w http.ResponseWriter, r *http.Request) error {
	var body struct {
		Credential json.RawMessage `json:"credential"`
		Label      *string         `json:"label"`
	}
	if err := decodeJSON(r, &body); err != nil {
		return err
	}
	user, _ := CurrentUser(r.Context())
	sess, _ := CurrentSession(r.Context())
	p, err := a.auth.RegisterPasskey(r.Context(), user, sess, cookieValue(r, WebAuthnCookie), body.Credential, body.Label)
	switch {
	case errors.Is(err, auth.ErrInvalidLabel):
		return NewError(http.StatusBadRequest, "invalid_label", "The label can be at most 64 characters long")
	case errors.Is(err, auth.ErrInvalidChallenge):
		a.clearChallengeCookie(w)
		return NewError(http.StatusBadRequest, "invalid_challenge", "The passkey request has expired, try again")
	case errors.Is(err, auth.ErrPasskeyInvalid):
		a.clearChallengeCookie(w)
		return NewError(http.StatusBadRequest, "passkey_invalid", "The passkey could not be verified")
	case errors.Is(err, auth.ErrPasskeyExists):
		a.clearChallengeCookie(w)
		return NewError(http.StatusConflict, "passkey_exists", "This passkey is already registered")
	case err != nil:
		return err
	}
	a.clearChallengeCookie(w)
	WriteJSON(w, http.StatusCreated, p)
	return nil
}

func (a *api) deletePasskey(w http.ResponseWriter, r *http.Request) error {
	user, _ := CurrentUser(r.Context())
	err := a.auth.DeletePasskey(r.Context(), user.ID, r.PathValue("id"))
	if errors.Is(err, auth.ErrPasskeyNotFound) {
		return NewError(http.StatusNotFound, "not_found", "Passkey not found")
	}
	if err != nil {
		return err
	}
	w.WriteHeader(http.StatusNoContent)
	return nil
}
