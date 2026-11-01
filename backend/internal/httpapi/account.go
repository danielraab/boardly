package httpapi

import (
	"errors"
	"net/http"

	"boardly/internal/accounts"
	"boardly/internal/users"
)

// deleteMe schedules the deletion of the signed-in user's account.
func (a *api) deleteMe(w http.ResponseWriter, r *http.Request) error {
	var body struct {
		Email  string `json:"email"`
		Locale string `json:"locale"`
	}
	if err := decodeJSON(r, &body); err != nil {
		return err
	}
	sess, _ := CurrentSession(r.Context())
	user, err := a.accounts.Schedule(r.Context(), sess, body.Email, body.Locale)
	switch {
	case errors.Is(err, accounts.ErrWindowClosed):
		return NewError(http.StatusForbidden, "deletion_window_closed", "Sign in again with an email link to delete your account")
	case errors.Is(err, accounts.ErrEmailMismatch):
		return NewError(http.StatusBadRequest, "email_mismatch", "The email address doesn't match your account")
	case errors.Is(err, accounts.ErrScheduled):
		return NewError(http.StatusConflict, "deletion_scheduled", "The deletion of your account is already scheduled")
	case errors.Is(err, users.ErrNotFound):
		return errUnauthenticated
	case err != nil:
		return err
	}
	a.writeMe(w, r, user)
	return nil
}

// restoreMe cancels the scheduled deletion of the signed-in user's account.
func (a *api) restoreMe(w http.ResponseWriter, r *http.Request) error {
	current, _ := CurrentUser(r.Context())
	user, err := a.accounts.Cancel(r.Context(), current.ID)
	switch {
	case errors.Is(err, accounts.ErrNotScheduled):
		return NewError(http.StatusConflict, "deletion_not_scheduled", "No deletion of your account is scheduled")
	case errors.Is(err, users.ErrNotFound):
		return errUnauthenticated
	case err != nil:
		return err
	}
	a.writeMe(w, r, user)
	return nil
}
