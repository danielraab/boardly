package httpapi

import (
	"encoding/json"
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
	case errors.Is(err, dashboards.ErrInvalidLayout):
		return NewError(http.StatusBadRequest, "invalid_layout", err.Error())
	case errors.Is(err, dashboards.ErrLayoutConflict):
		return NewError(http.StatusConflict, "layout_conflict", "The dashboard was changed elsewhere")
	case errors.Is(err, dashboards.ErrInvalidColumns):
		return NewError(http.StatusBadRequest, "invalid_columns",
			"Column counts need 1 to 12 with mobile not above desktop, and the breakpoint 640, 768 or 1024")
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
		Title            string `json:"title"`
		Slug             string `json:"slug"`
		IsDefault        bool   `json:"isDefault"`
		IsPublic         bool   `json:"isPublic"`
		DesktopColumns   *int   `json:"desktopColumns"`
		MobileColumns    *int   `json:"mobileColumns"`
		MobileBreakpoint *int   `json:"mobileBreakpoint"`
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
		Title            *string `json:"title"`
		Slug             *string `json:"slug"`
		IsDefault        *bool   `json:"isDefault"`
		IsPublic         *bool   `json:"isPublic"`
		DesktopColumns   *int    `json:"desktopColumns"`
		MobileColumns    *int    `json:"mobileColumns"`
		MobileBreakpoint *int    `json:"mobileBreakpoint"`
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

// maxLayoutBytes caps a layout request; 500 widgets fit well within it.
const maxLayoutBytes = 256 << 10

// saveLayout replaces a dashboard's layout if the submitted version is current.
func (a *api) saveLayout(w http.ResponseWriter, r *http.Request) error {
	var body struct {
		Version  *int                 `json:"version"`
		Sections []dashboards.Section `json:"sections"`
	}
	dec := json.NewDecoder(http.MaxBytesReader(nil, r.Body, maxLayoutBytes))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&body); err != nil {
		return NewError(http.StatusBadRequest, "invalid_layout", "The layout is not valid: "+err.Error())
	}
	if body.Version == nil || body.Sections == nil {
		return NewError(http.StatusBadRequest, "invalid_layout", "The layout needs a version and sections")
	}
	user, _ := CurrentUser(r.Context())
	l, err := a.dashboards.SaveLayout(r.Context(), user.ID, r.PathValue("id"), *body.Version, body.Sections)
	if err != nil {
		return dashboardsError(err)
	}
	WriteJSON(w, http.StatusOK, l)
	return nil
}
