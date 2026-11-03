package httpapi

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"
)

type exportDoc struct {
	Format      string                 `json:"format"`
	Version     int                    `json:"version"`
	WithFiles   bool                   `json:"withFiles"`
	Profile     struct{ Email string } `json:"profile"`
	Attachments []struct {
		Name string  `json:"name"`
		File *string `json:"file"`
	} `json:"attachments"`
}

func TestExportJSON(t *testing.T) {
	s := newServer(t)
	ann := s.signIn(t, "ann@example.org")
	s.setSlug(t, ann, "ann")
	s.mustUploadAttachment(t, ann, "report.pdf", testPDF)

	rec := s.req("GET", "/api/me/export", "", withCookie(ann))
	if rec.Code != http.StatusOK {
		t.Fatalf("export: %d %s", rec.Code, rec.Body)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type = %q", ct)
	}
	if cd := rec.Header().Get("Content-Disposition"); cd != `attachment; filename=boardly-export-ann-2026-03-01.json` {
		t.Errorf("Content-Disposition = %q", cd)
	}
	if cc := rec.Header().Get("Cache-Control"); cc != "no-store" {
		t.Errorf("Cache-Control = %q", cc)
	}
	var doc exportDoc
	if err := json.Unmarshal(rec.Body.Bytes(), &doc); err != nil {
		t.Fatal(err)
	}
	if doc.Format != "boardly" || doc.Version != 1 || doc.WithFiles || doc.Profile.Email != "ann@example.org" {
		t.Errorf("document = %+v", doc)
	}
	if len(doc.Attachments) != 1 || doc.Attachments[0].Name != "report.pdf" || doc.Attachments[0].File != nil {
		t.Errorf("attachments = %+v, want report.pdf without file", doc.Attachments)
	}
}

func TestExportZIP(t *testing.T) {
	s := newServer(t)
	ann := s.signIn(t, "ann@example.org")
	a := s.mustUploadAttachment(t, ann, "report.pdf", testPDF)

	rec := s.req("GET", "/api/me/export?files=true", "", withCookie(ann))
	if rec.Code != http.StatusOK {
		t.Fatalf("export: %d %s", rec.Code, rec.Body)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/zip" {
		t.Errorf("Content-Type = %q", ct)
	}
	// Ann has no slug yet.
	if cd := rec.Header().Get("Content-Disposition"); !strings.Contains(cd, "boardly-export-") || !strings.HasSuffix(cd, "-2026-03-01.zip") {
		t.Errorf("Content-Disposition = %q", cd)
	}
	zr, err := zip.NewReader(bytes.NewReader(rec.Body.Bytes()), int64(rec.Body.Len()))
	if err != nil {
		t.Fatal(err)
	}
	names := map[string]bool{}
	for _, f := range zr.File {
		names[f.Name] = true
	}
	path := "attachments/" + a.ID + "/report.pdf"
	if !names["boardly.json"] || !names[path] {
		t.Errorf("archive has %v, want boardly.json and %s", names, path)
	}
}

func TestExportInvalidFiles(t *testing.T) {
	s := newServer(t)
	ann := s.signIn(t, "ann@example.org")
	for range 5 {
		if rec := s.req("GET", "/api/me/export?files=yes", "", withCookie(ann)); rec.Code != http.StatusBadRequest || errorCode(t, rec) != "invalid_request" {
			t.Fatalf("files=yes: %d %s", rec.Code, rec.Body)
		}
	}
	// Rejected requests didn't count.
	if rec := s.req("GET", "/api/me/export", "", withCookie(ann)); rec.Code != http.StatusOK {
		t.Fatalf("export after invalid requests: %d %s", rec.Code, rec.Body)
	}
}

func TestExportRequiresSession(t *testing.T) {
	s := newServer(t)
	ann := s.signIn(t, "ann@example.org")
	tok := s.createToken(t, ann, nil)
	if rec := s.req("GET", "/api/me/export", ""); rec.Code != http.StatusUnauthorized {
		t.Errorf("signed out: %d", rec.Code)
	}
	if rec := s.req("GET", "/api/me/export", "", withBearer(tok.Secret)); rec.Code != http.StatusUnauthorized {
		t.Errorf("app token: %d", rec.Code)
	}
}

func TestExportWhileDeletionScheduled(t *testing.T) {
	s := newServer(t)
	ann := s.signIn(t, "ann@example.org")
	if rec := s.req("DELETE", "/api/me", `{"email":"ann@example.org"}`, withCookie(ann)); rec.Code != http.StatusOK {
		t.Fatalf("schedule: %d %s", rec.Code, rec.Body)
	}
	if rec := s.req("GET", "/api/me/export", "", withCookie(ann)); rec.Code != http.StatusOK {
		t.Fatalf("export: %d %s", rec.Code, rec.Body)
	}
}

func TestExportRateLimit(t *testing.T) {
	s := newServer(t)
	ann := s.signIn(t, "ann@example.org")
	bob := s.signIn(t, "bob@example.org")

	for i, path := range []string{"/api/me/export", "/api/me/export?files=true", "/api/me/export"} {
		if rec := s.req("GET", path, "", withCookie(ann)); rec.Code != http.StatusOK {
			t.Fatalf("export %d: %d %s", i+1, rec.Code, rec.Body)
		}
		s.clock.Advance(time.Minute)
	}
	rec := s.req("GET", "/api/me/export", "", withCookie(ann))
	if rec.Code != http.StatusTooManyRequests || errorCode(t, rec) != "rate_limited" {
		t.Fatalf("fourth export: %d %s", rec.Code, rec.Body)
	}
	// The first export was 3 minutes ago, so the next is allowed in 57 minutes.
	if ra := rec.Header().Get("Retry-After"); ra != "3420" {
		t.Errorf("Retry-After = %q, want 3420", ra)
	}
	if rec := s.req("GET", "/api/me/export", "", withCookie(bob)); rec.Code != http.StatusOK {
		t.Errorf("Bob's export: %d", rec.Code)
	}
	s.clock.Advance(57 * time.Minute)
	if rec := s.req("GET", "/api/me/export", "", withCookie(ann)); rec.Code != http.StatusOK {
		t.Errorf("after the window: %d %s", rec.Code, rec.Body)
	}
}
