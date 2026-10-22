package httpapi

import (
	"errors"
	"net/http"
	"time"

	"famodash/internal/apptokens"
)

// appTokensError maps store errors to API errors; other errors pass through.
func appTokensError(err error) error {
	switch {
	case errors.Is(err, apptokens.ErrNotFound):
		return NewError(http.StatusNotFound, "not_found", "App token not found")
	case errors.Is(err, apptokens.ErrInvalidName):
		return NewError(http.StatusBadRequest, "invalid_name", "The name needs 1 to 100 characters")
	case errors.Is(err, apptokens.ErrInvalidApp):
		return NewError(http.StatusBadRequest, "invalid_app", "Unknown app")
	case errors.Is(err, apptokens.ErrInvalidRoot):
		return NewError(http.StatusBadRequest, "invalid_root", "The root must be one of your folders")
	case errors.Is(err, apptokens.ErrInvalidExpiry):
		return errInvalidExpiry
	case errors.Is(err, apptokens.ErrInactive):
		return NewError(http.StatusConflict, "token_inactive", "Only active tokens can be revoked")
	case errors.Is(err, apptokens.ErrActive):
		return NewError(http.StatusConflict, "token_active", "Revoke the token before deleting it")
	}
	return err
}

var errInvalidExpiry = NewError(http.StatusBadRequest, "invalid_expiry", "The expiry must be a time in the future")

func (a *api) listAppTokens(w http.ResponseWriter, r *http.Request) error {
	user, _ := CurrentUser(r.Context())
	list, err := a.appTokens.List(r.Context(), user.ID)
	if err != nil {
		return err
	}
	WriteJSON(w, http.StatusOK, list)
	return nil
}

func (a *api) createAppToken(w http.ResponseWriter, r *http.Request) error {
	var body struct {
		Name      string        `json:"name"`
		App       apptokens.App `json:"app"`
		RootID    string        `json:"rootId"`
		ExpiresAt *string       `json:"expiresAt"`
	}
	if err := decodeJSON(r, &body); err != nil {
		return err
	}
	n := apptokens.NewToken{Name: body.Name, App: body.App, RootID: body.RootID}
	if body.ExpiresAt != nil {
		t, err := time.Parse(time.RFC3339, *body.ExpiresAt)
		if err != nil {
			return errInvalidExpiry
		}
		n.ExpiresAt = &t
	}
	user, _ := CurrentUser(r.Context())
	token, secret, err := a.appTokens.Create(r.Context(), user.ID, n)
	if err != nil {
		return appTokensError(err)
	}
	WriteJSON(w, http.StatusCreated, struct {
		apptokens.Token
		Secret string `json:"secret"`
	}{token, secret})
	return nil
}

func (a *api) revokeAppToken(w http.ResponseWriter, r *http.Request) error {
	user, _ := CurrentUser(r.Context())
	token, err := a.appTokens.Revoke(r.Context(), user.ID, r.PathValue("id"))
	if err != nil {
		return appTokensError(err)
	}
	WriteJSON(w, http.StatusOK, token)
	return nil
}

func (a *api) deleteAppToken(w http.ResponseWriter, r *http.Request) error {
	user, _ := CurrentUser(r.Context())
	if err := a.appTokens.Delete(r.Context(), user.ID, r.PathValue("id")); err != nil {
		return appTokensError(err)
	}
	w.WriteHeader(http.StatusNoContent)
	return nil
}
