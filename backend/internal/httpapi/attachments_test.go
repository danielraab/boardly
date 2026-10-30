package httpapi

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"boardly/internal/attachments"
	"boardly/internal/notes"
)

var testPDF = []byte("%PDF-1.7\n" + strings.Repeat("x", 300))

// uploadAttachment posts data as the raw body of an attachment named name.
func (s *server) uploadAttachment(name string, data []byte, opts ...func(*http.Request)) *httptest.ResponseRecorder {
	r := httptest.NewRequest("POST", "/api/attachments?name="+url.QueryEscape(name), bytes.NewReader(data))
	r.RemoteAddr = "192.0.2.1:4000"
	r.Header.Set("Content-Type", "application/octet-stream")
	r.Header.Set("Origin", testOrigin)
	for _, o := range opts {
		o(r)
	}
	rec := httptest.NewRecorder()
	s.h.ServeHTTP(rec, r)
	return rec
}

func (s *server) mustUploadAttachment(t *testing.T, cookie, name string, data []byte) attachments.Attachment {
	t.Helper()
	rec := s.uploadAttachment(name, data, withCookie(cookie))
	if rec.Code != http.StatusCreated {
		t.Fatalf("upload %s: %d %s", name, rec.Code, rec.Body)
	}
	return decodeJSONBody[attachments.Attachment](t, rec)
}

func TestAttachmentsRequireSignIn(t *testing.T) {
	s := newServer(t)
	for _, tt := range []struct{ method, path, body string }{
		{"GET", "/api/attachments", ""},
		{"PATCH", "/api/attachments/x", `{"keep":true}`},
		{"DELETE", "/api/attachments/x", ""},
	} {
		if rec := s.req(tt.method, tt.path, tt.body); rec.Code != http.StatusUnauthorized {
			t.Errorf("%s %s = %d, want 401", tt.method, tt.path, rec.Code)
		}
	}
	if rec := s.uploadAttachment("a.pdf", testPDF); rec.Code != http.StatusUnauthorized {
		t.Errorf("upload = %d, want 401", rec.Code)
	}
}

func TestUploadAttachments(t *testing.T) {
	s := newServer(t)
	ann := s.signIn(t, "ann@example.org")

	a := s.mustUploadAttachment(t, ann, "report.pdf", testPDF)
	if a.Name != "report.pdf" || a.ContentType != "application/pdf" || a.Size != int64(len(testPDF)) ||
		a.Keep || a.DeletedAt != nil || a.URL != "/api/attachments/"+a.ID+"/report.pdf" {
		t.Fatalf("uploaded %+v", a)
	}

	for _, tt := range []struct {
		name   string
		data   []byte
		status int
		code   string
	}{
		{"big.pdf", append([]byte("%PDF-"), make([]byte, 2<<20)...), http.StatusRequestEntityTooLarge, "attachment_too_large"},
		{"notes.zip", []byte("PK\x03\x04rest"), http.StatusBadRequest, "unsupported_attachment_type"},
		{"logo.svg", testSVG, http.StatusBadRequest, "unsupported_attachment_type"},
		{"../etc/passwd", testPDF, http.StatusBadRequest, "invalid_attachment_name"},
		{"", testPDF, http.StatusBadRequest, "invalid_attachment_name"},
	} {
		rec := s.uploadAttachment(tt.name, tt.data, withCookie(ann))
		if rec.Code != tt.status || errorCode(t, rec) != tt.code {
			t.Errorf("upload %q: %d %s; want %d %s", tt.name, rec.Code, rec.Body, tt.status, tt.code)
		}
	}
	rec := s.req("GET", "/api/attachments", "", withCookie(ann))
	if n := strings.Count(rec.Body.String(), `"id":`); n != 1 {
		t.Fatalf("stored %d attachments, want only report.pdf: %s", n, rec.Body)
	}
}

func TestServeAttachments(t *testing.T) {
	s := newServer(t)
	ann := s.signIn(t, "ann@example.org")
	pdf := s.mustUploadAttachment(t, ann, "Bericht ä.pdf", testPDF)

	// Anyone can load it, without a session, under any name.
	for _, path := range []string{pdf.URL, "/api/attachments/" + pdf.ID, "/api/attachments/" + pdf.ID + "/other.pdf"} {
		rec := s.req("GET", path, "")
		h := rec.Header()
		if rec.Code != http.StatusOK || !bytes.Equal(rec.Body.Bytes(), testPDF) || h.Get("Content-Type") != "application/pdf" {
			t.Fatalf("GET %s: %d %q", path, rec.Code, h.Get("Content-Type"))
		}
		if h.Get("X-Content-Type-Options") != "nosniff" || h.Get("Content-Security-Policy") != "sandbox" ||
			h.Get("Cache-Control") != uploadCacheControl ||
			h.Get("Content-Disposition") != `inline; filename*=utf-8''Bericht%20%C3%A4.pdf` {
			t.Fatalf("headers = %v", h)
		}
	}

	// Range requests, for seeking in videos.
	rec := s.req("GET", pdf.URL, "", func(r *http.Request) { r.Header.Set("Range", "bytes=0-3") })
	if rec.Code != http.StatusPartialContent || rec.Body.String() != "%PDF" {
		t.Fatalf("range: %d %q", rec.Code, rec.Body)
	}

	// HTML is downloaded.
	s.attachments.Allowed = append(s.attachments.Allowed, "text/html")
	html := s.mustUploadAttachment(t, ann, "page.html", []byte("<script>alert(1)</script>"))
	rec = s.req("GET", html.URL, "")
	if rec.Code != http.StatusOK || rec.Header().Get("Content-Disposition") != "attachment; filename=page.html" ||
		rec.Header().Get("Content-Security-Policy") != "sandbox" {
		t.Fatalf("html: %d %v", rec.Code, rec.Header())
	}

	// Marked attachments are still served.
	if _, err := s.db.Exec(`UPDATE attachments SET deleted_at = '2026-03-01T12:00:00.000Z' WHERE id = ?`, pdf.ID); err != nil {
		t.Fatal(err)
	}
	if rec := s.req("GET", pdf.URL, ""); rec.Code != http.StatusOK {
		t.Fatalf("marked: %d", rec.Code)
	}

	// Deleted and unknown attachments are not found.
	if rec := s.req("DELETE", "/api/attachments/"+html.ID, "", withCookie(ann)); rec.Code != http.StatusNoContent {
		t.Fatalf("delete: %d %s", rec.Code, rec.Body)
	}
	for _, path := range []string{html.URL, "/api/attachments/doesnotexist"} {
		if rec := s.req("GET", path, ""); rec.Code != http.StatusNotFound || errorCode(t, rec) != "not_found" {
			t.Errorf("GET %s: %d %s", path, rec.Code, rec.Body)
		}
	}
}

func TestAttachmentsAPI(t *testing.T) {
	s := newServer(t)
	ann := s.signIn(t, "ann@example.org")
	bob := s.signIn(t, "bob@example.org")
	report := s.mustUploadAttachment(t, ann, "report.pdf", testPDF)
	old := s.mustUploadAttachment(t, ann, "old.pdf", testPDF)

	// A note refers to report.pdf.
	rec := s.req("POST", "/api/notes", `{"title":"Taxes","content":{"v":1,"blocks":[{"type":"attachment","id":"`+report.ID+`"}]}}`, withCookie(ann))
	if rec.Code != http.StatusCreated {
		t.Fatalf("create note: %d %s", rec.Code, rec.Body)
	}
	taxes := decodeJSONBody[notes.Note](t, rec)

	rec = s.req("GET", "/api/attachments", "", withCookie(ann))
	if rec.Code != http.StatusOK {
		t.Fatalf("list: %d %s", rec.Code, rec.Body)
	}
	list := decodeJSONBody[struct {
		Attachments  []attachments.Attachment `json:"attachments"`
		MaxSize      int64                    `json:"maxSize"`
		AllowedTypes []string                 `json:"allowedTypes"`
	}](t, rec)
	if len(list.Attachments) != 2 || list.MaxSize != 1<<20 || len(list.AllowedTypes) != len(attachments.DefaultAllowed()) {
		t.Fatalf("list = %+v", list)
	}
	for _, a := range list.Attachments {
		if want := a.ID == report.ID; (len(a.NoteIDs) == 1 && a.NoteIDs[0] == taxes.ID) != want {
			t.Errorf("%s noteIds = %v", a.Name, a.NoteIDs)
		}
	}
	if rec := s.req("GET", "/api/attachments", "", withCookie(bob)); !strings.Contains(rec.Body.String(), `"attachments":[]`) {
		t.Errorf("Bob's list = %s", rec.Body)
	}

	// Keep a marked attachment, then don't keep it.
	if _, err := s.db.Exec(`UPDATE attachments SET deleted_at = '2026-03-01T12:00:00.000Z' WHERE id = ?`, old.ID); err != nil {
		t.Fatal(err)
	}
	rec = s.req("PATCH", "/api/attachments/"+old.ID, `{"keep":true}`, withCookie(ann))
	if k := decodeJSONBody[attachments.Attachment](t, rec); rec.Code != http.StatusOK || !k.Keep || k.DeletedAt == nil || k.DeleteAt != nil {
		t.Fatalf("keep: %d %s", rec.Code, rec.Body)
	}
	rec = s.req("PATCH", "/api/attachments/"+old.ID, `{"keep":false}`, withCookie(ann))
	if k := decodeJSONBody[attachments.Attachment](t, rec); rec.Code != http.StatusOK || k.Keep || k.DeletedAt != nil {
		t.Fatalf("don't keep: %d %s", rec.Code, rec.Body)
	}
	if rec := s.req("PATCH", "/api/attachments/"+old.ID, `{"keep":"yes"}`, withCookie(ann)); rec.Code != http.StatusBadRequest {
		t.Errorf("keep yes: %d %s", rec.Code, rec.Body)
	}
	if rec := s.req("PATCH", "/api/attachments/"+old.ID, `{}`, withCookie(ann)); rec.Code != http.StatusBadRequest || errorCode(t, rec) != "invalid_request" {
		t.Errorf("keep missing: %d %s", rec.Code, rec.Body)
	}

	// Delete an attachment in use, another user's, and an unused one.
	if rec := s.req("DELETE", "/api/attachments/"+report.ID, "", withCookie(ann)); rec.Code != http.StatusConflict || errorCode(t, rec) != "attachment_in_use" {
		t.Fatalf("delete in use: %d %s", rec.Code, rec.Body)
	}
	for _, tt := range []struct{ method, body string }{{"DELETE", ""}, {"PATCH", `{"keep":true}`}} {
		if rec := s.req(tt.method, "/api/attachments/"+old.ID, tt.body, withCookie(bob)); rec.Code != http.StatusNotFound || errorCode(t, rec) != "not_found" {
			t.Errorf("Bob %s: %d %s", tt.method, rec.Code, rec.Body)
		}
	}
	if rec := s.req("DELETE", "/api/attachments/"+old.ID, "", withCookie(ann)); rec.Code != http.StatusNoContent {
		t.Fatalf("delete unused: %d %s", rec.Code, rec.Body)
	}
	if rec := s.req("GET", old.URL, ""); rec.Code != http.StatusNotFound {
		t.Fatalf("deleted still served: %d", rec.Code)
	}
}

func TestNotesWithAttachments(t *testing.T) {
	s := newServer(t)
	ann := s.signIn(t, "ann@example.org")
	bob := s.signIn(t, "bob@example.org")
	roses := s.mustUploadAttachment(t, ann, "roses.pdf", testPDF)
	plan := s.mustUploadAttachment(t, ann, "plan.pdf", testPDF)
	bobs := s.mustUploadAttachment(t, bob, "bob.pdf", testPDF)

	block := func(id string) string { return `{"type":"attachment","id":"` + id + `"}` }
	rec := s.req("POST", "/api/notes", `{"content":{"v":1,"blocks":[`+block(roses.ID)+`,`+block(plan.ID)+`,`+block(roses.ID)+`]}}`, withCookie(ann))
	if rec.Code != http.StatusCreated {
		t.Fatalf("create: %d %s", rec.Code, rec.Body)
	}
	n := decodeJSONBody[notes.Note](t, rec)
	rec = s.req("GET", "/api/notes/"+n.ID, "", withCookie(ann))
	got := decodeJSONBody[notes.Note](t, rec)
	if len(got.Attachments) != 2 || got.Attachments[0].ID != roses.ID || got.Attachments[1].ID != plan.ID ||
		got.Attachments[1].URL != plan.URL || got.Attachments[1].ContentType != "application/pdf" {
		t.Fatalf("attachments = %+v", got.Attachments)
	}

	// Another user's attachment is an invalid document.
	rec = s.req("PATCH", "/api/notes/"+n.ID, `{"version":1,"content":{"v":1,"blocks":[`+block(bobs.ID)+`]}}`, withCookie(ann))
	if rec.Code != http.StatusBadRequest || errorCode(t, rec) != "invalid_document" {
		t.Fatalf("Bob's attachment: %d %s", rec.Code, rec.Body)
	}
}

func TestMemosKeepAttachments(t *testing.T) {
	s := newServer(t)
	ann := s.signIn(t, "ann@example.org")
	c := s.memosToken(t, ann, "")
	report := s.mustUploadAttachment(t, ann, "report.pdf", testPDF)

	rec := s.req("POST", "/api/notes", `{"title":"Taxes","content":{"v":1,"blocks":[{"type":"attachment","id":"`+report.ID+`"},{"type":"paragraph","content":[{"type":"text","text":"due"}]}]}}`, withCookie(ann))
	if rec.Code != http.StatusCreated {
		t.Fatalf("create: %d %s", rec.Code, rec.Body)
	}
	n := decodeJSONBody[notes.Note](t, rec)

	// A file with text after it is in the memo's content, as a link to its
	// URL, and not among its attachments.
	memo := c.list(t, "").Memos[0]
	link := "[report.pdf](" + testOrigin + report.URL + ")"
	if !strings.Contains(memo.Content, link) || len(memo.Attachments) != 0 {
		t.Fatalf("memo = %q with %d attachments, want the content to contain %q", memo.Content, len(memo.Attachments), link)
	}

	// Editing the text in the app keeps the attachment block.
	edited := strings.Replace(memo.Content, "due", "due in May", 1)
	if rec := c.do("PATCH", memosV1+"/"+memo.Name, mustJSON(map[string]any{"content": edited})); rec.Code != http.StatusOK {
		t.Fatalf("update: %d %s", rec.Code, rec.Body)
	}
	after := c.note(t, memo)
	if after.Version != n.Version+1 || len(after.Attachments) != 1 || after.Attachments[0].ID != report.ID ||
		!strings.Contains(mustJSON(after.Content), `{"type":"attachment","id":"`+report.ID+`"}`) {
		t.Fatalf("after update: %+v", after)
	}
}
