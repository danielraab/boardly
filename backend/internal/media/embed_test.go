package media

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"boardly/internal/notes"
)

const (
	albumID    = "4aawyAB9vmqN3uQ7FjRGTy"
	playlistID = "37i9dQZF1DXcBWIGoYBM5M"
	albumLink  = "https://open.spotify.com/album/" + albumID
)

func TestParseEmbed(t *testing.T) {
	accepted := map[string]string{
		albumLink:                          albumLink,
		"  " + albumLink + "\n":            albumLink,
		albumLink + "/":                    albumLink,
		albumLink + "?si=abc&utm_source=x": albumLink,
		"https://open.spotify.com/playlist/" + playlistID:                     "https://open.spotify.com/playlist/" + playlistID,
		"https://open.spotify.com/intl-de/playlist/" + playlistID + "?si=abc": "https://open.spotify.com/playlist/" + playlistID,
		"spotify:album:" + albumID:                                            albumLink,
		"spotify:playlist:" + playlistID:                                      "https://open.spotify.com/playlist/" + playlistID,
	}
	for link, want := range accepted {
		e, err := ParseEmbed(link)
		if err != nil {
			t.Errorf("%q: %v", link, err)
			continue
		}
		if e.URL != want || e.Provider != "spotify" || e.Shape != ShapeAudio {
			t.Errorf("%q = %+v, want URL %s", link, e, want)
		}
	}

	rejected := []string{
		"",
		"not a link",
		"http://open.spotify.com/album/" + albumID,
		"https://open.spotify.com/track/" + albumID,
		"spotify:track:" + albumID,
		"https://open.spotify.com/artist/" + albumID,
		"https://open.spotify.com/album/short",
		"https://open.spotify.com/album/" + albumID + "x",
		"https://open.spotify.com/album/" + albumID + "/extra",
		"https://open.spotify.com/embed/album/" + albumID,
		"https://spotify.link/abc",
		"https://example.org/album/" + albumID,
		"https://open.spotify.com.example.org/album/" + albumID,
		"https://open.spotify.com:8443/album/" + albumID,
		"https://user@open.spotify.com/album/" + albumID,
	}
	for _, link := range rejected {
		if e, err := ParseEmbed(link); !errors.Is(err, ErrEmbedUnsupported) {
			t.Errorf("%q = %+v, %v; want unsupported", link, e, err)
		}
	}
}

func TestParseEmbedBuildsAddresses(t *testing.T) {
	e, err := ParseEmbed("https://open.spotify.com/intl-de/album/" + albumID + "?evil=1")
	if err != nil {
		t.Fatal(err)
	}
	if want := "https://open.spotify.com/embed/album/" + albumID; e.PlayerURL != want {
		t.Errorf("player = %s, want %s", e.PlayerURL, want)
	}
	want := "https://open.spotify.com/oembed?url=https%3A%2F%2Fopen.spotify.com%2Falbum%2F" + albumID
	if e.InfoURL != want {
		t.Errorf("info = %s, want %s", e.InfoURL, want)
	}
}

func TestEmbedFetcher(t *testing.T) {
	handler := http.NewServeMux()
	handler.HandleFunc("/ok", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"title":"Global Warming","thumbnail_url":"https://i.example/c.jpg","html":"<iframe>"}`))
	})
	handler.HandleFunc("/gone", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNotFound) })
	handler.HandleFunc("/huge", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"title":"` + strings.Repeat("x", MaxEmbedInfoSize) + `"}`))
	})
	handler.HandleFunc("/broken", func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte(`<html>`)) })
	handler.HandleFunc("/slow", func(_ http.ResponseWriter, r *http.Request) {
		select {
		case <-r.Context().Done():
		case <-time.After(2 * time.Second):
		}
	})
	srv := httptest.NewServer(handler)
	defer srv.Close()
	ctx := context.Background()

	f := NewEmbedFetcher(true)
	f.Timeout = 200 * time.Millisecond
	info, err := f.Fetch(ctx, srv.URL+"/ok")
	if err != nil || info.Title != "Global Warming" || info.CoverURL != "https://i.example/c.jpg" {
		t.Errorf("ok = %+v, %v", info, err)
	}
	for _, path := range []string{"/gone", "/huge", "/broken", "/slow"} {
		if info, err := f.Fetch(ctx, srv.URL+path); err == nil {
			t.Errorf("%s = %+v, want an error", path, info)
		}
	}

	if _, err := NewEmbedFetcher(false).Fetch(ctx, srv.URL+"/ok"); err == nil {
		t.Error("a private address was fetched")
	}
}

func (f *fixture) coverImage() string {
	f.t.Helper()
	const img = "cover-image"
	if _, err := f.db.ExecContext(f.ctx,
		`INSERT INTO images (id, source, user_id, content_type, size, created_at) VALUES (?, 'upload', ?, 'image/png', 1, '2026-01-01T00:00:00Z')`,
		img, f.user); err != nil {
		f.t.Fatal(err)
	}
	return img
}

func TestCreateEmbed(t *testing.T) {
	t.Run("title and cover from the provider", func(t *testing.T) {
		f := newFixture(t)
		img := f.coverImage()
		f.info = EmbedInfo{Title: "  Global Warming ", CoverURL: "https://i.example/c.jpg"}
		f.logos["https://i.example/c.jpg"] = img
		src, err := f.s.Create(f.ctx, f.user, NewSource{Kind: KindEmbed, URL: albumLink + "?si=x", LogoURL: "https://ignored.example/l.png"})
		if err != nil {
			t.Fatal(err)
		}
		if src.Title != "Global Warming" || src.URL != albumLink || src.Kind != KindEmbed {
			t.Errorf("source = %+v", src)
		}
		if src.Icon == nil || src.Icon.ImageID != img {
			t.Errorf("icon = %+v", src.Icon)
		}
		if src.Provider != "spotify" || src.EmbedURL != "https://open.spotify.com/embed/album/"+albumID || src.Shape != ShapeAudio {
			t.Errorf("embed fields = %+v", src)
		}
		if len(f.infoCalls) != 1 {
			t.Errorf("info calls = %v", f.infoCalls)
		}
	})

	t.Run("own title keeps it and still gets the cover", func(t *testing.T) {
		f := newFixture(t)
		img := f.coverImage()
		f.info = EmbedInfo{Title: "Theirs", CoverURL: "https://i.example/c.jpg"}
		f.logos["https://i.example/c.jpg"] = img
		src, err := f.s.Create(f.ctx, f.user, NewSource{Kind: KindEmbed, Title: "Focus", URL: albumLink})
		if err != nil {
			t.Fatal(err)
		}
		if src.Title != "Focus" || src.Icon == nil || src.Icon.ImageID != img {
			t.Errorf("source = %+v", src)
		}
	})

	t.Run("own title and icon ask nothing", func(t *testing.T) {
		f := newFixture(t)
		src, err := f.s.Create(f.ctx, f.user, NewSource{Kind: KindEmbed, Title: "Focus", URL: albumLink, Icon: &notes.Icon{Type: notes.IconSymbol, Name: "music"}})
		if err != nil {
			t.Fatal(err)
		}
		if len(f.infoCalls) != 0 || src.Icon == nil || src.Icon.Name != "music" {
			t.Errorf("calls = %v, source = %+v", f.infoCalls, src)
		}
	})

	t.Run("provider down without a title", func(t *testing.T) {
		f := newFixture(t)
		f.infoErr = errors.New("down")
		_, err := f.s.Create(f.ctx, f.user, NewSource{Kind: KindEmbed, Title: "   ", URL: albumLink})
		if !errors.Is(err, ErrEmbedUnreachable) {
			t.Fatalf("err = %v", err)
		}
		if got := f.titles(f.user); len(got) != 0 {
			t.Errorf("saved = %v", got)
		}
	})

	t.Run("provider down with a title", func(t *testing.T) {
		f := newFixture(t)
		f.infoErr = errors.New("down")
		src, err := f.s.Create(f.ctx, f.user, NewSource{Kind: KindEmbed, Title: "Focus", URL: albumLink})
		if err != nil {
			t.Fatal(err)
		}
		if src.Title != "Focus" || src.Icon != nil {
			t.Errorf("source = %+v", src)
		}
	})

	t.Run("provider without a usable title", func(t *testing.T) {
		f := newFixture(t)
		f.info = EmbedInfo{Title: "  "}
		if _, err := f.s.Create(f.ctx, f.user, NewSource{Kind: KindEmbed, URL: albumLink}); !errors.Is(err, ErrEmbedUnreachable) {
			t.Fatalf("err = %v", err)
		}
	})

	t.Run("long title from the provider is cut", func(t *testing.T) {
		f := newFixture(t)
		f.info = EmbedInfo{Title: strings.Repeat("é", MaxTitle+20)}
		src, err := f.s.Create(f.ctx, f.user, NewSource{Kind: KindEmbed, URL: albumLink})
		if err != nil {
			t.Fatal(err)
		}
		if got := len([]rune(src.Title)); got != MaxTitle {
			t.Errorf("title has %d characters", got)
		}
	})

	t.Run("unsupported link asks nothing", func(t *testing.T) {
		f := newFixture(t)
		_, err := f.s.Create(f.ctx, f.user, NewSource{Kind: KindEmbed, URL: "https://example.org/album/" + albumID})
		if !errors.Is(err, ErrEmbedUnsupported) || len(f.infoCalls) != 0 {
			t.Fatalf("err = %v, calls = %v", err, f.infoCalls)
		}
	})

	t.Run("a stream link isn't checked as an embed", func(t *testing.T) {
		f := newFixture(t)
		src := f.create("FM4")
		data, err := json.Marshal(src)
		if err != nil {
			t.Fatal(err)
		}
		for _, field := range []string{"provider", "embedUrl", "shape"} {
			if strings.Contains(string(data), field) {
				t.Errorf("a stream has %s: %s", field, data)
			}
		}
	})
}

func TestUpdateEmbedLink(t *testing.T) {
	f := newFixture(t)
	f.info = EmbedInfo{Title: "Global Warming"}
	src, err := f.s.Create(f.ctx, f.user, NewSource{Kind: KindEmbed, URL: albumLink, Icon: &notes.Icon{Type: notes.IconSymbol, Name: "music"}})
	if err != nil {
		t.Fatal(err)
	}
	calls := len(f.infoCalls)
	link := "https://open.spotify.com/playlist/" + playlistID + "?si=x"
	got, err := f.s.Update(f.ctx, f.user, src.ID, Patch{URL: &link})
	if err != nil {
		t.Fatal(err)
	}
	if got.URL != "https://open.spotify.com/playlist/"+playlistID || got.EmbedURL != "https://open.spotify.com/embed/playlist/"+playlistID {
		t.Errorf("source = %+v", got)
	}
	if got.Title != "Global Warming" || got.Icon == nil || got.Icon.Name != "music" || len(f.infoCalls) != calls {
		t.Errorf("title or icon changed: %+v, calls %v", got, f.infoCalls)
	}
	bad := "https://example.org/"
	if _, err := f.s.Update(f.ctx, f.user, src.ID, Patch{URL: &bad}); !errors.Is(err, ErrEmbedUnsupported) {
		t.Errorf("err = %v", err)
	}
}

func TestInsertEmbed(t *testing.T) {
	f := newFixture(t)
	id, err := f.s.InsertTx(f.ctx, f.db, f.user, Imported{Kind: KindEmbed, Title: "Album", URL: albumLink + "?si=x"})
	if err != nil {
		t.Fatal(err)
	}
	src, err := f.s.Get(f.ctx, f.user, id)
	if err != nil {
		t.Fatal(err)
	}
	if src.URL != albumLink || src.EmbedURL != "https://open.spotify.com/embed/album/"+albumID {
		t.Errorf("source = %+v", src)
	}
	if len(f.infoCalls) != 0 {
		t.Errorf("the provider was asked: %v", f.infoCalls)
	}
	if _, err := f.s.InsertTx(f.ctx, f.db, f.user, Imported{Kind: KindEmbed, Title: "X", URL: "https://example.org/player"}); !errors.Is(err, ErrEmbedUnsupported) {
		t.Errorf("err = %v", err)
	}
}
