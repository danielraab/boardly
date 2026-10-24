package main

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"boardly/internal/dbtest"
	"boardly/internal/favourites"
	"boardly/internal/images"
	"boardly/internal/jobs"
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
			_, _ = w.Write([]byte(`<meta property="og:description" content="Left over">`))
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

	store, err := images.NewStore(database, filepath.Join(t.TempDir(), "data"))
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
	if err := startJobs(ctx, queue, favicons, previews); err != nil {
		t.Fatal(err)
	}

	deadline := time.Now().Add(5 * time.Second)
	for {
		var icon, description sql.NullString
		_ = database.QueryRow(`SELECT image_id FROM favicons WHERE origin = ?`, site.URL).Scan(&icon)
		if err := database.QueryRow(`SELECT description FROM favourite_nodes WHERE id = ?`, link.ID).Scan(&description); err != nil {
			t.Fatal(err)
		}
		if icon.Valid && description.String == "Left over" {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("favicon %v, description %v; want both from the leftover jobs", icon, description)
		}
		time.Sleep(20 * time.Millisecond)
	}
}

func ptr(s string) *string { return &s }
