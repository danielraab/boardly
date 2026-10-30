package httpapi

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"boardly/internal/notes"
)

const karakeepAssets = karakeepPrefix + "/api/assets"

// uploadAsset sends data as the "file" part of a multipart upload, as the
// Karakeep mobile app does.
func (s *server) uploadAsset(field, name string, data []byte, opts ...func(*http.Request)) *httptest.ResponseRecorder {
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	part, _ := mw.CreateFormFile(field, name)
	part.Write(data)
	mw.Close()
	r := httptest.NewRequest("POST", karakeepAssets, &body)
	r.RemoteAddr = "192.0.2.1:4000"
	r.Header.Set("Content-Type", mw.FormDataContentType())
	for _, o := range opts {
		o(r)
	}
	rec := httptest.NewRecorder()
	s.h.ServeHTTP(rec, r)
	return rec
}

type uploadedAsset struct {
	AssetID     string `json:"assetId"`
	ContentType string `json:"contentType"`
	Size        int    `json:"size"`
	FileName    string `json:"fileName"`
}

func wantRESTError(t *testing.T, rec *httptest.ResponseRecorder, status int) {
	t.Helper()
	var e struct{ Error string }
	if err := json.Unmarshal(rec.Body.Bytes(), &e); err != nil || rec.Code != status || e.Error == "" {
		t.Errorf("error = %d %s, want %d with an error message", rec.Code, rec.Body, status)
	}
}

func TestKarakeepAssets(t *testing.T) {
	s := newServer(t)
	ann := s.signIn(t, "ann@example.org")
	bob := s.signIn(t, "bob@example.org")
	inbox := s.noteCategory(t, ann, "Inbox", "")
	work := s.noteCategory(t, ann, "Work", "")
	folder := s.create(t, ann, map[string]any{"kind": "folder", "title": "Reading"})
	rec := s.req("POST", "/api/app-tokens", mustJSON(map[string]any{
		"name": "Phone", "app": "karakeep", "rootId": folder.ID, "rootCategoryId": inbox,
	}), withCookie(ann))
	bearer := withBearer(decodeJSONBody[createdToken](t, rec).Secret)

	wantRESTError(t, s.uploadAsset("file", "a.png", testPNG), http.StatusUnauthorized)

	// Share a photo.
	rec = s.uploadAsset("file", "IMG_0042.png", testPNG, bearer)
	photo := decodeJSONBody[uploadedAsset](t, rec)
	if rec.Code != http.StatusOK || photo.AssetID == "" || photo.ContentType != "image/png" || photo.Size != len(testPNG) || photo.FileName != "IMG_0042.png" {
		t.Fatalf("upload: %d %s", rec.Code, rec.Body)
	}
	// A file just uploaded can be loaded before it is in a note.
	if rec := s.req("GET", karakeepAssets+"/"+photo.AssetID, "", bearer); rec.Code != http.StatusOK || !bytes.Equal(rec.Body.Bytes(), testPNG) {
		t.Errorf("unused upload: %d", rec.Code)
	}

	// Failed uploads.
	stored := func() int {
		return len(decodeJSONBody[struct{ Attachments []any }](t, s.req("GET", "/api/attachments", "", withCookie(ann))).Attachments)
	}
	before := stored()
	wantRESTError(t, s.uploadAsset("file", "big.pdf", append([]byte("%PDF-"), make([]byte, 2<<20)...), bearer), http.StatusRequestEntityTooLarge)
	wantRESTError(t, s.uploadAsset("file", "x.svg", []byte("<svg/>"), bearer), http.StatusBadRequest)
	wantRESTError(t, s.uploadAsset("other", "a.png", testPNG, bearer), http.StatusBadRequest)
	wantRESTError(t, s.req("POST", karakeepAssets, `{}`, bearer), http.StatusBadRequest)
	if after := stored(); after != before {
		t.Errorf("failed uploads stored %d files", after-before)
	}

	// Save a file: a note in the root category, shown as an asset bookmark.
	rec = s.trpcMutation("bookmarks.createBookmark", input(map[string]any{"type": "asset", "assetId": photo.AssetID, "assetType": "image", "source": "mobile"}), bearer)
	data, _ := trpcData(t, rec)
	var created struct {
		ID      string
		Title   string
		Content struct{ Type, AssetType, AssetID, FileName string }
		Assets  []struct{ ID, AssetType string }
	}
	_ = json.Unmarshal(data, &created)
	if created.Title != "IMG_0042.png" || created.Content.Type != "asset" || created.Content.AssetType != "image" ||
		created.Content.AssetID != photo.AssetID || len(created.Assets) != 1 || created.Assets[0].AssetType != "bookmarkAsset" {
		t.Fatalf("created = %s", data)
	}
	n := decodeJSONBody[notes.Note](t, s.req("GET", "/api/notes/"+created.ID, "", withCookie(ann)))
	if n.CategoryID == nil || *n.CategoryID != inbox || len(n.Attachments) != 1 || n.Attachments[0].ID != photo.AssetID {
		t.Errorf("note = %+v", n)
	}
	if rec := s.req("GET", karakeepAssets+"/"+photo.AssetID, "", bearer); rec.Code != http.StatusOK || rec.Header().Get("Content-Type") != "image/png" ||
		!strings.HasPrefix(rec.Header().Get("Content-Disposition"), "inline") || rec.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Errorf("view: %d %v", rec.Code, rec.Header())
	}
	rec = s.trpcMutation("bookmarks.createBookmark", input(map[string]any{"type": "asset", "assetId": "unknown", "assetType": "image"}), bearer)
	if rec.Code != http.StatusBadRequest || trpcErrorCode(t, rec) != "BAD_REQUEST" {
		t.Errorf("unknown file: %d %s", rec.Code, rec.Body)
	}

	// Files outside the scope, of another user, or unknown are not found.
	budget := s.mustUploadAttachment(t, ann, "budget.pdf", testPDF)
	if rec := s.req("POST", "/api/notes", `{"categoryId":"`+work+`","content":{"v":1,"blocks":[{"type":"attachment","id":"`+budget.ID+`"}]}}`, withCookie(ann)); rec.Code != http.StatusCreated {
		t.Fatalf("work note: %d %s", rec.Code, rec.Body)
	}
	bobs := s.mustUploadAttachment(t, bob, "bob.pdf", testPDF)
	for _, id := range []string{budget.ID, bobs.ID, "unknown"} {
		wantRESTError(t, s.req("GET", karakeepAssets+"/"+id, "", bearer), http.StatusNotFound)
	}
	wantRESTError(t, s.req("GET", karakeepAssets+"/"+photo.AssetID, ""), http.StatusUnauthorized)
}
