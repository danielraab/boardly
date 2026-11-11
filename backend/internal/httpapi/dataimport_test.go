package httpapi

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func (s *server) importData(cookie string, params map[string]string, body string, opts ...func(*http.Request)) *httptest.ResponseRecorder {
	q := url.Values{}
	for k, v := range params {
		q.Set(k, v)
	}
	all := []func(*http.Request){func(r *http.Request) { r.Header.Set("Content-Type", "application/octet-stream") }}
	if cookie != "" {
		all = append(all, withCookie(cookie))
	}
	return s.req("POST", "/api/import?"+q.Encode(), body, append(all, opts...)...)
}

type dataImportReport struct {
	Favourites *struct {
		Links int `json:"links"`
	} `json:"favourites"`
	Notes *struct {
		Notes int `json:"notes"`
	} `json:"notes"`
	Dashboards *struct {
		Dashboards int `json:"dashboards"`
	} `json:"dashboards"`
	Media *struct {
		Sources int `json:"sources"`
	} `json:"media"`
	Files struct {
		Attachments int `json:"attachments"`
	} `json:"files"`
}

func TestImportDataRoundTrip(t *testing.T) {
	s := newServer(t)
	ann := s.signIn(t, "ann@example.org")
	bob := s.signIn(t, "bob@example.org")
	root := s.tree(t, ann).ID
	s.create(t, ann, map[string]any{"parentId": root, "kind": "link", "url": "https://go.dev"})
	a := s.mustUploadAttachment(t, ann, "report.pdf", testPDF)
	note := `{"title":"Plan","content":{"v":1,"blocks":[{"type":"attachment","id":"` + a.ID + `"}]}}`
	if rec := s.req("POST", "/api/notes", note, withCookie(ann)); rec.Code != http.StatusCreated {
		t.Fatalf("note: %d %s", rec.Code, rec.Body)
	}
	s.createDashboard(t, ann, map[string]any{"title": "Home", "slug": "home", "isPublic": true})
	s.createStation(t, ann, "Radio", "https://radio.example.org/stream")

	for _, files := range []string{"false", "true"} {
		export := s.req("GET", "/api/me/export?files="+files, "", withCookie(ann))
		if export.Code != http.StatusOK {
			t.Fatalf("export: %d %s", export.Code, export.Body)
		}
		bobRoot := s.tree(t, bob).ID
		rec := s.importData(bob, map[string]string{"parts": "favourites,dashboards,notes,media", "folderId": bobRoot}, export.Body.String())
		if rec.Code != http.StatusOK {
			t.Fatalf("import files=%s: %d %s", files, rec.Code, rec.Body)
		}
		var report dataImportReport
		if err := json.Unmarshal(rec.Body.Bytes(), &report); err != nil {
			t.Fatal(err)
		}
		if report.Favourites == nil || report.Notes == nil || report.Dashboards == nil || report.Media == nil || report.Dashboards.Dashboards != 1 {
			t.Fatalf("files=%s: report = %s", files, rec.Body)
		}
		if files == "true" && report.Files.Attachments != 1 {
			t.Errorf("files=true: attachments = %d", report.Files.Attachments)
		}
	}
	// Second import skipped the duplicates, but added the dashboard again.
	if rec := s.req("GET", "/api/dashboards", "", withCookie(bob)); !strings.Contains(rec.Body.String(), `"home-2"`) {
		t.Errorf("Bob's dashboards: %s", rec.Body)
	}
}

func TestImportDataErrors(t *testing.T) {
	s := newServer(t)
	ann := s.signIn(t, "ann@example.org")
	root := s.tree(t, ann).ID
	export := s.req("GET", "/api/me/export", "", withCookie(ann)).Body.String()
	tok := s.createToken(t, ann, nil)

	for name, c := range map[string]struct {
		params map[string]string
		body   string
		status int
		code   string
	}{
		"no parts":         {map[string]string{}, export, 400, "invalid_request"},
		"unknown part":     {map[string]string{"parts": "favourites,passkeys", "folderId": root}, export, 400, "invalid_request"},
		"bad skip":         {map[string]string{"parts": "notes", "skipDuplicates": "yes"}, export, 400, "invalid_request"},
		"empty body":       {map[string]string{"parts": "notes"}, "", 400, "invalid_request"},
		"no folder":        {map[string]string{"parts": "favourites"}, export, 400, "invalid_folder"},
		"foreign folder":   {map[string]string{"parts": "favourites", "folderId": "nope"}, export, 400, "invalid_folder"},
		"unknown category": {map[string]string{"parts": "notes", "categoryId": "nope"}, export, 400, "invalid_category"},
		"karakeep file":    {map[string]string{"parts": "notes"}, `{"bookmarks":[]}`, 400, "invalid_import_file"},
		"version 4":        {map[string]string{"parts": "notes"}, strings.Replace(export, `"version": 3`, `"version": 4`, 1), 400, "invalid_import_file"},
	} {
		rec := s.importData(ann, c.params, c.body)
		if rec.Code != c.status || errorCode(t, rec) != c.code {
			t.Errorf("%s: %d %s, want %d %s", name, rec.Code, rec.Body, c.status, c.code)
		}
	}

	big := string(bytes.Repeat([]byte("x"), 10<<20+1))
	if rec := s.importData(ann, map[string]string{"parts": "notes"}, big); rec.Code != http.StatusRequestEntityTooLarge || errorCode(t, rec) != "file_too_large" {
		t.Errorf("too large: %d %s", rec.Code, rec.Body)
	}
	if rec := s.importData(ann, map[string]string{"parts": "notes"}, export, func(r *http.Request) {
		r.Header.Set("Content-Type", "multipart/form-data; boundary=x")
	}); rec.Code != http.StatusForbidden {
		t.Errorf("form post: %d", rec.Code)
	}
	if rec := s.importData("", map[string]string{"parts": "notes"}, export); rec.Code != http.StatusUnauthorized {
		t.Errorf("signed out: %d", rec.Code)
	}
	if rec := s.importData("", map[string]string{"parts": "notes"}, export, withBearer(tok.Secret)); rec.Code != http.StatusUnauthorized {
		t.Errorf("app token: %d", rec.Code)
	}
}
