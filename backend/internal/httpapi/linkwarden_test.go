package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"image"
	"image/png"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"boardly/internal/images"
)

const lwV1 = linkwardenPrefix + "/api/v1"

// linkwardenClient is a Linkwarden token rooted at a new folder "Reading".
type linkwardenClient struct {
	s      *server
	secret string
	rootID string
}

func (s *server) linkwardenToken(t *testing.T, cookie string) linkwardenClient {
	t.Helper()
	folder := s.create(t, cookie, map[string]any{"kind": "folder", "title": "Reading"})
	rec := s.req("POST", "/api/app-tokens", mustJSON(map[string]any{
		"name": "Phone", "app": "linkwarden", "rootId": folder.ID, "expiresAt": nil,
	}), withCookie(cookie))
	if rec.Code != http.StatusCreated {
		t.Fatalf("create token: %d %s", rec.Code, rec.Body)
	}
	return linkwardenClient{s: s, secret: decodeJSONBody[createdToken](t, rec).Secret, rootID: folder.ID}
}

// do sends a request as the Linkwarden clients do: a bearer token, no cookie.
func (c linkwardenClient) do(method, path, body string) *httptest.ResponseRecorder {
	return c.s.req(method, path, body, withBearer(c.secret), fromExtension)
}

// envelope is a Linkwarden response.
type envelope struct {
	Response json.RawMessage `json:"response"`
	Data     json.RawMessage `json:"data"`
}

func lwDecode[T any](t *testing.T, rec *httptest.ResponseRecorder, wantStatus int) T {
	t.Helper()
	if rec.Code != wantStatus {
		t.Fatalf("status %d, want %d: %s", rec.Code, wantStatus, rec.Body)
	}
	env := decodeJSONBody[envelope](t, rec)
	raw := env.Response
	if env.Data != nil {
		raw = env.Data
	}
	var v T
	if err := json.Unmarshal(raw, &v); err != nil {
		t.Fatalf("decode %s: %v", raw, err)
	}
	return v
}

type lwLink struct {
	ID           int64  `json:"id"`
	Name         string `json:"name"`
	URL          string `json:"url"`
	CollectionID int64  `json:"collectionId"`
	Preview      string `json:"preview"`
	Tags         []struct {
		ID   int64  `json:"id"`
		Name string `json:"name"`
	} `json:"tags"`
	Collection struct {
		ID      int64 `json:"id"`
		OwnerID int64 `json:"ownerId"`
	} `json:"collection"`
}

type lwCollection struct {
	ID       int64  `json:"id"`
	Name     string `json:"name"`
	ParentID *int64 `json:"parentId"`
}

type lwLinks struct {
	Links      []lwLink `json:"links"`
	NextCursor *int64   `json:"nextCursor"`
}

func TestLinkwardenExtensionFlow(t *testing.T) {
	s := newServer(t)
	ann := s.signIn(t, "ann@example.org")
	c := s.linkwardenToken(t, ann)

	// Options page: version check, collections and tags.
	config := lwDecode[map[string]any](t, c.do("GET", lwV1+"/config", ""), http.StatusOK)
	if config["INSTANCE_VERSION"] != "v2.16.3" {
		t.Errorf("config = %v", config)
	}
	collections := lwDecode[[]lwCollection](t, c.do("GET", lwV1+"/collections", ""), http.StatusOK)
	if len(collections) != 1 || collections[0].Name != "Reading" || collections[0].ParentID != nil {
		t.Fatalf("collections = %+v", collections)
	}
	tags := lwDecode[struct {
		Tags       []any  `json:"tags"`
		NextCursor *int64 `json:"nextCursor"`
	}](t, c.do("GET", lwV1+"/tags?sort=2&cursor=0&search=", ""), http.StatusOK)
	if len(tags.Tags) != 0 || tags.NextCursor != nil {
		t.Errorf("tags = %+v", tags)
	}

	// Not saved yet, then saved with the default collection, then the badge.
	check := func() int {
		rec := c.do("GET", lwV1+"/search?sort=0&searchQueryString="+urlQueryEscape("url:https://example.org/docs#intro"), "")
		return len(lwDecode[lwLinks](t, rec, http.StatusOK).Links)
	}
	if n := check(); n != 0 {
		t.Fatalf("already saved before saving: %d", n)
	}
	link := lwDecode[lwLink](t, c.do("POST", lwV1+"/links", `{"url":"https://example.org/docs/","name":"Docs","description":"",`+
		`"collection":{"name":"Unorganized"},"tags":[{"name":"dev"}]}`), http.StatusOK)
	if link.Name != "Docs" || link.CollectionID != collections[0].ID || len(link.Tags) != 1 || link.Tags[0].Name != "dev" {
		t.Errorf("created = %+v", link)
	}
	if n := check(); n != 1 {
		t.Errorf("already saved after saving: %d", n)
	}

	// Screenshot upload after saving: it becomes the link's preview.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	s.previews.Run(ctx)
	screenshot := testScreenshot(t, 1600, 900)
	rec := c.upload(link.ID, "0", screenshot)
	if got := lwDecode[lwLink](t, rec, http.StatusOK); got.ID != link.ID {
		t.Errorf("upload response = %+v", got)
	}
	preview := lwV1 + "/archives/" + strconv.FormatInt(link.ID, 10) + "?format=0&preview=true"
	deadline := time.Now().Add(5 * time.Second)
	for {
		rec := c.do("GET", preview, "")
		if rec.Code == http.StatusOK && rec.Header().Get("Content-Type") == "image/jpeg" {
			img, _, err := image.DecodeConfig(rec.Body)
			if err != nil || img.Width != 800 || img.Height != 450 {
				t.Fatalf("preview = %+v, %v; want an 800×450 JPEG", img, err)
			}
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("the screenshot never became the preview: %d %v", rec.Code, rec.Header())
		}
		time.Sleep(20 * time.Millisecond)
	}
	if got := lwDecode[lwLink](t, c.do("GET", lwV1+"/links/"+strconv.FormatInt(link.ID, 10), ""), http.StatusOK); got.Preview == "unavailable" {
		t.Errorf("link preview = %q after the upload", got.Preview)
	}
}

// testScreenshot returns a w×h PNG.
func testScreenshot(t *testing.T, w, h int) []byte {
	t.Helper()
	var b bytes.Buffer
	if err := png.Encode(&b, image.NewRGBA(image.Rect(0, 0, w, h))); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}

// upload sends a file to the archive endpoint as the browser extension does.
func (c linkwardenClient) upload(linkID int64, format string, data []byte) *httptest.ResponseRecorder {
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	part, _ := mw.CreateFormFile("file", "screenshot.png")
	_, _ = part.Write(data)
	_ = mw.Close()
	r := httptest.NewRequest("POST", lwV1+"/archives/"+strconv.FormatInt(linkID, 10)+"?format="+format, &body)
	r.RemoteAddr = "192.0.2.1:4000"
	r.Header.Set("Content-Type", mw.FormDataContentType())
	withBearer(c.secret)(r)
	fromExtension(r)
	rec := httptest.NewRecorder()
	c.s.h.ServeHTTP(rec, r)
	return rec
}

func TestLinkwardenScreenshotUploadRefusals(t *testing.T) {
	s := newServer(t)
	ann := s.signIn(t, "ann@example.org")
	c := s.linkwardenToken(t, ann)
	link := lwDecode[lwLink](t, c.do("POST", lwV1+"/links", `{"url":"https://example.org/"}`), http.StatusOK)
	outside := s.create(t, ann, map[string]any{"kind": "link", "url": "https://example.org/outside"})
	var outsideNum int64
	if err := s.db.QueryRow(`SELECT num FROM favourite_nodes WHERE id = ?`, outside.ID).Scan(&outsideNum); err != nil {
		t.Fatal(err)
	}

	big := append(testScreenshot(t, 10, 10), make([]byte, 2<<20)...)
	for name, tt := range map[string]struct {
		id     int64
		format string
		data   []byte
		status int
		text   string
	}{
		"too large":    {link.ID, "0", big, http.StatusBadRequest, "too large"},
		"text file":    {link.ID, "0", []byte("hello"), http.StatusBadRequest, "PNG, JPEG"},
		"pdf":          {link.ID, "2", []byte("%PDF-1.4"), http.StatusBadRequest, "PDFs"},
		"out of scope": {outsideNum, "0", testScreenshot(t, 10, 10), http.StatusNotFound, "Link not found"},
	} {
		rec := c.upload(tt.id, tt.format, tt.data)
		if rec.Code != tt.status || !strings.Contains(rec.Body.String(), tt.text) {
			t.Errorf("%s: %d %s, want %d with %q", name, rec.Code, rec.Body, tt.status, tt.text)
		}
	}
	var jobs int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM jobs WHERE kind = 'preview_convert'`).Scan(&jobs); err != nil || jobs != 0 {
		t.Fatalf("convert jobs = %d, err %v; want none", jobs, err)
	}
}

func TestLinkwardenMobileFlow(t *testing.T) {
	s := newServer(t)
	ann := s.signIn(t, "ann@example.org")
	c := s.linkwardenToken(t, ann)
	later := s.create(t, ann, map[string]any{"kind": "folder", "title": "Later", "parentId": c.rootID})
	work := s.create(t, ann, map[string]any{"kind": "folder", "title": "Work"})
	outside := s.create(t, ann, map[string]any{"kind": "link", "url": "https://example.org/work", "parentId": work.ID})
	for i := range 55 {
		s.create(t, ann, map[string]any{"kind": "link", "url": "https://example.org/" + strconv.Itoa(i), "title": "L" + strconv.Itoa(i), "parentId": later.ID})
	}

	// Login: token check, config, logins, then the dashboard.
	me := lwDecode[map[string]any](t, c.do("GET", lwV1+"/users/me", ""), http.StatusOK)
	if me["id"] != float64(1) || me["email"] != "ann@example.org" || me["linksRouteTo"] != "ORIGINAL" {
		t.Errorf("me = %v", me)
	}
	if rec := s.req("GET", lwV1+"/logins", ""); rec.Code != http.StatusOK || strings.TrimSpace(rec.Body.String()) != `{"buttonAuths":[]}` {
		t.Errorf("logins: %d %s", rec.Code, rec.Body)
	}
	if rec := s.req("GET", lwV1+"/config", ""); rec.Code != http.StatusOK {
		t.Errorf("config without token: %d", rec.Code)
	}
	dash := lwDecode[struct {
		Links        []lwLink `json:"links"`
		NumberOfTags int      `json:"numberOfTags"`
	}](t, c.do("GET", linkwardenPrefix+"/api/v2/dashboard", ""), http.StatusOK)
	if len(dash.Links) != 16 || dash.Links[0].Name != "L54" {
		t.Errorf("dashboard: %d links", len(dash.Links))
	}

	// Link list with the app's cursor parameter.
	first := lwDecode[lwLinks](t, c.do("GET", lwV1+"/search?cursor=0&sort=0", ""), http.StatusOK)
	if len(first.Links) != 50 || first.NextCursor == nil {
		t.Fatalf("first page: %d, %v", len(first.Links), first.NextCursor)
	}
	second := lwDecode[lwLinks](t, c.do("GET", lwV1+"/search?cursor="+strconv.FormatInt(*first.NextCursor, 10)+"&sort=0", ""), http.StatusOK)
	if len(second.Links) != 5 || second.NextCursor != nil {
		t.Errorf("second page: %d, %v", len(second.Links), second.NextCursor)
	}

	// Edit and move: the app sends the link back with its changes.
	link := first.Links[0]
	var body map[string]any
	rec := c.do("GET", lwV1+"/links/"+strconv.FormatInt(link.ID, 10), "")
	if err := json.Unmarshal(decodeJSONBody[envelope](t, rec).Response, &body); err != nil {
		t.Fatal(err)
	}
	collections := lwDecode[[]lwCollection](t, c.do("GET", lwV1+"/collections", ""), http.StatusOK)
	root := collections[0].ID
	body["name"] = "Renamed"
	body["collection"] = map[string]any{"id": root, "ownerId": 1}
	body["tags"] = []map[string]any{{"name": "go"}}
	moved := lwDecode[lwLink](t, c.do("PUT", lwV1+"/links/"+strconv.FormatInt(link.ID, 10), mustJSON(body)), http.StatusOK)
	if moved.Name != "Renamed" || moved.CollectionID != root || len(moved.Tags) != 1 {
		t.Errorf("moved = %+v", moved)
	}
	body["pinnedBy"] = []map[string]any{{"id": 1}}
	if rec := c.do("PUT", lwV1+"/links/"+strconv.FormatInt(link.ID, 10), mustJSON(body)); rec.Code != http.StatusBadRequest {
		t.Errorf("pin: %d %s", rec.Code, rec.Body)
	}

	// New tags only exist on links.
	if rec := c.do("POST", lwV1+"/tags", `{"tags":[{"label":"later"}]}`); rec.Code != http.StatusForbidden {
		t.Errorf("create tag: %d", rec.Code)
	}
	tagID := moved.Tags[0].ID
	lwDecode[map[string]any](t, c.do("GET", lwV1+"/tags/"+strconv.FormatInt(tagID, 10), ""), http.StatusOK)
	lwDecode[map[string]any](t, c.do("DELETE", lwV1+"/tags/"+strconv.FormatInt(tagID, 10), ""), http.StatusOK)

	// Delete a collection: its links move to the parent.
	var laterNum int64
	for _, col := range collections {
		if col.Name == "Later" {
			laterNum = col.ID
		}
	}
	lwDecode[lwCollection](t, c.do("DELETE", lwV1+"/collections/"+strconv.FormatInt(laterNum, 10), ""), http.StatusOK)
	all := lwDecode[lwLinks](t, c.do("GET", lwV1+"/search?collectionId="+strconv.FormatInt(root, 10), ""), http.StatusOK)
	if len(all.Links) != 50 || all.NextCursor == nil {
		t.Errorf("links in root after delete: %d", len(all.Links))
	}
	if rec := c.do("DELETE", lwV1+"/collections/"+strconv.FormatInt(root, 10), ""); rec.Code != http.StatusForbidden {
		t.Errorf("delete root collection: %d", rec.Code)
	}

	// A link outside the subtree is not found.
	var outsideNum int64
	if err := s.db.QueryRow(`SELECT num FROM favourite_nodes WHERE id = ?`, outside.ID).Scan(&outsideNum); err != nil {
		t.Fatal(err)
	}
	rec = c.do("GET", lwV1+"/links/"+strconv.FormatInt(outsideNum, 10), "")
	if rec.Code != http.StatusNotFound || strings.TrimSpace(rec.Body.String()) != `{"response":"Link not found."}` {
		t.Errorf("outside: %d %s", rec.Code, rec.Body)
	}

	// Highlights and other unsupported endpoints are not found.
	for _, path := range []string{lwV1 + "/links/" + strconv.FormatInt(moved.ID, 10) + "/highlights", lwV1 + "/rss", linkwardenPrefix + "/nope"} {
		if rec := c.do("GET", path, ""); rec.Code != http.StatusNotFound || strings.TrimSpace(rec.Body.String()) != `{"response":"Not found."}` {
			t.Errorf("GET %s: %d %s", path, rec.Code, rec.Body)
		}
	}
	if rec := c.do("PUT", lwV1+"/collections/1", `{}`); rec.Code != http.StatusNotFound {
		t.Errorf("PUT collection: %d", rec.Code)
	}
}

func TestLinkwardenAuth(t *testing.T) {
	s := newServer(t)
	ann := s.signIn(t, "ann@example.org")
	karakeep := s.createToken(t, ann, nil)
	const unauthorized = `{"response":"You must be logged in."}`
	for name, opts := range map[string][]func(*http.Request){
		"no token":       nil,
		"cookie only":    {withCookie(ann)},
		"karakeep token": {withBearer(karakeep.Secret)},
		"unknown token":  {withBearer("bdly_AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA")},
	} {
		rec := s.req("GET", lwV1+"/users/me", "", opts...)
		if rec.Code != http.StatusUnauthorized || strings.TrimSpace(rec.Body.String()) != unauthorized {
			t.Errorf("%s: %d %s", name, rec.Code, rec.Body)
		}
	}
	rec := s.req("POST", lwV1+"/session", `{"username":"ann","password":"x"}`)
	if rec.Code != http.StatusUnauthorized || !strings.Contains(rec.Body.String(), "access token") {
		t.Errorf("session: %d %s", rec.Code, rec.Body)
	}

	// A Linkwarden token doesn't work for Karakeep.
	c := s.linkwardenToken(t, ann)
	if rec := s.req("GET", karakeepPrefix+"/api/trpc/users.whoami?batch=1&input={}", "", withBearer(c.secret)); rec.Code != http.StatusUnauthorized {
		t.Errorf("Linkwarden token on Karakeep: %d", rec.Code)
	}

	// Preflight for an update.
	rec = s.req("OPTIONS", lwV1+"/links/7", "", fromExtension, func(r *http.Request) {
		r.Header.Set("Access-Control-Request-Method", "PUT")
		r.Header.Set("Access-Control-Request-Headers", "authorization,content-type")
	})
	if rec.Code != http.StatusNoContent || rec.Header().Get("Access-Control-Allow-Methods") != "GET, POST, PUT, DELETE" ||
		rec.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Errorf("preflight: %d %v", rec.Code, rec.Header())
	}
}

func TestLinkwardenPreview(t *testing.T) {
	s := newServer(t)
	ann := s.signIn(t, "ann@example.org")
	c := s.linkwardenToken(t, ann)
	png := append([]byte("\x89PNG\r\n\x1a\n"), make([]byte, 32)...)
	img, err := s.images.Save(context.Background(), images.SourceFavicon, "", png)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.db.Exec(`INSERT INTO favicons (origin, image_id, fetched_at) VALUES ('https://go.dev', ?, '2026-03-01T12:00:00.000Z')`, img.ID); err != nil {
		t.Fatal(err)
	}
	withIcon := lwDecode[lwLink](t, c.do("POST", lwV1+"/links", `{"url":"https://go.dev/tour"}`), http.StatusOK)
	without := lwDecode[lwLink](t, c.do("POST", lwV1+"/links", `{"url":"https://example.org"}`), http.StatusOK)
	if withIcon.Preview == "unavailable" || without.Preview != "unavailable" {
		t.Errorf("previews = %q, %q", withIcon.Preview, without.Preview)
	}

	path := lwV1 + "/archives/" + strconv.FormatInt(withIcon.ID, 10)
	rec := c.do("GET", path+"?format=1&preview=true&updatedAt=2026-03-01", "")
	if rec.Code != http.StatusOK || rec.Header().Get("Content-Type") != "image/png" || !bytes.Equal(rec.Body.Bytes(), png) {
		t.Errorf("preview: %d %v", rec.Code, rec.Header())
	}
	for _, q := range []string{"?format=3", "?format=1", "?format=2&preview=true"} {
		if rec := c.do("GET", path+q, ""); rec.Code != http.StatusNotFound {
			t.Errorf("%s: %d", q, rec.Code)
		}
	}
	if rec := c.do("GET", lwV1+"/archives/"+strconv.FormatInt(without.ID, 10)+"?format=1&preview=true", ""); rec.Code != http.StatusNotFound {
		t.Errorf("without favicon: %d", rec.Code)
	}
}

func urlQueryEscape(s string) string {
	return strings.NewReplacer("%", "%25", "#", "%23", ":", "%3A", "/", "%2F", " ", "%20").Replace(s)
}
