package httpapi

import (
	"errors"
	"net/http"

	"famodash/internal/dashboards"
)

// dashboardsError maps store errors to API errors; other errors pass through.
func dashboardsError(err error) error {
	switch {
	case errors.Is(err, dashboards.ErrNotFound):
		return NewError(http.StatusNotFound, "not_found", "Dashboard not found")
	case errors.Is(err, dashboards.ErrInvalidTitle):
		return NewError(http.StatusBadRequest, "invalid_title", "Titles need 1 to 100 characters")
	case errors.Is(err, dashboards.ErrInvalidSlug):
		return errInvalidSlug
	case errors.Is(err, dashboards.ErrSlugTaken):
		return errSlugTaken
	}
	return err
}

func (a *api) listDashboards(w http.ResponseWriter, r *http.Request) error {
	user, _ := CurrentUser(r.Context())
	list, err := a.dashboards.List(r.Context(), user.ID)
	if err != nil {
		return err
	}
	WriteJSON(w, http.StatusOK, list)
	return nil
}

func (a *api) createDashboard(w http.ResponseWriter, r *http.Request) error {
	var body struct {
		Title     string `json:"title"`
		Slug      string `json:"slug"`
		IsDefault bool   `json:"isDefault"`
		IsPublic  bool   `json:"isPublic"`
	}
	if err := decodeJSON(r, &body); err != nil {
		return err
	}
	user, _ := CurrentUser(r.Context())
	d, err := a.dashboards.Create(r.Context(), user.ID, dashboards.Fields(body))
	if err != nil {
		return dashboardsError(err)
	}
	WriteJSON(w, http.StatusCreated, d)
	return nil
}

func (a *api) updateDashboard(w http.ResponseWriter, r *http.Request) error {
	var body struct {
		Title     *string `json:"title"`
		Slug      *string `json:"slug"`
		IsDefault *bool   `json:"isDefault"`
		IsPublic  *bool   `json:"isPublic"`
	}
	if err := decodeJSON(r, &body); err != nil {
		return err
	}
	user, _ := CurrentUser(r.Context())
	d, err := a.dashboards.Update(r.Context(), user.ID, r.PathValue("id"), dashboards.Patch(body))
	if err != nil {
		return dashboardsError(err)
	}
	WriteJSON(w, http.StatusOK, d)
	return nil
}

func (a *api) deleteDashboard(w http.ResponseWriter, r *http.Request) error {
	user, _ := CurrentUser(r.Context())
	if err := a.dashboards.Delete(r.Context(), user.ID, r.PathValue("id")); err != nil {
		return dashboardsError(err)
	}
	w.WriteHeader(http.StatusNoContent)
	return nil
}

// viewDashboard resolves a dashboard address for the signed-in user or, without
// a session, for an anonymous visitor.
func (a *api) viewDashboard(w http.ResponseWriter, r *http.Request) error {
	user, _ := CurrentUser(r.Context())
	v, err := a.dashboards.Lookup(r.Context(), r.PathValue("userSlug"), r.PathValue("dashboardSlug"), user.ID)
	if err != nil {
		return dashboardsError(err)
	}
	WriteJSON(w, http.StatusOK, v)
	return nil
}
