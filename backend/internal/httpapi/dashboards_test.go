package httpapi

import (
	"net/http"
	"strings"
	"testing"

	"famodash/internal/dashboards"
)

func (s *server) createDashboard(t *testing.T, cookie string, body map[string]any) dashboards.Dashboard {
	t.Helper()
	rec := s.req("POST", "/api/dashboards", mustJSON(body), withCookie(cookie))
	if rec.Code != http.StatusCreated {
		t.Fatalf("create %v: %d %s", body, rec.Code, rec.Body)
	}
	return decodeJSONBody[dashboards.Dashboard](t, rec)
}

func (s *server) setSlug(t *testing.T, cookie, slug string) {
	t.Helper()
	if rec := s.req("PATCH", "/api/me", `{"slug":"`+slug+`"}`, withCookie(cookie)); rec.Code != http.StatusOK {
		t.Fatalf("set slug: %d %s", rec.Code, rec.Body)
	}
}

func TestDashboardsRequireSession(t *testing.T) {
	s := newServer(t)
	for _, r := range [][2]string{
		{"GET", "/api/dashboards"},
		{"POST", "/api/dashboards"},
		{"PATCH", "/api/dashboards/x"},
		{"DELETE", "/api/dashboards/x"},
		{"PUT", "/api/dashboards/x/layout"},
	} {
		if rec := s.req(r[0], r[1], `{}`); rec.Code != http.StatusUnauthorized {
			t.Errorf("%s %s: %d, want 401", r[0], r[1], rec.Code)
		}
	}
}

func TestDashboardsCRUD(t *testing.T) {
	s := newServer(t)
	ann := s.signIn(t, "ann@example.org")
	bob := s.signIn(t, "bob@example.org")

	home := s.createDashboard(t, ann, map[string]any{"title": "Home", "slug": "Home", "isDefault": true})
	if home.Slug != "home" || !home.IsDefault || home.IsPublic {
		t.Fatalf("created %+v", home)
	}
	work := s.createDashboard(t, ann, map[string]any{"title": "Work", "slug": "work"})

	for body, want := range map[string]struct {
		status int
		code   string
	}{
		`{"title":"X","slug":"home"}`: {http.StatusConflict, "slug_taken"},
		`{"title":"X","slug":"a_b"}`:  {http.StatusBadRequest, "invalid_slug"},
		`{"title":" ","slug":"x"}`:    {http.StatusBadRequest, "invalid_title"},
	} {
		rec := s.req("POST", "/api/dashboards", body, withCookie(ann))
		if rec.Code != want.status || errorCode(t, rec) != want.code {
			t.Errorf("%s: %d %s, want %d %s", body, rec.Code, rec.Body, want.status, want.code)
		}
	}

	rec := s.req("PATCH", "/api/dashboards/"+work.ID, `{"isDefault":true,"isPublic":true}`, withCookie(ann))
	if rec.Code != http.StatusOK {
		t.Fatalf("patch: %d %s", rec.Code, rec.Body)
	}
	list := decodeJSONBody[[]dashboards.Dashboard](t, s.req("GET", "/api/dashboards", "", withCookie(ann)))
	if len(list) != 2 || list[0].IsDefault || !list[1].IsDefault || !list[1].IsPublic {
		t.Fatalf("list = %+v", list)
	}

	if rec := s.req("PATCH", "/api/dashboards/"+work.ID, `{"title":"Mine"}`, withCookie(bob)); rec.Code != http.StatusNotFound || errorCode(t, rec) != "not_found" {
		t.Fatalf("foreign patch: %d %s", rec.Code, rec.Body)
	}
	if rec := s.req("DELETE", "/api/dashboards/"+work.ID, "", withCookie(bob)); rec.Code != http.StatusNotFound {
		t.Fatalf("foreign delete: %d", rec.Code)
	}
	if rec := s.req("DELETE", "/api/dashboards/"+work.ID, "", withCookie(ann)); rec.Code != http.StatusNoContent {
		t.Fatalf("delete: %d %s", rec.Code, rec.Body)
	}
	if list := decodeJSONBody[[]dashboards.Dashboard](t, s.req("GET", "/api/dashboards", "", withCookie(bob))); len(list) != 0 {
		t.Fatalf("bob's list = %+v", list)
	}
}

func TestDashboardColumns(t *testing.T) {
	s := newServer(t)
	ann := s.signIn(t, "ann@example.org")

	home := s.createDashboard(t, ann, map[string]any{"title": "Home", "slug": "home"})
	if home.Columns != dashboards.DefaultColumns {
		t.Fatalf("defaults = %+v", home.Columns)
	}
	wide := s.createDashboard(t, ann, map[string]any{"title": "Wide", "slug": "wide", "desktopColumns": 12, "mobileColumns": 3, "mobileBreakpoint": 1024})
	if wide.Columns != (dashboards.Columns{DesktopColumns: 12, MobileColumns: 3, MobileBreakpoint: 1024}) {
		t.Fatalf("custom = %+v", wide.Columns)
	}

	rec := s.req("PATCH", "/api/dashboards/"+home.ID, `{"desktopColumns":8}`, withCookie(ann))
	if got := decodeJSONBody[dashboards.Dashboard](t, rec); rec.Code != http.StatusOK || got.DesktopColumns != 8 || got.MobileColumns != 2 || got.Title != "Home" {
		t.Fatalf("patch: %d %s", rec.Code, rec.Body)
	}

	for _, r := range [][2]string{
		{"POST", `{"title":"X","slug":"x","desktopColumns":0}`},
		{"POST", `{"title":"X","slug":"x","desktopColumns":4,"mobileColumns":5}`},
		{"POST", `{"title":"X","slug":"x","mobileBreakpoint":800}`},
		{"PATCH", `{"mobileColumns":9}`},
		{"PATCH", `{"desktopColumns":13}`},
	} {
		path := "/api/dashboards"
		if r[0] == "PATCH" {
			path += "/" + home.ID
		}
		rec := s.req(r[0], path, r[1], withCookie(ann))
		if rec.Code != http.StatusBadRequest || errorCode(t, rec) != "invalid_columns" {
			t.Errorf("%s %s: %d %s, want 400 invalid_columns", r[0], r[1], rec.Code, rec.Body)
		}
	}
}

func TestViewDashboard(t *testing.T) {
	s := newServer(t)
	ann := s.signIn(t, "ann@example.org")
	bob := s.signIn(t, "bob@example.org")
	s.setSlug(t, ann, "ann")
	s.createDashboard(t, ann, map[string]any{"title": "Home", "slug": "home", "isPublic": true})
	s.createDashboard(t, ann, map[string]any{"title": "Work", "slug": "work"})

	rec := s.req("GET", "/api/u/Ann/Work", "", withCookie(ann))
	if v := decodeJSONBody[dashboards.View](t, rec); rec.Code != http.StatusOK || !v.IsOwner || v.Dashboard.Title != "Work" || v.OwnerSlug != "ann" || len(v.Dashboards) != 2 {
		t.Fatalf("owner: %d %s", rec.Code, rec.Body)
	}

	for name, opts := range map[string][]func(*http.Request){
		"signed out":     nil,
		"other user":     {withCookie(bob)},
		"invalid cookie": {withCookie("garbage")},
	} {
		rec := s.req("GET", "/api/u/ann/home", "", opts...)
		if v := decodeJSONBody[dashboards.View](t, rec); rec.Code != http.StatusOK || v.IsOwner || len(v.Dashboards) != 1 {
			t.Errorf("%s/public: %d %s", name, rec.Code, rec.Body)
		}
		for _, path := range []string{"/api/u/ann/work", "/api/u/ann/nothing", "/api/u/nobody/home"} {
			if rec := s.req("GET", path, "", opts...); rec.Code != http.StatusNotFound || errorCode(t, rec) != "not_found" {
				t.Errorf("%s %s: %d %s", name, path, rec.Code, rec.Body)
			}
		}
	}
}

func TestSaveLayout(t *testing.T) {
	s := newServer(t)
	ann := s.signIn(t, "ann@example.org")
	bob := s.signIn(t, "bob@example.org")
	s.setSlug(t, ann, "ann")
	home := s.createDashboard(t, ann, map[string]any{"title": "Home", "slug": "home", "isPublic": true})
	path := "/api/dashboards/" + home.ID + "/layout"

	rec := s.req("PUT", path, `{"version":0,"sections":[{"width":3,"title":"Links","border":true,"height":"auto","children":[]},
		{"width":6,"title":null,"border":false,"height":4,"children":[{"width":2,"title":"Tools","border":true,"height":"auto","children":[]}]}]}`, withCookie(ann))
	l := decodeJSONBody[dashboards.Layout](t, rec)
	if rec.Code != http.StatusOK || l.Version != 1 || len(l.Sections) != 2 || len(l.Sections[1].Children) != 1 || l.Sections[1].Height != 4 {
		t.Fatalf("save: %d %s", rec.Code, rec.Body)
	}

	for name, r := range map[string]struct {
		cookie, body string
		status       int
		code         string
	}{
		"stale version":   {ann, `{"version":0,"sections":[]}`, http.StatusConflict, "layout_conflict"},
		"foreign":         {bob, `{"version":1,"sections":[]}`, http.StatusNotFound, "not_found"},
		"unknown field":   {ann, `{"version":1,"sections":[{"width":3,"color":"red"}]}`, http.StatusBadRequest, "invalid_layout"},
		"bad height":      {ann, `{"version":1,"sections":[{"width":3,"height":"tall"}]}`, http.StatusBadRequest, "invalid_layout"},
		"width 13":        {ann, `{"version":1,"sections":[{"width":13}]}`, http.StatusBadRequest, "invalid_layout"},
		"missing version": {ann, `{"sections":[]}`, http.StatusBadRequest, "invalid_layout"},
	} {
		rec := s.req("PUT", path, r.body, withCookie(r.cookie))
		if rec.Code != r.status || errorCode(t, rec) != r.code {
			t.Errorf("%s: %d %s, want %d %s", name, rec.Code, rec.Body, r.status, r.code)
		}
	}

	// The lookup carries the layout; only the owner gets the dashboard ID.
	owner := decodeJSONBody[dashboards.View](t, s.req("GET", "/api/u/ann/home", "", withCookie(ann)))
	if owner.ID != home.ID || owner.LayoutVersion != 1 || len(owner.Sections) != 2 || owner.Columns != dashboards.DefaultColumns {
		t.Fatalf("owner view = %+v", owner)
	}
	rec = s.req("GET", "/api/u/ann/home", "")
	if v := decodeJSONBody[dashboards.View](t, rec); v.ID != "" || len(v.Sections) != 2 || strings.Contains(rec.Body.String(), home.ID) {
		t.Fatalf("visitor view: %s", rec.Body)
	}
}
