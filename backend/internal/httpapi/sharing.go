package httpapi

import (
	"errors"
	"net/http"
	"strconv"

	"boardly/internal/groups"
	"boardly/internal/invitations"
	"boardly/internal/shares"
)

// invitationsPerHour limits how many invitations one user can create.
const invitationsPerHour = 30

func groupsError(err error) error {
	switch {
	case errors.Is(err, groups.ErrNotFound):
		return NewError(http.StatusNotFound, "not_found", "Group not found")
	case errors.Is(err, groups.ErrForbidden):
		return NewError(http.StatusForbidden, "forbidden", "Only group admins can do this")
	case errors.Is(err, groups.ErrInvalidName):
		return NewError(http.StatusBadRequest, "invalid_name", "Group names need 1 to 100 characters")
	case errors.Is(err, groups.ErrInvalidRole):
		return NewError(http.StatusBadRequest, "invalid_role", "The role must be admin or member")
	case errors.Is(err, groups.ErrLastAdmin):
		return NewError(http.StatusConflict, "last_admin", "The group needs another group admin first")
	}
	return err
}

func sharesError(err error) error {
	switch {
	case errors.Is(err, shares.ErrNotFound):
		return NewError(http.StatusNotFound, "not_found", "Not found")
	case errors.Is(err, shares.ErrInvalidLevel):
		return errInvalidShareLevel
	case errors.Is(err, shares.ErrGroupNotFound):
		return NewError(http.StatusNotFound, "group_not_found", "Group not found")
	case errors.Is(err, shares.ErrAlreadyShared):
		return NewError(http.StatusConflict, "already_shared", "Already shared with this group")
	}
	return err
}

var errInvalidShareLevel = NewError(http.StatusBadRequest, "invalid_share_level", "The level must be read")

func invitationsError(err error) error {
	switch {
	case errors.Is(err, invitations.ErrNotFound):
		return NewError(http.StatusNotFound, "not_found", "Not found")
	case errors.Is(err, invitations.ErrInvalidEmail):
		return NewError(http.StatusBadRequest, "invalid_email", "The email address is not valid")
	case errors.Is(err, invitations.ErrInvalidLevel):
		return errInvalidShareLevel
	case errors.Is(err, invitations.ErrAlreadyInvited):
		return NewError(http.StatusConflict, "already_invited", "This address is already invited")
	case errors.Is(err, invitations.ErrCannotSelf):
		return NewError(http.StatusBadRequest, "cannot_invite_self", "You can't invite your own address")
	case errors.Is(err, invitations.ErrInvitationNotFound):
		return NewError(http.StatusNotFound, "invitation_not_found", "This invitation is no longer valid")
	case errors.Is(err, invitations.ErrWrongAccount):
		return NewError(http.StatusForbidden, "invitation_wrong_account", "This invitation was sent to another address")
	}
	return groupsError(err)
}

func (a *api) listGroups(w http.ResponseWriter, r *http.Request) error {
	user, _ := CurrentUser(r.Context())
	list, err := a.groups.List(r.Context(), user.ID)
	if err != nil {
		return err
	}
	WriteJSON(w, http.StatusOK, list)
	return nil
}

func (a *api) createGroup(w http.ResponseWriter, r *http.Request) error {
	var body struct {
		Name string `json:"name"`
	}
	if err := decodeJSON(r, &body); err != nil {
		return err
	}
	user, _ := CurrentUser(r.Context())
	g, err := a.groups.Create(r.Context(), user.ID, body.Name)
	if err != nil {
		return groupsError(err)
	}
	WriteJSON(w, http.StatusCreated, g)
	return nil
}

func (a *api) getGroup(w http.ResponseWriter, r *http.Request) error {
	user, _ := CurrentUser(r.Context())
	d, err := a.groups.Get(r.Context(), user.ID, r.PathValue("id"))
	if err != nil {
		return groupsError(err)
	}
	result := struct {
		groups.Details
		// Invitations are only there for group admins.
		Invitations []invitations.Invitation `json:"invitations,omitzero"`
	}{Details: d}
	if d.Role == groups.RoleAdmin {
		if result.Invitations, err = a.invitations.ListForGroup(r.Context(), d.ID); err != nil {
			return err
		}
	}
	WriteJSON(w, http.StatusOK, result)
	return nil
}

func (a *api) updateGroup(w http.ResponseWriter, r *http.Request) error {
	var body struct {
		Name string `json:"name"`
	}
	if err := decodeJSON(r, &body); err != nil {
		return err
	}
	user, _ := CurrentUser(r.Context())
	g, err := a.groups.Rename(r.Context(), user.ID, r.PathValue("id"), body.Name)
	if err != nil {
		return groupsError(err)
	}
	WriteJSON(w, http.StatusOK, g)
	return nil
}

func (a *api) deleteGroup(w http.ResponseWriter, r *http.Request) error {
	user, _ := CurrentUser(r.Context())
	if err := a.groups.Delete(r.Context(), user.ID, r.PathValue("id")); err != nil {
		return groupsError(err)
	}
	w.WriteHeader(http.StatusNoContent)
	return nil
}

func (a *api) updateGroupMember(w http.ResponseWriter, r *http.Request) error {
	var body struct {
		Role string `json:"role"`
	}
	if err := decodeJSON(r, &body); err != nil {
		return err
	}
	user, _ := CurrentUser(r.Context())
	id := r.PathValue("id")
	if err := a.groups.SetRole(r.Context(), user.ID, id, r.PathValue("userId"), body.Role); err != nil {
		return groupsError(err)
	}
	d, err := a.groups.Get(r.Context(), user.ID, id)
	if err != nil {
		return groupsError(err)
	}
	WriteJSON(w, http.StatusOK, d)
	return nil
}

func (a *api) removeGroupMember(w http.ResponseWriter, r *http.Request) error {
	user, _ := CurrentUser(r.Context())
	if err := a.groups.RemoveMember(r.Context(), user.ID, r.PathValue("id"), r.PathValue("userId")); err != nil {
		return groupsError(err)
	}
	w.WriteHeader(http.StatusNoContent)
	return nil
}

// knownShareType answers 404 for resource types that can't be shared yet.
func knownShareType(r *http.Request) error {
	if r.PathValue("type") != "dashboard" {
		return NewError(http.StatusNotFound, "not_found", "Not found")
	}
	return nil
}

func (a *api) listShares(w http.ResponseWriter, r *http.Request) error {
	if err := knownShareType(r); err != nil {
		return err
	}
	user, _ := CurrentUser(r.Context())
	id := r.PathValue("id")
	list, err := a.shares.List(r.Context(), user.ID, id)
	if err != nil {
		return sharesError(err)
	}
	open, err := a.invitations.ListForDashboard(r.Context(), id)
	if err != nil {
		return err
	}
	WriteJSON(w, http.StatusOK, map[string]any{"shares": list, "invitations": open})
	return nil
}

func (a *api) createShare(w http.ResponseWriter, r *http.Request) error {
	if err := knownShareType(r); err != nil {
		return err
	}
	var body struct {
		GroupID string `json:"groupId"`
		Level   string `json:"level"`
	}
	if err := decodeJSON(r, &body); err != nil {
		return err
	}
	user, _ := CurrentUser(r.Context())
	sh, err := a.shares.ShareWithGroup(r.Context(), user.ID, r.PathValue("id"), body.GroupID, body.Level)
	if err != nil {
		return sharesError(err)
	}
	WriteJSON(w, http.StatusCreated, sh)
	return nil
}

func (a *api) deleteShare(w http.ResponseWriter, r *http.Request) error {
	if err := knownShareType(r); err != nil {
		return err
	}
	user, _ := CurrentUser(r.Context())
	if err := a.shares.Remove(r.Context(), user.ID, r.PathValue("id"), r.PathValue("shareId")); err != nil {
		return sharesError(err)
	}
	w.WriteHeader(http.StatusNoContent)
	return nil
}

func (a *api) sharedWithMe(w http.ResponseWriter, r *http.Request) error {
	user, _ := CurrentUser(r.Context())
	list, err := a.shares.SharedWithMe(r.Context(), user.ID)
	if err != nil {
		return err
	}
	WriteJSON(w, http.StatusOK, map[string]any{"dashboards": list})
	return nil
}

// allowInvitation applies the limit per user, before anything is looked at,
// so it answers the same whatever the address is.
func (a *api) allowInvitation(w http.ResponseWriter, userID string) error {
	if ok, retry := a.invitationLimiter.Allow(userID, a.invitations.Now()); !ok {
		w.Header().Set("Retry-After", strconv.Itoa(retryAfterSeconds(retry)))
		return NewError(http.StatusTooManyRequests, "rate_limited", "Too many invitations, try again later")
	}
	return nil
}

func actor(r *http.Request) invitations.Actor {
	user, _ := CurrentUser(r.Context())
	return invitations.Actor{ID: user.ID, Email: user.Email}
}

func (a *api) inviteToDashboard(w http.ResponseWriter, r *http.Request) error {
	var body struct {
		Email  string `json:"email"`
		Level  string `json:"level"`
		Locale string `json:"locale"`
	}
	if err := decodeJSON(r, &body); err != nil {
		return err
	}
	inviter := actor(r)
	if err := a.allowInvitation(w, inviter.ID); err != nil {
		return err
	}
	inv, err := a.invitations.InviteToDashboard(r.Context(), inviter, r.PathValue("id"), body.Email, body.Level, body.Locale)
	if err != nil {
		return invitationsError(err)
	}
	WriteJSON(w, http.StatusCreated, inv)
	return nil
}

func (a *api) revokeDashboardInvitation(w http.ResponseWriter, r *http.Request) error {
	user, _ := CurrentUser(r.Context())
	if err := a.invitations.RevokeDashboard(r.Context(), user.ID, r.PathValue("id"), r.PathValue("invitationId")); err != nil {
		return invitationsError(err)
	}
	w.WriteHeader(http.StatusNoContent)
	return nil
}

func (a *api) inviteToGroup(w http.ResponseWriter, r *http.Request) error {
	var body struct {
		Email  string `json:"email"`
		Role   string `json:"role"`
		Locale string `json:"locale"`
	}
	if err := decodeJSON(r, &body); err != nil {
		return err
	}
	inviter := actor(r)
	if err := a.allowInvitation(w, inviter.ID); err != nil {
		return err
	}
	inv, err := a.invitations.InviteToGroup(r.Context(), inviter, r.PathValue("id"), body.Email, body.Role, body.Locale)
	if err != nil {
		return invitationsError(err)
	}
	WriteJSON(w, http.StatusCreated, inv)
	return nil
}

func (a *api) revokeGroupInvitation(w http.ResponseWriter, r *http.Request) error {
	user, _ := CurrentUser(r.Context())
	if err := a.invitations.RevokeGroup(r.Context(), user.ID, r.PathValue("id"), r.PathValue("invitationId")); err != nil {
		return invitationsError(err)
	}
	w.WriteHeader(http.StatusNoContent)
	return nil
}

func (a *api) listInvitations(w http.ResponseWriter, r *http.Request) error {
	list, err := a.invitations.ListOpen(r.Context(), actor(r).Email)
	if err != nil {
		return err
	}
	WriteJSON(w, http.StatusOK, list)
	return nil
}

func (a *api) previewInvitation(w http.ResponseWriter, r *http.Request) error {
	var body struct {
		Token string `json:"token"`
	}
	if err := decodeJSON(r, &body); err != nil {
		return err
	}
	p, err := a.invitations.Preview(r.Context(), actor(r), body.Token)
	if err != nil {
		return invitationsError(err)
	}
	WriteJSON(w, http.StatusOK, p)
	return nil
}

func (a *api) acceptInvitation(w http.ResponseWriter, r *http.Request) error {
	var body struct {
		Token        string `json:"token"`
		InvitationID string `json:"invitationId"`
	}
	if err := decodeJSON(r, &body); err != nil {
		return err
	}
	if (body.Token == "") == (body.InvitationID == "") {
		return NewError(http.StatusBadRequest, "invalid_request", "Send either a token or an invitation ID")
	}
	result, err := a.invitations.Accept(r.Context(), actor(r), body.Token, body.InvitationID)
	if err != nil {
		return invitationsError(err)
	}
	WriteJSON(w, http.StatusOK, result)
	return nil
}
