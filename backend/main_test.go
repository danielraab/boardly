package main

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"boardly/internal/dbtest"
	"boardly/internal/favourites"
	"boardly/internal/images"
	"boardly/internal/jobs"
	"boardly/internal/offline"
	"boardly/internal/users"
)

// TestStartJobsRunsLeftoverJobs starts the background work on a database
// that a stopped server left with a pending favicon fetch and an interrupted
// preview fetch, and checks that both run.
func TestStartJobsRunsLeftoverJobs(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	database := dbtest.Open(t)
	site := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/favicon.ico":
			_, _ = w.Write(append([]byte{0, 0, 1, 0, 1, 0}, make([]byte, 32)...))
		case "/article":
			w.Header().Set("Content-Type", "text/html")
			_, _ = w.Write([]byte(`<meta property="og:description" content="Left over"><title>Left over</title><article><p>` +
				strings.Repeat("A paragraph of a long article that has more than enough text. ", 10) + `</p></article>`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer site.Close()

	ann, err := users.NewStore(database).EnsureByEmail(ctx, database, "ann@example.org")
	if err != nil {
		t.Fatal(err)
	}
	link, err := favourites.NewStore(database).Create(ctx, ann.ID, favourites.NewNode{Kind: favourites.KindLink, URL: ptr(site.URL + "/article")})
	if err != nil {
		t.Fatal(err)
	}
	const ts = "2026-01-01T00:00:00.000Z"
	insert := `INSERT INTO jobs (kind, dedup_key, payload, attempts, created_at, started_at) VALUES (?, ?, ?, ?, ?, ?)`
	if _, err := database.Exec(insert, jobs.KindFavicon, site.URL, `{"url":"`+site.URL+`/article"}`, 0, ts, nil); err != nil {
		t.Fatal(err)
	}
	if _, err := database.Exec(insert, jobs.KindPreviewFetch, link.ID,
		`{"userId":"`+ann.ID+`","nodeId":"`+link.ID+`","url":"`+site.URL+`/article"}`, 1, ts, ts); err != nil {
		t.Fatal(err)
	}

	// An interrupted offline capture.
	if _, err := database.Exec(`INSERT INTO offline_copies (node_id, user_id, url, status, updated_at) VALUES (?, ?, ?, 'pending', ?)`,
		link.ID, ann.ID, site.URL+"/article", ts); err != nil {
		t.Fatal(err)
	}
	if _, err := database.Exec(insert, jobs.KindOfflineCapture, link.ID,
		`{"userId":"`+ann.ID+`","nodeId":"`+link.ID+`","url":"`+site.URL+`/article","setPreview":true}`, 1, ts, ts); err != nil {
		t.Fatal(err)
	}

	dataDir := filepath.Join(t.TempDir(), "data")
	store, err := images.NewStore(database, dataDir)
	if err != nil {
		t.Fatal(err)
	}
	queue := jobs.New(database)
	favicons := images.NewFavicons(store, true)
	favicons.Jobs = queue
	previews, err := images.NewPreviews(store, queue, favicons.Client, 1<<20, images.Converter{MaxPixels: 1 << 20, Width: 800})
	if err != nil {
		t.Fatal(err)
	}
	offlineStore, err := offline.NewStore(database, dataDir)
	if err != nil {
		t.Fatal(err)
	}
	capturer := &offline.Capturer{Store: offlineStore, Jobs: queue, Client: favicons.Client, MaxBytes: 1 << 20}
	if err := startJobs(ctx, queue, favicons, previews, capturer); err != nil {
		t.Fatal(err)
	}

	deadline := time.Now().Add(5 * time.Second)
	for {
		var icon, description, status sql.NullString
		_ = database.QueryRow(`SELECT image_id FROM favicons WHERE origin = ?`, site.URL).Scan(&icon)
		_ = database.QueryRow(`SELECT status FROM offline_copies WHERE node_id = ?`, link.ID).Scan(&status)
		if err := database.QueryRow(`SELECT description FROM favourite_nodes WHERE id = ?`, link.ID).Scan(&description); err != nil {
			t.Fatal(err)
		}
		if icon.Valid && description.String == "Left over" && status.String == "ready" {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("favicon %v, description %v, offline copy %v; want all from the leftover jobs", icon, description, status)
		}
		time.Sleep(20 * time.Millisecond)
	}
}

func ptr(s string) *string { return &s }

// TestLoadConfigChecksLegalPages checks that a broken legal directory stops
// the server before it starts, with the file named.
func TestLoadConfigChecksLegalPages(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "impressum.md"), []byte("# Impressum"), 0o600); err != nil {
		t.Fatal(err)
	}
	env := map[string]string{"BOARDLY_BASE_URL": "http://localhost:8080", "BOARDLY_LEGAL_DIR": dir}
	lookup := func(k string) (string, bool) { v, ok := env[k]; return v, ok }

	if _, _, err := loadConfig(lookup); err == nil || !strings.Contains(err.Error(), "impressum.md") {
		t.Fatalf("loadConfig error = %v, want one naming impressum.md", err)
	}

	if err := os.Rename(filepath.Join(dir, "impressum.md"), filepath.Join(dir, "imprint.de.md")); err != nil {
		t.Fatal(err)
	}
	_, pages, err := loadConfig(lookup)
	if err != nil {
		t.Fatal(err)
	}
	if pages.Href("imprint") != "/imprint" {
		t.Fatalf("imprint href = %q", pages.Href("imprint"))
	}
}

// TestLoadConfigChecksConsent checks that analytics consent stops the server
// when the snippet would run without consent or no privacy policy exists.
func TestLoadConfigChecksConsent(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "privacy.md"), []byte("# Privacy"), 0o600); err != nil {
		t.Fatal(err)
	}
	marked := `<script type="text/plain" data-category="analytics" src="https://stats.example.org/a.js"></script>`
	tests := []struct {
		name    string
		env     map[string]string
		wantErr string
	}{
		{"unmarked snippet", map[string]string{
			"BOARDLY_LEGAL_DIR": dir, "BOARDLY_CONSENT_ANALYTICS": "Matomo",
			"BOARDLY_ANALYTICS_SNIPPET": `<script src="https://stats.example.org/a.js"></script>`,
		}, "BOARDLY_ANALYTICS_SNIPPET"},
		{"no privacy policy", map[string]string{
			"BOARDLY_CONSENT_ANALYTICS": "Matomo", "BOARDLY_ANALYTICS_SNIPPET": marked,
		}, "privacy policy"},
		{"valid", map[string]string{
			"BOARDLY_LEGAL_DIR": dir, "BOARDLY_CONSENT_ANALYTICS": "Matomo", "BOARDLY_ANALYTICS_SNIPPET": marked,
		}, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.env["BOARDLY_BASE_URL"] = "http://localhost:8080"
			lookup := func(k string) (string, bool) { v, ok := tt.env[k]; return v, ok }
			_, _, err := loadConfig(lookup)
			if tt.wantErr == "" {
				if err != nil {
					t.Fatal(err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("loadConfig error = %v, want one naming %s", err, tt.wantErr)
			}
		})
	}
}
